package services

import (
	"fmt"
	"io"
	"os"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

func PutProcessInSwap(pid int) error {
	// Abrir (o crear) el archivo swapfile.bin en modo lectura/escritura
	file, err := os.OpenFile(models.MemoryConfig.SwapFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("no se pudo abrir swapfile: %v", err)
	}
	defer file.Close()

	// Obtener la lista de índices de frame para este PID
	pf, exists := models.ProcessFramesTable[pid]
	if !exists {
		return fmt.Errorf("no se encontraron frames para el PID %d", pid)
	}

	// Llevar el cursor al final del archivo para escribir los datos al final
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("error posicionando el cursor al final de swapfile: %v", err)
	}

	var totalSize int64 = 0
	frameSize := int64(models.MemoryConfig.PageSize)

	// Por cada frame asignado al proceso, volcamos su contenido a SWAP y lo liberamos
	for _, frameIndex := range pf.Frames {
		frame := models.FrameTable[frameIndex]
		start := int64(frame.StartAddr)
		end := start + frameSize

		if end > int64(len(models.UserMemory)) {
			return fmt.Errorf("los límites del frame %d exceden UserMemory", frameIndex)
		}

		data := models.UserMemory[start:end]
		n, err := file.Write(data)
		if err != nil {
			return fmt.Errorf("error escribiendo frame %d en swapfile: %v", frameIndex, err)
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

	// TODO: aquí se debe implementar la liberación de UserMemory y de la tabla de páginas
	//     para el proceso 'pid'. Es decir:
	//       • Liberar/llenar con ceros los bytes de UserMemory que correspondían a este PID
	//       • Destruir o marcar como inválidas las entradas de PageTables[pid]

	return nil
}

func RemoveProcessInSwap(pid int) error {
	// Buscar la entrada del proceso en la tabla de swap
	swapEntry, exists := models.ProcessSwapTable[pid]
	if !exists {
		return fmt.Errorf("el proceso con PID %d no se encuentra en SWAP", pid)
	}

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
		return fmt.Errorf("no se pudo abrir el archivo de swap: %v", err)
	}
	defer file.Close()

	// Mover el puntero de lectura al offset donde está el contenido del proceso
	_, err = file.Seek(swapEntry.Offset, io.SeekStart)
	if err != nil {
		return fmt.Errorf("error al posicionarse en el offset %d del archivo swap: %v", swapEntry.Offset, err)
	}

	// Leer el contenido del proceso desde el archivo swap
	processData := make([]byte, swapEntry.Size)
	_, err = io.ReadFull(file, processData)
	if err != nil {
		return fmt.Errorf("error al leer contenido del proceso desde SWAP: %v", err)
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
	models.ProcessFramesTable[pid] = models.ProcessFrames{
		PID:    pid,
		Frames: freeFrames,
	}

	// Eliminar el proceso de la tabla de procesos en swap
	delete(models.ProcessSwapTable, pid)

	// TODO: reconstruir la tabla de páginas para el proceso PID
	// Esto depende del diseño jerárquico y debe realizarlo otro submódulo

	return nil
}
