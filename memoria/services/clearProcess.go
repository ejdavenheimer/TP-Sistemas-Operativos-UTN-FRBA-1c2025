package services

import (
	"fmt"
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

// ClearMemoryOfProcess libera todos los recursos de memoria de un proceso específico
func ClearMemoryProcess(pid uint) error {
	memoryLock.Lock()
	defer memoryLock.Unlock()

	// Verificar que el proceso existe
	process, exists := models.ProcessTable[pid]
	if !exists {
		err := fmt.Errorf("proceso PID %d no existe", pid)
		slog.Error(err.Error())
		return err
	}

	// Obtener métricas antes de eliminar el proceso
	metrics := models.ProcessMetrics[pid]

	// Verificar si el proceso está en swap y deswapearlo primero
	if _, inSwap := models.ProcessSwapTable[pid]; inSwap {
		slog.Debug("Proceso encontrado en swap, deswapeando antes de eliminar", "pid", pid)
		err := RemoveProcessInSwap(pid)
		if err != nil {
			slog.Error("Error al deswapear proceso", "pid", pid, "error", err)
			return fmt.Errorf("error al deswapear proceso PID %d: %v", pid, err)
		}
	}

	// Liberar todos los frames asignados al proceso
	framesLiberados := 0
	pageTableRoot, exists := models.PageTables[pid]
	if exists {
		framesLiberados = releaseProcessFrames(pageTableRoot, pid)
	}

	// Eliminar la tabla de páginas del proceso
	delete(models.PageTables, pid)

	// Eliminar las instrucciones del proceso
	delete(models.InstructionsMap, pid)

	// Eliminar el proceso de la tabla de procesos
	delete(models.ProcessTable, pid)

	// Eliminar las métricas del proceso
	delete(models.ProcessMetrics, pid)

	// Generar log obligatorio con las métricas
	slog.Info(fmt.Sprintf("## PID: %d - Proceso Destruido - Métricas - Acc.T.Pag: %d; Inst.Sol.: %d; SWAP: %d; Mem.Prin.: %d; Lec.Mem.: %d; Esc.Mem.: %d",
		pid, metrics.PageTableAccesses, metrics.InstructionFetches, metrics.SwapsOut, metrics.SwapsIn, metrics.Reads, metrics.Writes))

	slog.Debug("Proceso finalizado correctamente",
		"pid", pid,
		"frames_liberados", framesLiberados,
		"paginas", process.Pages)

	return nil
}

// releaseProcessFrames recorre recursivamente la tabla de páginas y libera todos los frames
func releaseProcessFrames(pageTableLevel *models.PageTableLevel, pid uint) int {
	if pageTableLevel == nil {
		return 0
	}

	framesLiberados := 0

	// Si es una hoja (último nivel), liberar el frame
	if pageTableLevel.IsLeaf && pageTableLevel.Entry != nil {
		frame := pageTableLevel.Entry.Frame
		if frame >= 0 && frame < len(models.FreeFrames) {
			models.FreeFrames[frame] = true
			framesLiberados++
			slog.Debug("Frame liberado", "pid", pid, "frame", frame)
		}
		return framesLiberados
	}

	// Si no es hoja, recorrer recursivamente todos los sub-niveles
	if pageTableLevel.SubTables != nil {
		for _, subTable := range pageTableLevel.SubTables {
			framesLiberados += releaseProcessFrames(subTable, pid)
		}
	}

	return framesLiberados
}
