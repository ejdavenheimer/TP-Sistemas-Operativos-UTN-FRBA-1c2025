package services

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"log/slog"
	"time"
)

// Planificador de corto plazo
func ShortTermScheduler() {
	for {
		//if SchedulerState != models.EstadoPlanificadorActivo {
		//	time.Sleep(500 * time.Millisecond)
		//	continue
		//}

		if models.QueueReady.Size() == 0 {
			time.Sleep(500 * time.Millisecond) //TODO: es necesario?
			continue
		}

		process, _ := models.QueueReady.Get(0)

		if process.PID == 0 {
			time.Sleep(500 * time.Millisecond) //TODO: es necesario?
			continue
		}

		switch models.KernelConfig.SchedulerAlgorithm {
		case "FIFO":
			shortScheduleFIFO()
		default:
			slog.Warn("Algoritmo no reconocido, utilizando FIFO por defecto")
			shortScheduleFIFO() //TODO: que pasa en este caso? tiene que ejecutar FIFO?
		}

		time.Sleep(500 * time.Millisecond) //TODO: es necesario?
	}
}

func shortScheduleFIFO() {
	process, err := models.QueueReady.Dequeue() // Elimina el primer proceso de la cola READY

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return
	}

	process.EstadoActual = models.EstadoExecuting
	slog.Debug(fmt.Sprintf("Proceso PID=%d pasa a estado EXECUTING", process.PID))
	models.QueueExec.Add(process)

	SelectToExecute(process)

	slog.Info("Proceso movido a EXEC", "PID", process.PID)
}
