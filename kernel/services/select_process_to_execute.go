package services

import (
	"fmt"
	"log/slog"
	"strconv"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// Esto va en el plani de corto plazo
func SelectToExecute(pcb models.PCB) {

	//func SelectToExecute() { ----DESCOMENTAR!!!!!!!!!!!
	slog.Debug("Intentando seleccionar CPU libre para ejecutar proceso...")

	//VER CPU CONECTADA
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if !ok {
		slog.Info("No hay CPU libre.")
		return
	}

	//Cambiar a estado EXEC
	//pcb, err := models.QueueReady.Dequeue() ----DESCOMENTAR!!!!!!!!!!!!!
	// if err != nil {
	// 	slog.Warn("No se pudo obtener un proceso de la cola READY:", err)
	// 	cpu.IsFree = true
	// 	models.ConnectedCpuMap.Set(strconv.Itoa(cpu.Id), cpu)
	// 	return
	// }

	pcb.EstadoActual = models.EstadoExecuting
	slog.Info(fmt.Sprintf("Proceso PID=%d pasa a estado EXECUTING", pcb.PID))

	// Actualiza los datos para saber dónde se está ejecutando cada proceso
	cpu.PIDExecuting = pcb.PID
	key := strconv.Itoa(cpu.Id)

	// Activar solo cuando se necesite el algoritmo SJF
	// cpu.PIDRafaga = pcb.Rafaga

	slog.Debug(fmt.Sprintf("Asignando proceso PID=%d a CPU ID=%d", pcb.PID, cpu.Id))
	models.ConnectedCpuMap.Set(key, cpu)

	ExecuteProcess(pcb, cpu)
}
