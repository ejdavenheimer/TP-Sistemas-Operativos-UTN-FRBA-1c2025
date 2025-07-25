package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// --- Funciones del Planificador ---

func StartMediumTermScheduler() {
	select {
	case models.NotifyMediumScheduler <- 1:
	default:
	}
}

// MediumTermScheduler es el ciclo principal del PMP. Reacciona a notificaciones.
func MediumTermScheduler() {
	for {
		<-models.NotifyMediumScheduler
		slog.Debug("PMP: Planificador de Mediano Plazo activado.")

		if models.QueueSuspReady.Size() > 0 {
			handleSuspendedReady()
		}

		if models.QueueSuspBlocked.Size() > 0 {
			handleSuspendedBlocked()
		}
	}
}

// --- Lógica de Suspensión (SWAP-OUT) ---

func handleSuspendedBlocked() {
	pcb, err := models.QueueSuspBlocked.Dequeue()
	if err != nil {
		slog.Warn("PMP: Se intentó procesar SUSP_BLOCKED pero la cola estaba vacía.")
		return
	}

	slog.Debug("PMP: Solicitando a Memoria mover proceso a SWAP.", "PID", pcb.PID)
	req := struct {
		PID uint `json:"pid"`
	}{PID: pcb.PID}
	body, _ := json.Marshal(req)

	// Usando DoRequest
	_, err = client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/swapOut", body)

	if err != nil {
		slog.Error("PMP: Error al solicitar SWAP OUT a Memoria.", "PID", pcb.PID, "error", err)
		models.QueueSuspBlocked.Add(pcb)
	} else {
		slog.Debug("PMP: Memoria confirmó SWAP OUT. Notificando al PLP.", "PID", pcb.PID)
		StartLongTermScheduler()
	}
}

// --- Lógica de Desuspensión (SWAP-IN) ---

func handleSuspendedReady() {
	switch models.KernelConfig.NewAlgorithm {
	case "FIFO":
		if models.QueueSuspReady.Size() > 0 {
			pcb, _ := models.QueueSuspReady.Get(0)
			desuspendProcess(pcb)
		}
	case "PMCP":
		allSuspended := models.QueueSuspReady.GetAll()
		sort.Slice(allSuspended, func(i, j int) bool {
			return allSuspended[i].Size < allSuspended[j].Size
		})
		for _, pcb := range allSuspended {
			desuspendProcess(pcb)
		}
	default:
		if models.QueueSuspReady.Size() > 0 {
			pcb, _ := models.QueueSuspReady.Get(0)
			desuspendProcess(pcb)
		}
	}
}

func desuspendProcess(pcb *models.PCB) {
	_, _, found := models.QueueSuspReady.Find(func(p *models.PCB) bool { return p.PID == pcb.PID })
	if !found {
		return
	}

	if err := CheckUserMemoryCapacity(pcb.PID, pcb.Size); err != nil {
		slog.Debug("PMP: No hay memoria para desuspender proceso. Permanece en SUSP_READY.", "PID", pcb.PID)
		return
	}

	requestSwapIn(pcb)
}

func requestSwapIn(pcb *models.PCB) {
	slog.Debug("PMP: Solicitando a Memoria SWAP IN.", "PID", pcb.PID)
	req := struct {
		PID uint `json:"pid"`
	}{PID: pcb.PID}
	body, _ := json.Marshal(req)

	// Usando DoRequest
	_, err := client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/swapIn", body)

	if err != nil {
		slog.Error("PMP: Error al solicitar SWAP IN a Memoria. Finalizando proceso.", "PID", pcb.PID, "error", err)
		TransitionProcessState(pcb, models.EstadoExit)
		StartLongTermScheduler()
		return
	}

	slog.Info(fmt.Sprintf("## (%d) - Pasa de SUSPENDED_READY a READY", pcb.PID))
	TransitionProcessState(pcb, models.EstadoReady)
	StartShortTermScheduler()
}
