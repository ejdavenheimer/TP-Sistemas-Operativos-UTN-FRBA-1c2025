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
	process, exists := models.ProcessTable[pid]
	if !exists {
		return fmt.Errorf("proceso PID %d no existe para realizar DUMP", pid)
	}
	models.ProcessDataLock.RUnlock()
	entrySwap, exists := models.ProcessSwapTable[pid]
	if exists {
		err := fmt.Errorf("el proceso con PID %d no se encuentra en SWAP - ENTRY SWAP: %v", pid, entrySwap)
		slog.Error(err.Error())
		return err
	}
	slog.Debug(fmt.Sprintf("PID %d está en memoria, realizando dump desde UserMemory", pid))
	return dumpFromMemory(pid, process, file)
}

func dumpFromMemory(pid uint, process *models.Process, file *os.File) error {

	// El tamaño del dump debe ser el tamaño real del proceso.
	actualSize := process.Size
	dumpData := make([]byte, 0, actualSize)

	numberPages := (actualSize + models.MemoryConfig.PageSize - 1) / models.MemoryConfig.PageSize

	// Para leer de la memoria de usuario, necesitamos su lock.
	models.UMemoryLock.Lock()
	slog.Debug("UMemoryLock lockeado DUMP MEMORY")
	defer models.UMemoryLock.Unlock()

	for page := 0; page < numberPages; page++ {
		frame := SearchFrameWithoutLock(pid, page) // SearchFrame ya está protegido con su propio lock
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
		slog.Error("Fallo al escribir contenido en el archivo de dump")
		return fmt.Errorf("fallo al escribir datos al archivo de dump: %w", err)
	}

	slog.Info(fmt.Sprintf("Memoria: Memory Dump completado desde memoria para PID %d", pid))
	return nil
}

func SearchFrameWithoutLock(pid uint, pageNumber int) int {
	slog.Debug(fmt.Sprintf("SearchFrame llamado - PID: %d, Página: %d", pid, pageNumber))

	pageTableRoot, exists := models.PageTables[pid]
	if !exists {
		slog.Warn("Tabla de páginas no encontrada para PID", "pid", pid)
		return -1
	}

	slog.Debug("SearchFrame recibido", "pid", pid, "pageNumber", pageNumber)
	entry, err := FindPageEntry(pid, pageTableRoot, pageNumber, true)
	if err != nil {
		slog.Warn("No se encontró la entrada de página", "pid", pid, "page", pageNumber, "error", err)
		return -1
	}

	return entry.Frame
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
