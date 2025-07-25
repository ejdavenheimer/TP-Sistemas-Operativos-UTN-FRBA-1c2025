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
	time.Sleep(time.Duration(models.MemoryConfig.SwapDelay) * time.Millisecond)

	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

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
	delete(models.PageTables, pid)
	IncrementMetric(pid, "swap_out")

	slog.Debug(fmt.Sprintf("Proceso PID %d movido a swap. Offset: %d, Tamaño: %d", pid, offset, totalSize))
	return nil
}

func RemoveProcessInSwap(pid uint) error {
	time.Sleep(time.Duration(models.MemoryConfig.SwapDelay) * time.Millisecond)

	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

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
		// Es necesario inicializar la tabla de páginas aunque no tenga frames
		initializePageTables(pid)
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

	initializePageTables(pid)
	for pageNumber, frame := range freeFrames {
		MapPageToFrame(pid, pageNumber, frame)
	}

	models.ProcessFramesTable[pid] = &models.ProcessFrames{PID: pid, Frames: freeFrames}
	delete(models.ProcessSwapTable, pid)
	IncrementMetric(pid, "swap_in")

	slog.Debug(fmt.Sprintf("Proceso PID %d removido de swap y cargado en UserMemory", pid))
	return nil
}
