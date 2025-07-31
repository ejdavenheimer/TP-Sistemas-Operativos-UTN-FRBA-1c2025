package services

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

func PutProcessInSwap(pid uint) error {
	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

	slog.Debug("Inicia PUT PROCESS IN SWAP")
	time.Sleep(time.Duration(models.MemoryConfig.SwapDelay) * time.Millisecond)

	// **INICIO DE LA CORRECCIÓN**
	// Verificar si el proceso ya fue swapeado. Si es así, la operación es exitosa.
	if _, inSwap := models.ProcessSwapTable[pid]; inSwap {
		slog.Debug("Proceso ya se encuentra en SWAP, omitiendo SWAP OUT.", "PID", pid)
		return nil
	}
	// **FIN DE LA CORRECCIÓN**

	processFrames, exists := models.ProcessFramesTable[pid]
	if !exists {
		// Ahora este error solo ocurrirá si el proceso realmente no existe o nunca tuvo frames.
		err := fmt.Errorf("proceso PID %d no encontrado en tabla de frames para swapear", pid)
		slog.Error(err.Error())
		return err
	}

	slog.Debug("Iniciando suspensión de proceso", "PID", pid)

	if len(processFrames.Frames) == 0 {
		slog.Debug(fmt.Sprintf("Proceso PID %d tiene 0 frames, moviendo a swap conceptualmente", pid))
		models.ProcessSwapTable[pid] = models.SwapEntry{Offset: 0, Size: 0}
		delete(models.ProcessFramesTable, pid)
		updatePageTablePresenceBits(pid, false) // Marcar páginas como no presentes
		IncrementMetric(pid, "swap_out")
		return nil
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

	var totalSize int64 = 0
	frameSize := int64(models.MemoryConfig.PageSize)

	models.UMemoryLock.Lock()
	for _, frameIndex := range processFrames.Frames {
		start := int64(frameIndex) * frameSize
		end := start + frameSize
		data := models.UserMemory[start:end]
		n, err := file.Write(data)
		if err != nil {
			models.UMemoryLock.Unlock()
			slog.Error(fmt.Sprintf("error escribiendo frame %d en swapfile: %v", frameIndex, err))
			return err
		}
		totalSize += int64(n)
		models.FreeFrames[frameIndex] = true
	}
	models.UMemoryLock.Unlock()

	models.ProcessSwapTable[pid] = models.SwapEntry{Offset: offset, Size: totalSize}
	delete(models.ProcessFramesTable, pid)
	updatePageTablePresenceBits(pid, false) // Marcar páginas como no presentes
	slog.Debug("SE ACTUALIZAN BITS DE PRESENCIA EN TABLAS DE PAGINA (SWAP OUT)")
	IncrementMetric(pid, "swap_out")

	slog.Info(fmt.Sprintf("PID <%d> movido a swap - Offset: %d, Tamaño: %d", pid, offset, totalSize))
	return nil
}

func RemoveProcessInSwap(pid uint) error {
	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

	slog.Debug("INICIA REMOVE PROCESS IN SWAP")
	time.Sleep(time.Duration(models.MemoryConfig.SwapDelay) * time.Millisecond)

	swapEntry, exists := models.ProcessSwapTable[pid]
	if !exists {
		err := fmt.Errorf("el proceso con PID %d no se encuentra en SWAP", pid)
		slog.Error(err.Error())
		return err
	}

	slog.Debug(fmt.Sprintf("Inicio RemoveProcessInSwap para PID %d", pid))

	if swapEntry.Size == 0 {
		slog.Debug(fmt.Sprintf("Proceso PID %d (0 frames) removido de swap conceptualmente", pid))
		models.ProcessFramesTable[pid] = &models.ProcessFrames{PID: pid, Frames: []int{}}
		delete(models.ProcessSwapTable, pid)
		// EActualizar bit de presencia
		updatePageTablePresenceBits(pid, true)
		return nil
	}

	frameSize := int64(models.MemoryConfig.PageSize)
	framesNeeded := int(swapEntry.Size / frameSize)

	models.UMemoryLock.Lock()
	freeFrames := []int{}
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

	for _, frameIdx := range freeFrames {
		models.FreeFrames[frameIdx] = false
	}
	models.UMemoryLock.Unlock()

	file, err := os.Open(models.MemoryConfig.SwapFilePath)
	if err != nil {
		slog.Error(fmt.Sprintf("no se pudo abrir el archivo de swap: %v", err))
		models.UMemoryLock.Lock()
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
		slog.Error(fmt.Sprintf("error al leer contenido del proceso desde SWAP: %v", err))
		models.UMemoryLock.Lock()
		for _, frameIdx := range freeFrames {
			models.FreeFrames[frameIdx] = true
		}
		models.UMemoryLock.Unlock()
		return err
	}

	models.UMemoryLock.Lock()
	for i, frameIdx := range freeFrames {
		start := frameIdx * models.MemoryConfig.PageSize
		copy(models.UserMemory[start:start+models.MemoryConfig.PageSize], processData[i*models.MemoryConfig.PageSize:(i+1)*models.MemoryConfig.PageSize])
	}
	models.UMemoryLock.Unlock()

	if _, exists := models.PageTables[pid]; !exists {
		slog.Error(fmt.Sprintf("FALLO CRÍTICO: No existe tabla de páginas para PID %d al restaurar desde swap", pid))
		return fmt.Errorf("tabla de páginas no encontrada para PID %d", pid)
	}
	for pageNumber := 0; pageNumber < len(freeFrames); pageNumber++ {
		frame := freeFrames[pageNumber]
		slog.Debug(fmt.Sprintf("Mapeando página %d al frame %d para PID %d", pageNumber, frame, pid))
		MapPageToFrame(pid, pageNumber, frame)

		// Verificar que el mapeo funcionó
		if _, exists := models.PageTables[pid]; !exists {
			slog.Error(fmt.Sprintf("FALLO CRÍTICO: Tabla de páginas desapareció durante mapeo de página %d para PID %d", pageNumber, pid))
			return fmt.Errorf("tabla de páginas perdida durante mapeo para PID %d", pid)
		}
	}

	updatePageTablePresenceBits(pid, true) // Marcar páginas como presentes
	slog.Debug("SE ACTUALIZAN BITS DE PRESENCIA EN TABLAS DE PAGINA (SWAP IN)")

	models.ProcessFramesTable[pid] = &models.ProcessFrames{PID: pid, Frames: freeFrames}
	delete(models.ProcessSwapTable, pid)
	IncrementMetric(pid, "swap_in")

	slog.Info(fmt.Sprintf("PID <%d> removido de swap - Se restauran %d frames", pid, len(freeFrames)))
	return nil
}

func updatePageTablePresenceBits(pid uint, present bool) {
	// Obtener el proceso de la tabla de procesos

	process, exists := models.ProcessTable[pid]
	if !exists && present {
		slog.Warn(fmt.Sprintf("No se encontró proceso PID %d al actualizar bits de presencia", pid))
		return
	}

	// Si estamos marcando como no presente (swap out), iteramos sobre todas las páginas del proceso
	if !present {
		// Para swap out: marcar todas las páginas como no presentes
		if process != nil {
			for pageNumber := range process.Pages {
				if pageEntry := getPageEntryDirect(pid, pageNumber); pageEntry != nil {
					UpdatePageBit(pageEntry, "presence_off")
					slog.Debug(fmt.Sprintf("PID %d: Página %d marcada como NO presente (swap out)", pid, pageNumber))
				}
			}
		}
	} else {
		// Para swap in: marcar páginas como presentes basándose en los frames asignados
		if process != nil {
			for pageNumber := range process.Pages {
				if pageEntry := getPageEntryDirect(pid, pageNumber); pageEntry != nil {
					UpdatePageBit(pageEntry, "presence_on")
					slog.Debug(fmt.Sprintf("PID %d: Página %d marcada como presente (swap in)", pid, pageNumber))
				}
			}
		}
	}

	presentStr := "presentes"
	if !present {
		presentStr = "no presentes"
	}
	slog.Debug(fmt.Sprintf("Bits de presencia actualizados para PID %d: páginas marcadas como %s", pid, presentStr))
}
