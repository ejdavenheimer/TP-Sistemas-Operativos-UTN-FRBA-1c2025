package services

import (
	"encoding/json"
	"log/slog"

	kernelModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// executeDumpMemorySyscall maneja la petición de DUMP de un proceso.
func executeDumpMemorySyscall(pcb *kernelModels.PCB) {
	TransitionProcessState(pcb, kernelModels.EstadoBlocked)
	slog.Debug("Proceso bloqueado por DUMP_MEMORY.", "PID", pcb.PID)

	go func() {
		slog.Debug("Enviando solicitud de DUMP a Memoria...", "PID", pcb.PID)

		request := struct {
			Pid  uint `json:"pid"`
			Size int  `json:"size"`
		}{
			Pid:  pcb.PID,
			Size: pcb.Size,
		}

		body, err := json.Marshal(request)
		if err != nil {
			slog.Error("DUMP: Error al serializar la solicitud.", "PID", pcb.PID, "error", err)
			TransitionProcessState(pcb, kernelModels.EstadoExit)
			StartLongTermScheduler()
			return
		}

		// CORREGIDO: El segundo parámetro ahora es IpMemory.
		_, err = client.DoRequest(kernelModels.KernelConfig.PortMemory, kernelModels.KernelConfig.IpMemory, "POST", "memoria/dump-memory", body)

		if err != nil {
			slog.Error("DUMP: Memoria respondió con un error. Finalizando proceso.", "PID", pcb.PID, "error", err)
			TransitionProcessState(pcb, kernelModels.EstadoExit)
			StartLongTermScheduler()
		} else {
			slog.Debug("DUMP: Operación completada con éxito. Desbloqueando proceso.", "PID", pcb.PID)
			TransitionProcessState(pcb, kernelModels.EstadoReady)
			StartShortTermScheduler()
		}
	}()
}
