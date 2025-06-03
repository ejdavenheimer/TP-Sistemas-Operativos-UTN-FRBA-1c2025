package services

import (
	"fmt"
	"io"
	"os"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

func MoveToSwap(pid int) error {
	// Abrir archivo swapfile.bin (modo lectura/escritura, crear si no existe)
	file, err := os.OpenFile(models.MemoryConfig.SwapFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return fmt.Errorf("no se pudo abrir swapfile: %v", err)
	}
	defer file.Close()

	// Obtener frames del proceso (suponiendo que tenemos ProcessFramesTable)
	pf, ok := models.ProcessFramesTable[pid]
	if !ok {
		return fmt.Errorf("no se encontraron frames para PID %d", pid)
	}

	// Calcular offset para escribir: al final del archivo
	offset, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return fmt.Errorf("error buscando final de archivo: %v", err)
	}

	frameSize := int64(models.MemoryConfig.PageSize)
	totalSize := int64(0)

	for _, frameIndex := range pf.Frames {
		// Obtener inicio del frame en UserMemory
		start := int64(frameIndex) * frameSize
		end := start + frameSize
		if end > int64(len(models.UserMemory)) {
			return fmt.Errorf("frame %d excede tamaño de UserMemory", frameIndex)
		}

		data := models.UserMemory[start:end]

		n, err := file.Write(data)
		if err != nil {
			return fmt.Errorf("error escribiendo en swapfile: %v", err)
		}
		totalSize += int64(n)
	}

	// Guardar info en tabla de swap
	models.ProcessSwapTable[pid] = models.SwapEntry{
		Offset: offset,
		Size:   totalSize,
	}

	return nil
}
