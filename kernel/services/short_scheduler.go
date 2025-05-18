package services

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// Planificador de corto plazo
func ShortTermScheduler() {
	for {
		if models.QueueReady.Size() == 0 && models.QueueBlocked.Size() == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		switch models.KernelConfig.SchedulerAlgorithm {
		case "FIFO":
			shortScheduleFIFO()
		case "SJFSinDesalojo":
			sortByShortestTimeBurst()
			shortScheduleSJFSinDesalojo()
		case "SJFConDesalojo":
			sortByShortestTimeBurst()
			shortScheduleSJFConDesalojo()
		default:
			slog.Warn("Algoritmo no reconocido, utilizando FIFO por defecto")
			shortScheduleFIFO()
		}

		time.Sleep(500 * time.Millisecond)
	}
}

func shortScheduleFIFO() {
	slog.Debug("Intentando seleccionar CPU libre para ejecutar proceso...")

	//VER CPU CONECTADA
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if !ok {
		slog.Info("No hay CPU libre.")
		return
	}

	//Cambiar a estado EXEC
	pcb, err := models.QueueReady.Dequeue()
	if err != nil {
		slog.Warn("No se pudo obtener un proceso de la cola READY")
		cpu.IsFree = true
		models.ConnectedCpuMap.Set(strconv.Itoa(cpu.Id), cpu)
		return
	}

	pcb.EstadoActual = models.EstadoExecuting
	models.QueueExec.Add(pcb)
	slog.Info(fmt.Sprintf("Proceso PID=%d pasa a estado EXECUTING", pcb.PID))

	// Actualiza los datos para saber dónde se está ejecutando cada proceso
	cpu.PIDExecuting = pcb.PID
	key := strconv.Itoa(cpu.Id)

	slog.Debug(fmt.Sprintf("Asignando proceso PID=%d a CPU ID=%d", pcb.PID, cpu.Id))
	models.ConnectedCpuMap.Set(key, cpu)

	ExecuteProcess(pcb, cpu)
}

func shortScheduleSJFSinDesalojo() {
	slog.Debug("Intentando seleccionar CPU libre para ejecutar proceso...")

	//VER CPU CONECTADA
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if !ok {
		slog.Info("No hay CPU libre.")
		return
	}

	//Cambiar a estado EXEC
	pcb, err := models.QueueReady.Dequeue()
	if err != nil {
		slog.Warn("No se pudo obtener un proceso de la cola READY")
		cpu.IsFree = true
		models.ConnectedCpuMap.Set(strconv.Itoa(cpu.Id), cpu)
		return
	}

	pcb.EstadoActual = models.EstadoExecuting
	models.QueueExec.Add(pcb)
	slog.Info(fmt.Sprintf("Proceso PID=%d pasa a estado EXECUTING", pcb.PID))

	// Actualiza los datos para saber dónde se está ejecutando cada proceso
	cpu.PIDExecuting = pcb.PID
	key := strconv.Itoa(cpu.Id)

	slog.Debug(fmt.Sprintf("Asignando proceso PID=%d a CPU ID=%d", pcb.PID, cpu.Id))
	models.ConnectedCpuMap.Set(key, cpu)

	ExecuteProcess(pcb, cpu)
}

func sortByShortestTimeBurst() {
	models.QueueReady.Sort(func(a, b models.PCB) bool {
		return a.Rafaga < b.Rafaga
	})
}

func shortScheduleSJFConDesalojo() {
	pcb, err := models.QueueReady.Dequeue()

	slog.Debug("Intentando seleccionar CPU libre para ejecutar proceso...")

	//VER CPU CONECTADA
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if !ok {
		slog.Info("No hay CPU libre, pero chequeo si puedo desalojar")
		InterruptExec(pcb)
		return
	}

	//Cambiar a estado EXEC
	if err != nil {
		slog.Warn("No se pudo obtener un proceso de la cola READY")
		cpu.IsFree = true
		models.ConnectedCpuMap.Set(strconv.Itoa(cpu.Id), cpu)
		return
	}

	pcb.EstadoActual = models.EstadoExecuting
	models.QueueExec.Add(pcb)
	slog.Info(fmt.Sprintf("Proceso PID=%d pasa a estado EXECUTING", pcb.PID))

	// Actualiza los datos para saber dónde se está ejecutando cada proceso
	cpu.PIDExecuting = pcb.PID
	key := strconv.Itoa(cpu.Id)

	// Activar solo cuando se necesite el algoritmo SJF
	cpu.PIDRafaga = pcb.Rafaga

	slog.Debug(fmt.Sprintf("Asignando proceso PID=%d a CPU ID=%d", pcb.PID, cpu.Id))
	models.ConnectedCpuMap.Set(key, cpu)

	ExecuteProcess(pcb, cpu)
}
