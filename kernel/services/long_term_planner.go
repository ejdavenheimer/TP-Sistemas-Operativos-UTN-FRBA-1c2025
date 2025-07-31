package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// StartScheduler es la función de arranque que maneja el inicio manual del planificador.
func StartScheduler() {
	slog.Info("El Planificador de Largo Plazo está DETENIDO. Presione Enter para iniciar...")
	fmt.Scanln()

	models.SchedulerState = models.EstadoPlanificadorActivo
	slog.Info("Iniciando Planificador de Largo Plazo...")

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

// longTermScheduler es el ciclo principal del planificador.
func longTermScheduler() {
	for {
		<-models.NotifyLongScheduler
		slog.Debug("Planificador de Largo Plazo activado...")

		// Bucle principal del PLP: mientras haya algo que hacer, sigue trabajando.
		for {
			// Prioridad 1: Procesar la cola de EXIT.
			if models.QueueExit.Size() > 0 {
				FinishProcess()
				continue
			}

			// Prioridad 2: La cola SUSP_READY tiene prioridad sobre NEW.
			if models.QueueSuspReady.Size() > 0 {
				slog.Debug("Hay procesos en SUSP_READY. El PLP cede la prioridad y notifica al PMP.")
				// CORREGIDO: Notificamos al PMP para que actúe.
				StartMediumTermScheduler()
				break // Salimos del ciclo para que el PMP pueda trabajar.
			}

			// Prioridad 3: Procesar la cola de NEW.
			if models.QueueNew.Size() > 0 {
				admittedProcess := runNewToReadyScheduler()
				if !admittedProcess {
					break
				}
			} else {
				break
			}
		}
	}
}

// runNewToReadyScheduler elige el algoritmo y devuelve true si al menos un proceso fue admitido.
func runNewToReadyScheduler() bool {
	switch models.KernelConfig.NewAlgorithm {
	case "FIFO":
		return scheduleNewToReadyFIFO()
	case "PMCP":
		return scheduleNewToReadyPMCP()
	default:
		slog.Warn("Algoritmo de admisión no reconocido. Usando FIFO por defecto.")
		return scheduleNewToReadyFIFO()
	}
}

// scheduleNewToReadyFIFO implementa la lógica FIFO. Devuelve true si admitió un proceso.
func scheduleNewToReadyFIFO() bool {
	if models.QueueNew.Size() == 0 {
		return false
	}
	pcb, _ := models.QueueNew.Get(0)
	return admitProcess(pcb)
}

// scheduleNewToReadyPMCP implementa "Proceso más chico primero". Devuelve true si admitió al menos uno.
func scheduleNewToReadyPMCP() bool {
	if models.QueueNew.Size() == 0 {
		return false
	}
	allNewProcesses := models.QueueNew.GetAll()
	sort.Slice(allNewProcesses, func(i, j int) bool {
		return allNewProcesses[i].Size < allNewProcesses[j].Size
	})

	anyAdmitted := false
	for _, pcb := range allNewProcesses {
		if admitProcess(pcb) {
			anyAdmitted = true
		}
	}
	return anyAdmitted
}

// admitProcess contiene la lógica central. Devuelve true si el proceso fue admitido.
func admitProcess(pcb *models.PCB) bool {
	_, _, found := models.QueueNew.Find(func(p *models.PCB) bool { return p.PID == pcb.PID })
	if !found {
		return false
	}

	err := CheckUserMemoryCapacity(pcb.PID, pcb.Size)
	if err != nil {
		slog.Debug(fmt.Sprintf("Memoria no tiene espacio para PID %d (tamaño %d). Permanece en NEW.", pcb.PID, pcb.Size))
		return false
	}

	memRequest := models.MemoryRequest{
		PID:  pcb.PID,
		Size: pcb.Size,
		Path: pcb.PseudocodePath,
	}
	body, _ := json.Marshal(memRequest)

	_, err = client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/cargarpcb", body)
	if err != nil {
		slog.Error("Fallo la solicitud a Memoria para cargar el PCB.", "PID", pcb.PID, "error", err)
		return false
	}

	TransitionProcessState(pcb, models.EstadoReady)
	StartShortTermScheduler()
	return true
}
