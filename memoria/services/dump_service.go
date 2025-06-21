package services

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/helpers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"log/slog"
)

func ExecuteDumpMemory(pid uint, size int) error {
	slog.Info(fmt.Sprintf("## PID: %d - Memory Dump solicitado", pid)) //Log obligatorio
	dumpName := helpers.GetDumpName(pid)                               //obtiene el nombre del archivo

	//crea o abre el archivo
	dumpFilePath := models.MemoryConfig.DumpPath + dumpName
	file, err := helpers.CreateFile(dumpFilePath, size)
	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return err
	}
	defer file.Close()

	numberPages := size / models.MemoryConfig.PageSize

	if size%models.MemoryConfig.PageSize != 0 {
		numberPages++ // Si el tamaño no es múltiplo de PageSize, redondear hacia arriba
	}

	allFrames := make([]models.FrameInfo, 0) // Inicializa un slice vacío

	// Itera sobre todas las tablas de páginas de los procesos
	for _, rootTable := range models.PageTables { // Asumo models.PageTables es accesible
		CollectFramesFromTableV2(pid, rootTable, &allFrames) // Pasa el slice por referencia
	}

	groupedOutput := GroupFramesByPID(pid, allFrames)

	//buscar frame
	frame := SearchFrame(groupedOutput.PID, groupedOutput.Frames)
	//
	data, err := readFromMemory(frame, size)
	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return err
	}

	slog.Debug(fmt.Sprintf("PID: %d - Content: %s", pid, string(data)))

	// 5. Escribir el contenido en el archivo
	_, err = file.Write(data[:size]) // Escribir solo hasta el 'size' real
	if err != nil {
		slog.Error(fmt.Sprintf("Memoria: Fallo al escribir contenido en el archivo de dump '%s'", dumpFilePath))
		return fmt.Errorf("fallo al escribir datos al archivo de dump: %w", err)
	}

	slog.Debug(fmt.Sprintf("Memoria: Memory Dump completado para PID %d. Archivo: %s", pid, dumpFilePath))

	return nil
}

func GetPageContent(pid uint, pageNum int) ([]byte, error) {
	slog.Warn("TODO: implementar ")
	return nil, nil
}

func CollectFramesFromTableV2(pid uint, table *models.PageTableLevel, frames *[]models.FrameInfo) {
	if table == nil {
		return
	}

	// Si es una hoja y la entrada de página está presente, recolecta el frame.
	if table.IsLeaf && table.Entry != nil && table.Entry.Presence {
		frameInfo := models.FrameInfo{
			PID:   pid,
			Frame: table.Entry.Frame,
		}
		*frames = append(*frames, frameInfo) // Añade al slice pasado por puntero
		slog.Debug(fmt.Sprintf("Memoria: PID: %d - Frame: %d recolectado", pid, table.Entry.Frame))
	}

	// Recorre las subtables recursivamente
	for _, sub := range table.SubTables {
		CollectFramesFromTableV2(pid, sub, frames) // Pasa el mismo slice por referencia
	}
}

// GroupFramesByPID toma un slice plano de models.FrameInfo
// y lo transforma en un slice de models.GroupedFrames, agrupando los frames por PID.
func GroupFramesByPID(pid uint, flatFrames []models.FrameInfo) models.GroupedFrameInfo {
	// Usamos un mapa temporal para agrupar los frames por PID de manera eficiente
	pidToFramesMap := make(map[uint][]int)

	// Recorre la lista plana y agrupa los frames en el mapa
	for _, fi := range flatFrames {
		if fi.PID == pid {
			pidToFramesMap[fi.PID] = append(pidToFramesMap[fi.PID], fi.Frame)
		}
	}

	frames, found := pidToFramesMap[pid]
	if !found {
		slog.Debug(fmt.Sprintf("Memoria: No se encontraron frames para el PID %d.", pid))
		// Si no se encuentra el PID, devuelve una estructura con Entries vacío.
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
