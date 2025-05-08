package services

import (
	"time"

	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// MoveFromSuspBlockedToSuspReady is invoked by the I/O module when
// a process in SUSP_BLOCKED finishes its I/O and should go to SUSP_READY.
func MoveFromSuspBlockedToSuspReady(pid int) {
	// Find index of PCB in QueueSuspBlocked
	_, idx, found := models.QueueSuspBlocked.Find(func(p models.PCB) bool {
		return p.PID == pid
	})
	if !found {
		slog.Error("Process not found in SUSP_BLOCKED", "pid", pid)
		return
	}

	// Remove from SUSP_BLOCKED
	pcb, _ := models.QueueSuspBlocked.Get(idx)
	models.QueueSuspBlocked.Remove(idx)

	// Move to SUSP_READY
	pcb.EstadoActual = models.EstadoSuspendidoReady
	models.QueueSuspReady.Add(pcb)

	slog.Info("Process moved to SUSP_READY", "pid", pid)
}

// StartMediumTermScheduler launches the medium-term scheduler as a goroutine.
// It periodically checks the SUSP_READY queue and tries to initialize processes
// into memory, moving them to READY when successful.
func StartMediumTermScheduler() {
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			// While there are processes waiting in SUSP_READY
			for models.QueueSuspReady.Size() > 0 {
				// Check if memory has free space
				// if !memory_client.HasFreeSpace() {
				// 	break
				// }

				// Dequeue next PCB
				pcb, err := models.QueueSuspReady.Dequeue()
				if err != nil {
					slog.Error("Error dequeuing from SUSP_READY", "error", err)
					break
				}

				// Try to initialize it in memory
				// err = memory_client.InitializeProcess(pcb)
				// if err != nil {
				// 	slog.Error("Failed to initialize process from SUSP_READY", "pid", pcb.PID, "error", err)
				// 	// Put it back at front (or end—your choice)
				// 	models.QueueSuspReady.Insert(0, pcb)
				// 	break
				// }

				// Success: move to READY
				pcb.EstadoActual = models.EstadoReady
				models.QueueReady.Add(pcb)
				slog.Info("Process moved to READY", "pid", pcb.PID)
			}
		}
	}()
}
