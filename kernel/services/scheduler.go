package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

// Estado del planificador
var SchedulerState models.EstadoPlanificador = models.EstadoPlanificadorDetenido

func StartScheduler() {
	slog.Info("El planificador está en estado DETENIDO. Presione Enter para iniciar.")
	fmt.Scanln()
	SchedulerState = models.EstadoPlanificadorActivo
	slog.Info("Planificador iniciado.")
	go longTermScheduler()
}

// Planificador de largo plazo
func longTermScheduler() {
	for {
		if SchedulerState != models.EstadoPlanificadorActivo {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		//0. Finalizar procesos pendientes en EXIT
		if models.QueueExit.Size() > 0 {
			FinishProcess()
			continue
		}

		// 1. Procesos suspendidos listos tienen prioridad
		if models.QueueSuspReady.Size() > 0 {
			//	pcb, _ := models.QueueSuspReady.Get(0)
			//	process := &pcb
			//	admitProcess(process, models.QueueSuspReady, "SUSP_READY")
			time.Sleep(500 * time.Millisecond)
			continue
		}

		//2. Si no hay nada en NEW espera
		if models.QueueNew.Size() == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		//3. Caso especial: si hay un proceso en NEW se lo admite directamente
		if models.QueueNew.Size() == 1 {
			pcb, _ := models.QueueNew.Get(0)
			process := &pcb
			admitProcess(process, models.QueueNew)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 4. Planificación normal (algoritmo configurado)
		runScheduler()
	}
}

func admitProcess(process *models.PCB, fromQueue *list.ArrayList[models.PCB]) {
	err := requestMemorySpace(process.PID, process.Size, process.PseudocodePath)
	if err != nil {
		slog.Warn("Memoria insuficiente para proceso", "PID", process.PID)
		return
	}
	index := findProcessIndexByPID(fromQueue, process.PID)
	if index != -1 {
		fromQueue.Remove(index)
	}
	TransitionState(process, process.EstadoActual, models.EstadoReady)
	AddProcessToReady(process)

	//log obligatorio
	slog.Info(fmt.Sprintf("## PID %d Pasa del estado NEW al estado %s", process.PID, process.EstadoActual))
	select {
	case models.NotifyReady <- 1:
	default:
	}

}

func findProcessIndexByPID(queue *list.ArrayList[models.PCB], pid uint) int {
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
	process := &pcb
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
	processPtr := &process
	admitProcess(processPtr, models.QueueNew)
}
