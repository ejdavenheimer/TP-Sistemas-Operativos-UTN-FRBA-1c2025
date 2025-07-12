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

var ProcessTableLock sync.RWMutex

func PutProcessInSwap(pid uint) error {
	// Delay de swap
	time.Sleep(time.Duration(models.MemoryConfig.SwapDelay) * time.Millisecond)

	// Obtener la lista de índices de frame para este PID
	processFrames, exists := models.ProcessFramesTable[pid]
	if !exists {
		slog.Error(fmt.Sprintf("No está en tabla de frames el PID %d", pid))
		return fmt.Errorf("proceso PID %d no encontrado en tabla de frames", pid)
	}

	slog.Debug("Iniciando suspensión de proceso", "PID", pid)
	slog.Debug("Frames libres antes de poner proceso en swap", "cantidad", contarFramesLibres())
	slog.Debug("Tamaño actual del archivo de swap", "bytes", obtenerTamanioSwap())

	// Caso especial: proceso con 0 frames
	if processFrames == nil || len(processFrames.Frames) == 0 {
		slog.Debug(fmt.Sprintf("Proceso PID %d tiene 0 frames, solo moviendo entre tablas", pid))

		// Registrar en la tabla de SWAP con tamaño 0
		models.ProcessSwapTable[pid] = models.SwapEntry{
			Offset: 0,
			Size:   0,
		}

		// Incrementar métrica de swap realizado
		IncrementMetric(pid, "swap_out")

		// Eliminar la entrada de ProcessFramesTable
		delete(models.ProcessFramesTable, pid)

		slog.Debug(fmt.Sprintf("Proceso PID %d (0 frames) movido a swap conceptualmente", pid))
		return nil
	}

	// Caso normal: proceso con frames
	// Abrir (o crear) el archivo swapfile.bin en modo lectura/escritura
	file, err := os.OpenFile(models.MemoryConfig.SwapFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		slog.Error(fmt.Sprintf("no se pudo abrir swapfile: %v", err))
		return err
	}
	defer file.Close()

	// Llevar el cursor al final del archivo para escribir los datos al final
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		slog.Error(fmt.Sprintf("error posicionando el cursor al final de swapfile: %v", err))
		return err
	}

	var totalSize int64 = 0
	frameSize := int64(models.MemoryConfig.PageSize)

	// Por cada frame asignado al proceso, ponemos su contenido en SWAP y lo liberamos
	for _, frameIndex := range processFrames.Frames {
		// Calcular la dirección de inicio del frame basándose en el índice
		start := int64(frameIndex * models.MemoryConfig.PageSize)
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

		// Marcar el frame como libre en FreeFrames
		models.FreeFrames[frameIndex] = true
	}

	// Registrar en la tabla de SWAP dónde quedaron los datos de este proceso
	models.ProcessSwapTable[pid] = models.SwapEntry{
		Offset: offset,
		Size:   totalSize,
	}
	//Incremento la métrica de swap realizado
	IncrementMetric(pid, "swap_out")

	// Eliminar la entrada de ProcessFramesTable, ya que no ocupa frames en memoria
	delete(models.ProcessFramesTable, pid)

	slog.Debug(fmt.Sprintf("Proceso PID %d movido a swap. Offset: %d, Tamaño: %d", pid, offset, totalSize))
	slog.Debug(fmt.Sprintf("Frames liberados para PID %d", pid))
	slog.Debug("Frames libres después de swap-out", "cantidad", contarFramesLibres())
	slog.Debug("Tamaño del archivo de swap luego del guardado", "bytes", obtenerTamanioSwap())

	return nil
}

func RemoveProcessInSwap(pid uint) error {
	// Delay de swap
	time.Sleep(time.Duration(models.MemoryConfig.SwapDelay) * time.Millisecond)

	// Buscar la entrada del proceso en la tabla de swap
	swapEntry, exists := models.ProcessSwapTable[pid]
	if !exists {
		err := fmt.Errorf("el proceso con PID %d no se encuentra en SWAP", pid)
		slog.Error(err.Error())
		return err
	}

	slog.Debug(fmt.Sprintf("Inicio RemoveProcessInSwap para PID %d", pid))
	slog.Debug("Tamaño actual del archivo de swap", "bytes", obtenerTamanioSwap())
	slog.Debug("Frames libres antes de sacar proceso de swap", "cantidad", contarFramesLibres())

	// Caso especial: proceso con 0 frames
	if swapEntry.Size == 0 {
		slog.Debug(fmt.Sprintf("Proceso PID %d tiene 0 frames, solo moviendo entre tablas", pid))

		// Crear entrada en ProcessFramesTable con slice vacío
		models.ProcessFramesTable[pid] = &models.ProcessFrames{
			PID:    pid,
			Frames: []int{}, // Slice vacío para proceso sin frames
		}

		// Eliminar el proceso de la tabla de procesos en swap
		delete(models.ProcessSwapTable, pid)

		slog.Debug(fmt.Sprintf("Proceso PID %d (0 frames) removido de swap conceptualmente", pid))
		return nil
	}

	// Caso normal: proceso con frames
	// Calcular cuántos frames necesita el proceso para volver a cargarse en memoria
	frameSize := int64(models.MemoryConfig.PageSize)
	framesNeeded := int(swapEntry.Size / frameSize)

	// Verificar que haya suficientes frames libres en memoria
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

		// Marcar el frame como ocupado en FreeFrames
		models.FreeFrames[frameIdx] = false
	}

	// Guardar los frames asignados al proceso
	models.ProcessFramesTable[pid] = &models.ProcessFrames{
		PID:    pid,
		Frames: freeFrames,
	}

	// Eliminar el proceso de la tabla de procesos en swap
	delete(models.ProcessSwapTable, pid)

	slog.Debug(fmt.Sprintf("Proceso PID %d removido de swap y cargado en UserMemory", pid))
	slog.Debug(fmt.Sprintf("Frames asignados al proceso PID %d: %v", pid, freeFrames))
	slog.Debug("Frames libres después de swap-in", "cantidad", contarFramesLibres())
	slog.Debug("Tamaño del archivo de swap luego del swap-in", "bytes", obtenerTamanioSwap())

	return nil
}

func contarFramesLibres() int {
	count := 0
	for _, free := range models.FreeFrames {
		if free {
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
