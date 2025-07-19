package services

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// Estado del planificador
var SchedulerState models.EstadoPlanificador = models.EstadoPlanificadorDetenido

func StartScheduler() {
	slog.Debug("El planificador está en estado DETENIDO. Presione Enter para iniciar.")
	fmt.Scanln()
	SchedulerState = models.EstadoPlanificadorActivo
	go longTermScheduler()
	StartLongTermScheduler()
}

func StartLongTermScheduler() {
	select {
	case models.NotifyLongScheduler <- 1:
	default:
	}
}

// Planificador de largo plazo
func longTermScheduler() {
	for {
		<-models.NotifyLongScheduler
		slog.Debug("Planificador de Largo Plazo activo")

		//1. Finalizar procesos pendientes en EXIT
		if models.QueueExit.Size() > 0 {
			FinishProcess()
			continue
		}

		// 2. Procesos suspendidos listos tienen prioridad
		if models.QueueSuspReady.Size() > 0 {
			NotifyToMediumScheduler()
			continue
		}

		// 3. Caso especial: uno solo en NEW
		if models.QueueNew.Size() == 1 {
			pcb, _ := models.QueueNew.Get(0)
			process := pcb
			beforeSize := models.QueueNew.Size()
			admitProcess(process, models.QueueNew)

			// Si no se pudo admitir (sigue en NEW), salimos
			if models.QueueNew.Size() == beforeSize {
				continue
			}
		}

		// 4. Planificación normal (algoritmo configurado)
		if models.QueueNew.Size() > 1 {
			runScheduler()
		}

	}
}

func admitProcess(process *models.PCB, fromQueue *list.ArrayList[*models.PCB]) {
	success, err := requestMemorySpace(process.PID, process.Size)
	if err != nil {
		slog.Error("Error al contactar Memoria", "PID", process.PID, "error", err)
		return
	}
	if !success {
		slog.Debug("Memoria insuficiente para proceso, se mantiene en NEW", "PID", process.PID)
		// No se remueve de la cola, se mantiene en NEW para reintentar luego
		return
	}
	// Hay espacio → se carga efectivamente el PCB en Memoria
	request := models.MemoryRequest{
		PID:  process.PID,
		Size: process.Size,
		Path: process.PseudocodePath,
	}

	body, err := json.Marshal(request)
	if err != nil {
		slog.Error("Error al serializar MemoryRequest", "error", err)
		return
	}

	_, err = client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/cargarpcb", body)
	if err != nil {
		slog.Error("Error enviando request a Memoria", "error", err)
		return
	}

	index := findProcessIndexByPID(fromQueue, process.PID)
	if index != -1 {
		fromQueue.Remove(index)
	}
	TransitionState(process, models.EstadoReady)
	AddProcessToReady(process)

	// Le mandamos una señal al PCP que notifica que hay un proceso en ready, si ya tiene la señal en 1 no hacemos nada.
	//select {
	//case models.NotifyReady <- 1:
	//default:
	//}
	NotifyToReady()
}

func findProcessIndexByPID(queue *list.ArrayList[*models.PCB], pid uint) int {
	for i := 0; i < queue.Size(); i++ {
		p, _ := queue.Get(i)
		if p.PID == pid {
			return i
		}
	}
	return -1 // no encontrado
}

func runScheduler() {
	switch models.KernelConfig.NewAlgorithm {
	case "FIFO":
		scheduleFIFO()
	case "PMCP":
		scheduleShortestFirst()
	default:
		slog.Warn("Algoritmo no reconocido, utilizando FIFO por defecto")
		scheduleFIFO()
	}
}

func scheduleFIFO() {
	if models.QueueNew.Size() == 0 {
		return
	}

	pcb, _ := models.QueueNew.Get(0)
	process := pcb
	admitProcess(process, models.QueueNew)
}

func scheduleShortestFirst() {
	if models.QueueNew.Size() == 0 {
		return
	}

	indexMin := 0
	minProcess, _ := models.QueueNew.Get(0)

	for i := 1; i < models.QueueNew.Size(); i++ {
		proc, _ := models.QueueNew.Get(i)
		if proc.Size < minProcess.Size {
			minProcess = proc
			indexMin = i
		}
	}

	process, _ := models.QueueNew.Get(indexMin)
	processPtr := process
	admitProcess(processPtr, models.QueueNew)
}
