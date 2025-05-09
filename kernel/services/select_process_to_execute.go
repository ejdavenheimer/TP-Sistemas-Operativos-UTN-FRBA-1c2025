package services

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// Esto va en el plani de corto plazo
func SelectToExecute(pcb models.PCB) {
	//VER CPU CONECTADA
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if ok {
		//Cambiar a estado EXEC
		pcb.EstadoActual = models.EstadoExecuting
		ExecuteProcess(pcb, cpu)
	} else {
		slog.Info("No hay CPU libre.")
		return
	}
}
