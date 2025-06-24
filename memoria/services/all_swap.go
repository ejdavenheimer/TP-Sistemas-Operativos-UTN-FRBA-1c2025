package services

import (
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

// En el archivo de swap, cambiar las líneas que usan ProcessFramesTable:

func PutProcessInSwap(pid uint) error {
	// Abrir (o crear) el archivo swapfile.bin en modo lectura/escritura
	file, err := os.OpenFile(models.MemoryConfig.SwapFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("no se pudo abrir swapfile: %v", err))
		return err
	}
	defer file.Close()

	// Obtener la lista de índices de frame para este PID
	processFrames, exists := models.ProcessFramesTable[pid]
	if !exists || processFrames == nil { // *** CAMBIO: verificar que no sea nil ***
		slog.Error(fmt.Sprintf("No está en tabla de frames el PID %d", pid))
		return fmt.Errorf("proceso PID %d no encontrado en tabla de frames", pid)
	}

	slog.Info("Iniciando suspensión de proceso", "PID", pid)

	slog.Debug("Frames libres antes de swap-out", "cantidad", contarFramesLibres())
	slog.Debug("Tamaño actual del archivo de swap", "bytes", obtenerTamanioSwap())

	// Llevar el cursor al final del archivo para escribir los datos al final
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		slog.Error(fmt.Sprintf("error posicionando el cursor al final de swapfile: %v", err))
		return err
	}

	var totalSize int64 = 0
	frameSize := int64(models.MemoryConfig.PageSize)

	// Por cada frame asignado al proceso, volcamos su contenido a SWAP y lo liberamos
	for _, frameIndex := range processFrames.Frames { // *** CAMBIO: acceder con .Frames ***
		frame := models.FrameTable[frameIndex]
		start := int64(frame.StartAddr)
		end := start + frameSize

		if end > int64(len(models.UserMemory)) {
			slog.Error(fmt.Sprintf("los límites del frame %d exceden UserMemory", frameIndex))
			return fmt.Errorf("límites del frame %d exceden UserMemory", frameIndex)
		}

		data := models.UserMemory[start:end]
		n, err := file.Write(data)
		if err != nil {
			slog.Error(fmt.Sprintf("error escribiendo frame %d en swapfile: %v", frameIndex, err))
			return err
		}
		totalSize += int64(n)

		// Marcar el frame como libre
		models.FrameTable[frameIndex].IsFree = true
	}

	// Registrar en la tabla de SWAP dónde quedaron los datos de este proceso
	models.ProcessSwapTable[pid] = models.SwapEntry{
		Offset: offset,
		Size:   totalSize,
	}

	// Eliminar la entrada de ProcessFramesTable, ya no ocupa frames en memoria
	delete(models.ProcessFramesTable, pid)

	slog.Info(fmt.Sprintf("Proceso PID %d movido a swap. Offset: %d, Tamaño: %d", pid, offset, totalSize))
	slog.Info(fmt.Sprintf("Frames liberados para PID %d", pid))
	slog.Debug("Frames libres después de swap-out", "cantidad", contarFramesLibres())
	slog.Debug("Tamaño del archivo de swap luego del guardado", "bytes", obtenerTamanioSwap())

	return nil
}

func RemoveProcessInSwap(pid uint) error {
	// Buscar la entrada del proceso en la tabla de swap
	swapEntry, exists := models.ProcessSwapTable[pid]
	if !exists {
		err := fmt.Errorf("el proceso con PID %d no se encuentra en SWAP", pid)
		slog.Error(err.Error())
		return err
	}

	slog.Info(fmt.Sprintf("Inicio RemoveProcessInSwap para PID %d", pid))
	slog.Debug("Tamaño actual del archivo de swap", "bytes", obtenerTamanioSwap())
	slog.Debug("Frames libres antes de swap-in", "cantidad", contarFramesLibres())

	// Calcular cuántos frames necesita el proceso para volver a cargarse en memoria
	frameSize := int64(models.MemoryConfig.PageSize)
	framesNeeded := int(swapEntry.Size / frameSize)

	// Verificar que haya suficientes frames libres en memoria
	freeFrames := []int{}
	for idx, frame := range models.FrameTable {
		if frame.IsFree {
			freeFrames = append(freeFrames, idx)
			if len(freeFrames) == framesNeeded {
				break
			}
		}
	}
	if len(freeFrames) < framesNeeded {
		return fmt.Errorf("no hay suficientes frames libres para des-suspender el proceso PID %d", pid)
	}

	// Abrir el archivo swapfile.bin para leer el contenido del proceso
	file, err := os.Open(models.MemoryConfig.SwapFilePath)
	if err != nil {
		slog.Error(fmt.Sprintf("no se pudo abrir el archivo de swap: %v", err))
		return err
	}
	defer file.Close()

	// Mover el puntero de lectura al offset donde está el contenido del proceso
	_, err = file.Seek(swapEntry.Offset, io.SeekStart)
	if err != nil {
		slog.Error(fmt.Sprintf("error al posicionarse en el offset %d del archivo swap: %v", swapEntry.Offset, err))
		return err
	}

	// Leer el contenido del proceso desde el archivo swap
	processData := make([]byte, swapEntry.Size)
	_, err = io.ReadFull(file, processData)
	if err != nil {
		slog.Error(fmt.Sprintf("error al leer contenido del proceso desde SWAP: %v", err))
		return err
	}

	// Escribir el contenido del proceso en UserMemory utilizando los frames libres encontrados
	for i, frameIdx := range freeFrames {
		start := frameIdx * models.MemoryConfig.PageSize
		end := start + models.MemoryConfig.PageSize
		copy(models.UserMemory[start:end], processData[i*models.MemoryConfig.PageSize:(i+1)*models.MemoryConfig.PageSize])

		// Marcar el frame como ocupado
		models.FrameTable[frameIdx].IsFree = false
	}

	// Guardar los frames asignados al proceso
	models.ProcessFramesTable[pid] = &models.ProcessFrames{ // *** CAMBIO: crear puntero ***
		PID:    pid,
		Frames: freeFrames,
	}

	// Eliminar el proceso de la tabla de procesos en swap
	delete(models.ProcessSwapTable, pid)

	slog.Info(fmt.Sprintf("Proceso PID %d removido de swap y cargado en UserMemory", pid))
	slog.Info(fmt.Sprintf("Frames asignados al proceso PID %d: %v", pid, freeFrames))
	slog.Debug("Frames libres después de swap-in", "cantidad", contarFramesLibres())
	slog.Debug("Tamaño del archivo de swap luego del swap-in", "bytes", obtenerTamanioSwap())

	return nil
}

func contarFramesLibres() int {
	count := 0
	for _, f := range models.FrameTable {
		if f.IsFree {
			count++
		}
	}
	return count
}

func obtenerTamanioSwap() int64 {
	info, err := os.Stat(models.MemoryConfig.SwapFilePath)
	if err != nil {
		slog.Warn("No se pudo obtener tamaño del archivo de swap", "error", err)
		return -1
	}
	return info.Size()
}

func MockCargarProcesosEnMemoria() {
	pageSize := models.MemoryConfig.PageSize

	// Proceso 1 con dos páginas (frames 0 y 1)
	models.ProcessFramesTable[1] = &models.ProcessFrames{
		PID:    1,
		Frames: []int{0, 1},
	}
	for i, frameIdx := range models.ProcessFramesTable[1].Frames {
		start := models.FrameTable[frameIdx].StartAddr
		end := start + pageSize
		copy(models.UserMemory[start:end], []byte(fmt.Sprintf("PID 1 - Frame %d", i)))
		models.FrameTable[frameIdx].IsFree = false
	}

	// Proceso 2 con una página (frame 2)
	models.ProcessFramesTable[2] = &models.ProcessFrames{
		PID:    2,
		Frames: []int{2},
	}
	for i, frameIdx := range models.ProcessFramesTable[2].Frames {
		start := models.FrameTable[frameIdx].StartAddr
		end := start + pageSize
		copy(models.UserMemory[start:end], []byte(fmt.Sprintf("PID 2 - Frame %d", i)))
		models.FrameTable[frameIdx].IsFree = false
	}

	slog.Info("Mock: procesos 1 y 2 cargados en memoria")
}
