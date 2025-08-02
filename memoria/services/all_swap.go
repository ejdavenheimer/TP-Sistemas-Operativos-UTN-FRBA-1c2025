package services

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

var memorySwapMutex sync.Mutex

func IsProcessInSwap(pid uint) bool {
	models.ProcessDataLock.RLock()
	defer models.ProcessDataLock.RUnlock()
	_, inSwap := models.ProcessSwapTable[pid]
	return inSwap
}

func PutProcessInSwap(pid uint) error {
	memorySwapMutex.Lock()
	defer memorySwapMutex.Unlock()

	swapDelay := time.Duration(models.MemoryConfig.SwapDelay) * time.Millisecond
	pageSize := models.MemoryConfig.PageSize

	slog.Debug("Inicia PUT PROCESS IN SWAP")
	time.Sleep(swapDelay)

	var processFrames *models.ProcessFrames
	var processExists bool
	var framesToProcess []int

	// VALIDACIÓN 1: Verificar si el proceso ya fue swapeado
	models.ProcessDataLock.Lock()
	if _, inSwap := models.ProcessSwapTable[pid]; inSwap {
		models.ProcessDataLock.Unlock()
		slog.Debug("Memoria: Proceso ya se encuentra en SWAP, operación exitosa.", "PID", pid)
		return nil
	}
	if pf, exists := models.ProcessFramesTable[pid]; exists {
		processFrames = &models.ProcessFrames{
			PID:    pf.PID,
			Frames: make([]int, len(pf.Frames)),
		}
		copy(processFrames.Frames, pf.Frames)
		framesToProcess = make([]int, len(pf.Frames))
		copy(framesToProcess, pf.Frames)
		processExists = true
	}
	models.ProcessDataLock.Unlock()

	// VALIDACIÓN 2: Verificar que el proceso existe en la tabla de frames
	if !processExists {
		err := fmt.Errorf("proceso PID %d no encontrado en tabla de frames para swapear", pid)
		slog.Error(err.Error())
		return err
	}

	slog.Debug("Memoria: Iniciando suspensión de proceso", "PID", pid)

	// CASO ESPECIAL: Proceso con 0 frames
	if len(framesToProcess) == 0 {
		models.ProcessDataLock.Lock()
		models.ProcessSwapTable[pid] = models.SwapEntry{Offset: 0, Size: 0}
		delete(models.ProcessFramesTable, pid)
		models.ProcessDataLock.Unlock()

		if err := updatePageTablePresenceBitsSafe(pid, false); err != nil {
			slog.Warn("Error actualizando bits de presencia", "PID", pid, "error", err)
		}
		IncrementMetric(pid, "swap_out")
		return nil
	}

	// VALIDACIÓN 4: Verificar integridad de frames
	if !validateFrameIntegrity(framesToProcess) {
		err := fmt.Errorf("frames del proceso PID %d tienen problemas de integridad", pid)
		slog.Error(err.Error())
		return err
	}

	file, err := os.OpenFile(models.MemoryConfig.SwapFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("no se pudo abrir swapfile: %v", err))
		return err
	}
	defer file.Close()

	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		slog.Error(fmt.Sprintf("error posicionando el cursor al final de swapfile: %v", err))
		return err
	}

	totalFrames := len(framesToProcess)
	allFramesData := make([]byte, 0, totalFrames*pageSize)

	models.UMemoryLock.Lock()
	slog.Debug("UMemoryLock lockeado SWAP IN")
	// VALIDACIÓN 5: Re-verificar que los frames siguen siendo válidos
	for _, frameIndex := range processFrames.Frames {
		if frameIndex < 0 || frameIndex >= len(models.FreeFrames) {
			models.UMemoryLock.Unlock()
			return fmt.Errorf("frame index inválido: %d para PID %d", frameIndex, pid)
		}
		start := frameIndex * pageSize
		end := start + pageSize

		if end > len(models.UserMemory) {
			models.UMemoryLock.Unlock()
			return fmt.Errorf("acceso fuera de bounds en memoria para frame %d", frameIndex)
		}

		frameData := make([]byte, pageSize)
		copy(frameData, models.UserMemory[start:end])
		allFramesData = append(allFramesData, frameData...)

		models.FreeFrames[frameIndex] = true
	}
	models.UMemoryLock.Unlock()
	n, err := file.Write(allFramesData)
	if err != nil {
		slog.Error("Memoria: Error escribiendo frames en swapfile", "error", err)
		return err
	}
	totalSize := int64(n)
	models.ProcessDataLock.Lock()
	models.ProcessSwapTable[pid] = models.SwapEntry{Offset: offset, Size: totalSize}
	delete(models.ProcessFramesTable, pid)
	models.ProcessDataLock.Unlock()

	if err := updatePageTablePresenceBitsSafe(pid, false); err != nil {
		slog.Warn("Memoria: Error actualizando bits de presencia durante SWAP OUT", "PID", pid, "error", err)
	}

	slog.Debug("SE ACTUALIZAN BITS DE PRESENCIA EN TABLAS DE PAGINA (SWAP OUT)")
	IncrementMetric(pid, "swap_in")

	slog.Info(fmt.Sprintf("PID <%d> Movido a swap - Offset: %d, Tamaño: %d", pid, offset, totalSize))
	return nil
}

func RemoveProcessInSwap(pid uint) error {
	memorySwapMutex.Lock()
	defer memorySwapMutex.Unlock()
	swapDelay := time.Duration(models.MemoryConfig.SwapDelay) * time.Millisecond
	pageSize := models.MemoryConfig.PageSize
	frameSize := int64(pageSize)
	slog.Debug("INICIA REMOVE PROCESS IN SWAP")
	time.Sleep(swapDelay)

	var swapEntry models.SwapEntry
	var swapExists bool
	var alreadyInFrames bool

	// VALIDACIÓN 1: Verificar que el proceso está en swap
	models.ProcessDataLock.Lock()
	if entry, exists := models.ProcessSwapTable[pid]; exists {
		swapEntry = models.SwapEntry{
			Offset: entry.Offset,
			Size:   entry.Size,
		}
		swapExists = true
	}

	if _, inFrames := models.ProcessFramesTable[pid]; inFrames {
		alreadyInFrames = true
	}
	models.ProcessDataLock.Unlock()

	if !swapExists {
		err := fmt.Errorf("el proceso con PID %d no se encuentra en SWAP", pid)
		slog.Error(err.Error())
		return err
	}

	// VALIDACIÓN 2: Verificar que el proceso no está ya en frames
	if alreadyInFrames {
		slog.Warn("Memoria: Proceso ya está en frames, removiendo de swap conceptualmente", "PID", pid)
		models.ProcessDataLock.Lock()
		delete(models.ProcessSwapTable, pid)
		models.ProcessDataLock.Unlock()
		return nil
	}

	// VALIDACIÓN 3: Verificar estado del proceso
	if !validateProcessExists(pid) {
		err := fmt.Errorf("proceso PID %d no existe en tabla de procesos", pid)
		slog.Error(err.Error())
		return err
	}

	// CASO ESPECIAL: Proceso con 0 frames
	if swapEntry.Size == 0 {
		models.ProcessDataLock.Lock()
		models.ProcessFramesTable[pid] = &models.ProcessFrames{PID: pid, Frames: []int{}}
		delete(models.ProcessSwapTable, pid)
		models.ProcessDataLock.Unlock()

		if err := updatePageTablePresenceBitsSafe(pid, true); err != nil {
			slog.Warn("Error actualizando bits de presencia", "PID", pid, "error", err)
		}
		return nil
	}

	framesNeeded := int(swapEntry.Size / frameSize)
	var freeFrames []int

	models.UMemoryLock.Lock()
	slog.Debug("UMemoryLock lockeado SWAP OUT - buscar frames")

	for idx, free := range models.FreeFrames {
		if free {
			freeFrames = append(freeFrames, idx)
			if len(freeFrames) == framesNeeded {
				break
			}
		}
	}

	if len(freeFrames) < framesNeeded {
		models.UMemoryLock.Unlock()
		return fmt.Errorf("no hay suficientes frames libres para des-suspender el proceso PID %d", pid)
	}

	//Reservar frames
	for _, frameIdx := range freeFrames {
		models.FreeFrames[frameIdx] = false
	}
	models.UMemoryLock.Unlock()

	file, err := os.Open(models.MemoryConfig.SwapFilePath)
	if err != nil {
		slog.Error("Memoria: Error abriendo archivo de swap", "error", err)
		// Rollback: liberar frames reservados
		models.UMemoryLock.Lock()
		slog.Debug("UMemoryLock lockeado ROLLBACK SWAP OUT")
		for _, frameIdx := range freeFrames {
			models.FreeFrames[frameIdx] = true
		}
		models.UMemoryLock.Unlock()
		return err
	}
	defer file.Close()

	processData := make([]byte, swapEntry.Size)
	_, err = file.ReadAt(processData, swapEntry.Offset)
	if err != nil {
		slog.Error("Memoria: Error leyendo contenido desde SWAP", "error", err)
		// Rollback: liberar frames reservados
		models.UMemoryLock.Lock()
		slog.Debug("UMemoryLock lockeado SWAP OUT FRAMES")
		for _, frameIdx := range freeFrames {
			models.FreeFrames[frameIdx] = true
		}
		models.UMemoryLock.Unlock()
		return err
	}

	models.UMemoryLock.Lock()
	slog.Debug("UMemoryLock lockeado FREE FRAMES")
	for i, frameIdx := range freeFrames {
		start := frameIdx * models.MemoryConfig.PageSize
		end := start + models.MemoryConfig.PageSize

		// VALIDACIÓN 6: Verificar bounds
		if end > len(models.UserMemory) {
			models.UMemoryLock.Unlock()
			// Rollback frames
			for _, frameIdx := range freeFrames {
				models.FreeFrames[frameIdx] = true
			}
			return fmt.Errorf("acceso fuera de bounds al restaurar frame %d", frameIdx)
		}

		dataStart := i * models.MemoryConfig.PageSize
		dataEnd := dataStart + models.MemoryConfig.PageSize
		copy(models.UserMemory[start:end], processData[dataStart:dataEnd])
	}
	models.UMemoryLock.Unlock()

	// VALIDACIÓN 7: Verificar tabla de páginas antes de mapear
	models.ProcessDataLock.RLock()
	_, tableExists := models.PageTables[pid]
	models.ProcessDataLock.RUnlock()
	if !tableExists {
		slog.Error("Memoria: Tabla de páginas no encontrada", "PID", pid)
		// Rollback: liberar frames
		models.UMemoryLock.Lock()
		slog.Debug("UMemoryLock lockeado ROLLBACK 2")
		for _, frameIdx := range freeFrames {
			models.FreeFrames[frameIdx] = true
		}
		models.UMemoryLock.Unlock()
		return fmt.Errorf("tabla de páginas no encontrada para PID %d", pid)
	}

	// Mapear páginas a frames de forma segura
	for pageNumber := 0; pageNumber < len(freeFrames); pageNumber++ {
		frame := freeFrames[pageNumber]
		slog.Debug("Memoria: Mapeando página a frame", "PID", pid, "page", pageNumber, "frame", frame)
		MapPageToFrame(pid, pageNumber, frame)
	}
	models.ProcessDataLock.Lock()
	models.ProcessFramesTable[pid] = &models.ProcessFrames{PID: pid, Frames: freeFrames}
	delete(models.ProcessSwapTable, pid)
	models.ProcessDataLock.Unlock()

	// Actualizar bits de presencia de forma segura
	if err := updatePageTablePresenceBitsSafe(pid, true); err != nil {
		slog.Warn("Memoria: Error actualizando bits de presencia durante SWAP IN", "PID", pid, "error", err)
	}

	slog.Debug("Memoria: Bits de presencia actualizados (SWAP IN)")

	IncrementMetric(pid, "swap_out")

	slog.Info(fmt.Sprintf("Memoria: PID <%d> Removido de swap - Se restauran %d frames", pid, len(freeFrames)))
	return nil
}
func validateProcessExists(pid uint) bool {
	_, exists := models.ProcessTable[pid]
	return exists
}

func validateFrameIntegrity(frames []int) bool {
	for _, frame := range frames {
		if frame < 0 || frame >= len(models.FreeFrames) {
			slog.Error("Frame inválido detectado", "frame", frame, "max_frames", len(models.FreeFrames))
			return false
		}
	}
	return true
}
func updatePageTablePresenceBitsSafe(pid uint, present bool) error {
	// Obtener el proceso de la tabla de procesos
	process, exists := models.ProcessTable[pid]
	if !exists && present {
		return fmt.Errorf("proceso PID %d no encontrado al actualizar bits de presencia", pid)
	}

	// Si estamos marcando como no presente (swap out), iteramos sobre todas las páginas del proceso
	if !present {
		// Para swap out: marcar todas las páginas como no presentes
		if process != nil {
			for pageNumber := range process.Pages {
				if pageEntry := getPageEntryDirect(pid, pageNumber); pageEntry != nil {
					UpdatePageBit(pageEntry, "presence_off")
					slog.Debug("Memoria: Página marcada como NO presente (swap out)", "PID", pid, "page", pageNumber)
				}
			}
		}
	} else {
		// Para swap in: marcar páginas como presentes basándose en los frames asignados
		if process != nil {
			for pageNumber := range process.Pages {
				if pageEntry := getPageEntryDirect(pid, pageNumber); pageEntry != nil {
					UpdatePageBit(pageEntry, "presence_on")
					slog.Debug("Memoria: Página marcada como presente (swap in)", "PID", pid, "page", pageNumber)
				}
			}
		}
	}

	presentStr := "presentes"
	if !present {
		presentStr = "no presentes"
	}
	slog.Debug("Memoria: Bits de presencia actualizados", "PID", pid, "estado", presentStr)
	return nil
}
