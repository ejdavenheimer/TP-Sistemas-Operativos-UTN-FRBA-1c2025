package services

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

// Estado del planificador
var SchedulerState models.EstadoPlanificador = models.EstadoPlanificadorDetenido

// Inicia el planificador largo plazo (espera Enter y lanza goroutine)
func StartScheduler() {
	slog.Info("El planificador está en estado DETENIDO. Presione Enter para iniciar.")
	fmt.Scanln()
	SchedulerState = models.EstadoPlanificadorActivo
	slog.Info("Planificador iniciado.")
	go longTermScheduler()
	go ShortTermScheduler()
}

// Planificador de largo plazo
func longTermScheduler() {
	for {
		if SchedulerState != models.EstadoPlanificadorActivo {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		//0. Finalizar procesos pendientes en EXIT
		if models.QueueExit.Size() > 0{
			FinishProcess()
			continue
		}

		// 1. Procesos suspendidos listos tienen prioridad
		if models.QueueSuspReady.Size() > 0 {
			//	pcb, _ := models.QueueSuspReady.Get(0)
			//	process := &pcb
			//	admitProcess(process, models.QueueSuspReady, "SUSP_READY")
			//	time.Sleep(500 * time.Millisecond)
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
			admitProcess(process, models.QueueNew, "NEW")
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 4. Planificación normal (algoritmo configurado)
		pcb, _ := models.QueueNew.Get(0)
		process := &pcb
		admitProcess(process, models.QueueNew, "NEW")
	}
}

func admitProcess(process *models.PCB, fromQueue *list.ArrayList[models.PCB], estadoOrigen string) {
	err := requestMemorySpace(process.PID, process.Size, process.PseudocodePath)
	if err != nil {
		slog.Warn("Memoria insuficiente para proceso", "PID", process.PID)
		return
	}

	fromQueue.Remove(0)
	process.EstadoActual = models.EstadoReady
	process.UltimoCambio = time.Now()
	models.QueueReady.Add(*process)
    
	//log obligatorio
	slog.Info(fmt.Sprintf("## PID %d pasa de %s a READY", process.PID, estadoOrigen))
	runScheduler()
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

	process, _ := models.QueueNew.Get(0)
	err := requestMemorySpace(process.PID, process.Size, process.PseudocodePath)
	if err != nil {
		slog.Warn("Memoria insuficiente para proceso", "PID", process.PID)
		return
	}

	models.QueueNew.Remove(0) // Elimina el primer proceso de la cola NEW
	process.EstadoActual = models.EstadoReady
	models.QueueReady.Add(process)
	//log obligatorio
	slog.Info(fmt.Sprintf("## PID %d Pasa del estado NEW al estado %s", process.PID, process.EstadoActual))
}

func scheduleShortestFirst() {
	if models.QueueNew.Size() == 0 {
		return
	}

	var slice []models.PCB
	for i := 0; i < models.QueueNew.Size(); i++ {
		value, _ := models.QueueNew.Get(i)
		slice = append(slice, value)
	}

	// Ordenar los procesos por tamaño (ascendente)
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Size < slice[j].Size
	})

	// Verificar si hay suficiente memoria para el primer proceso en la cola NEW
	process := slice[0]
	err := requestMemorySpace(process.PID, process.Size, process.PseudocodePath)
	if err != nil {
		slog.Warn("Memoria insuficiente para proceso", "PID", process.PID)
		return
	}

	// Si hay espacio, mover a READY
	// Eliminar solo el primer proceso (más chico) de la cola NEW
	models.QueueNew.Remove(0) // Eliminar el primer proceso de la cola NEW
	process.EstadoActual = models.EstadoReady
	models.QueueReady.Add(process) // Agregarlo a la cola READY
	//log obligatorio
	slog.Info(fmt.Sprintf("## PID %d Pasa del estado NEW al estado %s", process.PID, process.EstadoActual))
}
