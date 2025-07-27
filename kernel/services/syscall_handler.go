package services

import (
	"fmt"
	"log/slog"

	kernelModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// handleBlockingSyscall es el punto de entrada para las syscalls que vienen de la CPU y requieren bloquear al proceso.
func handleBlockingSyscall(result kernelModels.PCBExecuteRequest, pcb *kernelModels.PCB) {
	syscallType := result.SyscallRequest.Type

	slog.Info(fmt.Sprintf("## (%d) Solicitó syscall bloqueante: %s", pcb.PID, syscallType))
	switch syscallType {
	case "DUMP_MEMORY":
		executeDumpMemorySyscall(pcb)

	case "IO":
		executeIOSyscall(pcb, result.SyscallRequest)

	default:
		slog.Error("Syscall bloqueante desconocida. Finalizando proceso por seguridad.", "tipo", syscallType, "PID", pcb.PID)
		TransitionProcessState(pcb, kernelModels.EstadoExit)
		StartLongTermScheduler()
	}
}
