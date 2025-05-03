package services

import (
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
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
	go StartShortTermScheduler()
}

// Planificador de largo plazo
func longTermScheduler() {
	for {
		if SchedulerState != models.EstadoPlanificadorActivo {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if models.QueueNew.Size() == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		// Si la cola NEW no está vacía, procesamos el primer proceso
		process, _ := models.QueueNew.Get(0)

		// Verificar si la cola estaba vacía antes de agregar el proceso
		if models.QueueNew.Size() == 1 {
			// Solicitar memoria si la cola estaba vacía y es el primer proceso
			err := requestMemorySpace(process.PID, process.Size, process.PseudocodePath)
			if err != nil {
				slog.Warn("Memoria insuficiente para proceso", "PID", process.PID)
				continue
			}
		}

		switch models.KernelConfig.SchedulerAlgorithm {
		case "FIFO":
			scheduleFIFO()
		case "ShortFirst":
			scheduleShortestFirst()
		default:
			slog.Warn("Algoritmo no reconocido, utilizando FIFO por defecto")
			scheduleFIFO()
		}

		time.Sleep(500 * time.Millisecond)
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
	slog.Info("Proceso movido a READY", "PID", process.PID)
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
	slog.Info("Proceso movido a READY", "PID", process.PID)
}

// PLANIFICADOR DE CORTO PLAZO
func StartShortTermScheduler() {
	go func() {
		for {
			//Si el planificador no está activo, duerme medio segundo y vuelve a chequear.
			if SchedulerState != models.EstadoPlanificadorActivo {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			//Si la cola READY está vacía, no hay procesos listos para ejecutar. Así que duerme medio segundo y vuelve a revisar.
			if models.QueueReady.Size() == 0 {
				time.Sleep(500 * time.Millisecond)
				continue
			}

			//Toma el primer proceso en la cola READY (sin sacarlo todavía).
			pcbInterface, err := models.QueueReady.Get(0)
			if err != nil {
				slog.Warn("Error obteniendo proceso de READY")
				continue
			}

			//Lo saca de la cola READY. Ya está listo para ejecutarse.
			models.QueueReady.Remove(0)
			slog.Info("Planificador de corto plazo: enviando proceso a ejecutar", "PID", pcbInterface.PID)

			go ExecuteProcess(pcbInterface)
		}
	}()
}
