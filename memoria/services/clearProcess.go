package services

import (
	"fmt"
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

func ClearMemoryProcess(pid uint) error {
	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

	if _, exists := models.ProcessTable[pid]; !exists {
		// Si no está en la tabla de procesos, puede que solo esté en swap.
		if _, inSwap := models.ProcessSwapTable[pid]; !inSwap {
			err := fmt.Errorf("proceso PID %d no existe para ser eliminado", pid)
			slog.Error(err.Error())
			return err
		}
	}

	// Si está en swap, sus datos de memoria principal ya fueron liberados.
	// Solo necesitamos eliminar su entrada de la tabla de swap.
	if _, inSwap := models.ProcessSwapTable[pid]; inSwap {
		slog.Debug("Proceso encontrado en swap, eliminando entrada de swap", "pid", pid)
		delete(models.ProcessSwapTable, pid)
	}

	// Liberar frames si el proceso estaba en memoria
	models.UMemoryLock.Lock()
	if frameData, inFrames := models.ProcessFramesTable[pid]; inFrames {
		for _, frame := range frameData.Frames {
			if frame < len(models.FreeFrames) {
				models.FreeFrames[frame] = true
			}
		}
	}
	models.UMemoryLock.Unlock()

	// Obtener métricas ANTES de eliminar el proceso
	metrics, metricsExist := models.ProcessMetrics[pid]

	// Eliminar todas las estructuras de metadatos
	delete(models.PageTables, pid)
	delete(models.InstructionsMap, pid)
	delete(models.ProcessTable, pid)
	delete(models.ProcessMetrics, pid)
	delete(models.ProcessFramesTable, pid)

	if metricsExist {
		slog.Info(fmt.Sprintf("## PID: %d - Proceso Destruido - Métricas - Acc.T.Pag: %d; Inst.Sol.: %d; SWAP: %d; Mem.Prin.: %d; Lec.Mem.: %d; Esc.Mem.: %d",
			pid, metrics.PageTableAccesses, metrics.InstructionFetches, metrics.SwapsOut, metrics.SwapsIn, metrics.Reads, metrics.Writes))
	} else {
		slog.Info(fmt.Sprintf("## PID: %d - Proceso Destruido", pid))
	}

	slog.Debug("Proceso finalizado y recursos liberados correctamente", "pid", pid)

	return nil
}
