package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// StartScheduler es la función de arranque que maneja el inicio manual del planificador.
func StartScheduler() {
	slog.Info("El Planificador de Largo Plazo está DETENIDO. Presione Enter para iniciar...")
	fmt.Scanln()

	models.SchedulerState = models.EstadoPlanificadorActivo
	slog.Info("Iniciando Planificador de Largo Plazo...")

	// Llamamos a la función con el nuevo nombre.
	go longTermScheduler()
	StartLongTermScheduler()
}

// StartLongTermScheduler notifica al planificador que hay trabajo que hacer.
func StartLongTermScheduler() {
	if models.SchedulerState != models.EstadoPlanificadorActivo {
		return
	}
	select {
	case models.NotifyLongScheduler <- 1:
	default:
	}
}

// longTermScheduler es el ciclo principal del planificador (nombre actualizado).
func longTermScheduler() {
	for {
		<-models.NotifyLongScheduler
		slog.Debug("Planificador de Largo Plazo activado...")

		// Prioridad 1: Procesar la cola de EXIT para liberar recursos.
		if models.QueueExit.Size() > 0 {
			FinishProcess()
			continue
		}

		// Prioridad 2: La cola SUSP_READY debe estar vacía para admitir de NEW.
		if models.QueueSuspReady.Size() > 0 {
			slog.Debug("Hay procesos en SUSP_READY. El PLP cede la prioridad.")
			continue
		}

		// Prioridad 3: Procesar la cola de NEW.
		if models.QueueNew.Size() > 0 {
			runNewToReadyScheduler()
		}
	}
}

// runNewToReadyScheduler elige el algoritmo de admisión según la configuración.
func runNewToReadyScheduler() {
	switch models.KernelConfig.NewAlgorithm {
	case "FIFO":
		scheduleNewToReadyFIFO()
	case "PMCP":
		scheduleNewToReadyPMCP()
	default:
		slog.Warn("Algoritmo de admisión no reconocido. Usando FIFO por defecto.")
		scheduleNewToReadyFIFO()
	}
}

// scheduleNewToReadyFIFO implementa la lógica FIFO.
func scheduleNewToReadyFIFO() {
	if models.QueueNew.Size() == 0 {
		return
	}
	pcb, err := models.QueueNew.Get(0)
	if err != nil {
		slog.Error("PLP (FIFO): No se pudo obtener el proceso de la cola NEW.")
		return
	}
	slog.Debug("PLP (FIFO): Evaluando proceso", "PID", pcb.PID)
	admitProcess(pcb)
}

// scheduleNewToReadyPMCP implementa "Proceso más chico primero".
func scheduleNewToReadyPMCP() {
	if models.QueueNew.Size() == 0 {
		return
	}
	allNewProcesses := models.QueueNew.GetAll()
	sort.Slice(allNewProcesses, func(i, j int) bool {
		return allNewProcesses[i].Size < allNewProcesses[j].Size
	})

	slog.Debug("PLP (PMCP): Evaluando procesos en orden de tamaño.")
	for _, pcb := range allNewProcesses {
		admitProcess(pcb)
	}
}

// admitProcess contiene la lógica central para mover un proceso de NEW a READY.
func admitProcess(pcb *models.PCB) {
	_, _, found := models.QueueNew.Find(func(p *models.PCB) bool { return p.PID == pcb.PID })
	if !found {
		return
	}

	err := CheckUserMemoryCapacity(pcb.PID, pcb.Size)
	if err != nil {
		slog.Info(fmt.Sprintf("Memoria no tiene espacio para PID %d (tamaño %d). Permanece en NEW.", pcb.PID, pcb.Size))
		return
	}

	memRequest := models.MemoryRequest{
		PID:  pcb.PID,
		Size: pcb.Size,
		Path: pcb.PseudocodePath,
	}
	body, err := json.Marshal(memRequest)
	if err != nil {
		slog.Error("Error al serializar la solicitud para cargar PCB en memoria", "PID", pcb.PID, "error", err)
		return
	}

	url := fmt.Sprintf("http://%s:%d/memoria/cargarpcb", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil || resp.StatusCode != http.StatusOK {
		slog.Error("Fallo la solicitud a Memoria para cargar el PCB. El proceso se queda en NEW.", "PID", pcb.PID)
		if err != nil {
			slog.Error("Detalle del error", "error", err)
		}
		return
	}
	defer resp.Body.Close()

	TransitionProcessState(pcb, models.EstadoReady)

	//Notificamos al Planificador de Corto Plazo que tiene un nuevo proceso para planificar.
	StartShortTermScheduler()
}
