package services

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/helpers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

func ExecuteDumpMemory(pid uint, size int) error {
	slog.Info(fmt.Sprintf("## PID: %d - Memory Dump solicitado", pid))
	dumpName := helpers.GetDumpName(pid)
	dumpFilePath := models.MemoryConfig.DumpPath + dumpName

	file, err := os.Create(dumpFilePath)
	if err != nil {
		slog.Error(fmt.Sprintf("error al crear archivo de dump: %v", err))
		return err
	}
	defer file.Close()

	// Adquirimos un lock de lectura, ya que no vamos a modificar las estructuras, solo leerlas.
	models.ProcessDataLock.RLock()
	defer models.ProcessDataLock.RUnlock()

	process, exists := models.ProcessTable[pid]
	if !exists {
		return fmt.Errorf("proceso PID %d no existe para realizar DUMP", pid)
	}

	// El tamaño del dump debe ser el tamaño real del proceso.
	actualSize := process.Size
	dumpData := make([]byte, 0, actualSize)

	numberPages := (actualSize + models.MemoryConfig.PageSize - 1) / models.MemoryConfig.PageSize

	// Para leer de la memoria de usuario, necesitamos su lock.
	models.UMemoryLock.RLock()
	defer models.UMemoryLock.RUnlock()

	for page := 0; page < numberPages; page++ {
		frame := SearchFrame(pid, page) // SearchFrame ya está protegido con su propio lock
		if frame == -1 {
			slog.Warn("Página no encontrada en memoria durante dump, rellenando con ceros", "pid", pid, "page", page)
			emptyPage := make([]byte, models.MemoryConfig.PageSize)
			dumpData = append(dumpData, emptyPage...)
			continue
		}

		startAddr := frame * models.MemoryConfig.PageSize
		endAddr := startAddr + models.MemoryConfig.PageSize
		if endAddr > len(models.UserMemory) {
			endAddr = len(models.UserMemory)
		}

		pageContent := models.UserMemory[startAddr:endAddr]
		dumpData = append(dumpData, pageContent...)
	}

	// Nos aseguramos de escribir solo el tamaño exacto del proceso
	if len(dumpData) > actualSize {
		dumpData = dumpData[:actualSize]
	}

	if _, err := file.Write(dumpData); err != nil {
		slog.Error(fmt.Sprintf("Fallo al escribir contenido en el archivo de dump '%s'", dumpFilePath))
		return fmt.Errorf("fallo al escribir datos al archivo de dump: %w", err)
	}

	slog.Debug(fmt.Sprintf("Memoria: Memory Dump completado para PID %d. Archivo: %s", pid, dumpFilePath))
	return nil
}

// CollectFramesFromTableV2 es una función recursiva para recolectar frames.
// Debe ser llamada desde un contexto que ya tenga un lock (RLock o Lock) sobre ProcessDataLock.
func CollectFramesFromTableV2(pid uint, table *models.PageTableLevel, frames *[]models.FrameInfo) {
	if table == nil {
		return
	}

	if table.IsLeaf && table.Entry != nil && table.Entry.Presence {
		frameInfo := models.FrameInfo{
			PID:   pid,
			Frame: table.Entry.Frame,
		}
		*frames = append(*frames, frameInfo)
	}

	for _, sub := range table.SubTables {
		CollectFramesFromTableV2(pid, sub, frames)
	}
}

// GroupFramesByPID no necesita locks porque opera sobre una copia de los datos.
func GroupFramesByPID(pid uint, flatFrames []models.FrameInfo) models.GroupedFrameInfo {
	pidToFramesMap := make(map[uint][]int)
	for _, fi := range flatFrames {
		if fi.PID == pid {
			pidToFramesMap[fi.PID] = append(pidToFramesMap[fi.PID], fi.Frame)
		}
	}

	frames, found := pidToFramesMap[pid]
	if !found {
		return models.GroupedFrameInfo{
			PID:    pid,
			Frames: []int{},
		}
	}

	return models.GroupedFrameInfo{
		PID:    pid,
		Frames: frames,
	}
}
