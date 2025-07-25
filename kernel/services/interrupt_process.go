package services

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	kernelModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// checkForPreemption se ejecuta cuando un proceso llega a READY bajo el algoritmo SRT.
func checkForPreemption(newPcb *kernelModels.PCB) {
	slog.Debug("SRT: Verificando desalojo.", "Nuevo PID", newPcb.PID, "Estimación", newPcb.RafagaEstimada)

	executingProcesses := kernelModels.QueueExec.GetAll()
	if len(executingProcesses) == 0 {
		return // No hay nadie ejecutando, no hay a quién desalojar.
	}

	var victimPcb *kernelModels.PCB = nil
	var maxRemainingTime float32 = -1

	// 1. Encontrar al proceso en ejecución con el MAYOR tiempo restante.
	for _, runningPcb := range executingProcesses {
		remainingTime := runningPcb.RafagaEstimada - runningPcb.RafagaReal
		if remainingTime > maxRemainingTime {
			maxRemainingTime = remainingTime
			victimPcb = runningPcb
		}
	}

	if victimPcb == nil {
		return // No se encontró una víctima válida.
	}

	// 2. Comparar la ráfaga del nuevo proceso con el tiempo restante de la víctima.
	if newPcb.RafagaEstimada < maxRemainingTime {
		slog.Info(fmt.Sprintf("SRT: Desalojo necesario. PID %d (est: %.2f) es más corto que el tiempo restante de PID %d (rest: %.2f).", newPcb.PID, newPcb.RafagaEstimada, victimPcb.PID, maxRemainingTime))

		cpu := kernelModels.ConnectedCpuMap.GetCPUByPid(victimPcb.PID)
		if cpu != nil {
			SendInterruption(victimPcb.PID, cpu)
		} else {
			slog.Warn("SRT: No se encontró la CPU para el proceso a desalojar.", "PID", victimPcb.PID)
		}
	}
}

// SendInterruption envía una señal de interrupción a una CPU específica.
func SendInterruption(pid uint, cpu *models.CpuN) {
	slog.Debug("Enviando interrupción a CPU.", "PID", pid, "cpu_id", cpu.Id)

	bodyRequest, err := json.Marshal(pid)
	if err != nil {
		slog.Error("Error al serializar el PID para interrupción.", "error", err)
		return
	}

	_, err = client.DoRequest(cpu.Port, cpu.Ip, "POST", "cpu/interrupt", bodyRequest)
	if err != nil {
		slog.Error("Error enviando la interrupción a la CPU.", "cpu_id", cpu.Id, "error", err)
	}
}
