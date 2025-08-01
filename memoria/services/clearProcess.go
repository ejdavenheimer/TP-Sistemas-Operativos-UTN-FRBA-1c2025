package services

import (
	"fmt"
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

func ClearMemoryProcess(pid uint) error {
	var processExists, inSwap, inFrames bool
	var frameData *models.ProcessFrames
	var metrics *models.Metrics

	models.ProcessDataLock.RLock()
	_, processExists = models.ProcessTable[pid]
	_, inSwap = models.ProcessSwapTable[pid]
	frameData, inFrames = models.ProcessFramesTable[pid]
	metrics = models.ProcessMetrics[pid]
	models.ProcessDataLock.RUnlock()

	if !processExists && !inSwap {
		err := fmt.Errorf("proceso PID %d no existe para ser eliminado", pid)
		slog.Error(err.Error())
		return err
	}

	// LIBERACIÓN DE FRAMES (sección crítica mínima)
	if inFrames && frameData != nil && len(frameData.Frames) > 0 {
		models.UMemoryLock.Lock()
		slog.Debug("UMemoryLock lockeado CLEAR PROCESS")
		// Liberación batch de frames
		for _, frame := range frameData.Frames {
			if frame >= 0 && frame < len(models.FreeFrames) {
				models.FreeFrames[frame] = true
			}
		}
		models.UMemoryLock.Unlock() // Liberar inmediatamente

		slog.Debug("Frames liberados", "pid", pid, "count", len(frameData.Frames))
	}
	models.ProcessDataLock.Lock()
	// Eliminación batch de todas las estructuras
	delete(models.ProcessSwapTable, pid)
	delete(models.PageTables, pid)
	delete(models.InstructionsMap, pid)
	delete(models.ProcessTable, pid)
	delete(models.ProcessMetrics, pid)
	delete(models.ProcessFramesTable, pid)
	models.ProcessDataLock.Unlock()

	if metrics != nil {
		slog.Info(fmt.Sprintf("## PID: %d - Proceso Destruido - Métricas - Acc.T.Pag: %d; Inst.Sol.: %d; SWAP: %d; Mem.Prin.: %d; Lec.Mem.: %d; Esc.Mem.: %d",
			pid, metrics.PageTableAccesses, metrics.InstructionFetches, metrics.SwapsOut, metrics.SwapsIn, metrics.Reads, metrics.Writes))
	} else {
		slog.Info(fmt.Sprintf("## PID: %d - Proceso Destruido", pid))
	}

	slog.Debug("Proceso finalizado y recursos liberados correctamente", "pid", pid)
	return nil
}
