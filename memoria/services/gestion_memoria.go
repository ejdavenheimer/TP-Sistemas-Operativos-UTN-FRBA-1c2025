package services

import (
	"fmt"
	"log/slog"
	"math"
	"runtime/debug"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

func ReserveMemory(pid uint, size int, path string) error {
	if size < 0 {
		return fmt.Errorf("el tamaño del proceso debe ser mayor a 0 (PID %d)", pid)
	}

	pageSize := models.MemoryConfig.PageSize
	pageCount := int(math.Ceil(float64(size) / float64(pageSize)))

	models.UMemoryLock.RLock()
	freeFramesCount := CountFreeFrames()
	models.UMemoryLock.RUnlock()

	if freeFramesCount < pageCount {
		return fmt.Errorf("no hay suficientes frames libres para el proceso PID %d", pid)
	}

	if err := GetInstructionsByName(pid, path, models.InstructionsMap, models.MemoryConfig.ScriptsPath); err != nil {
		slog.Error("Error al cargar instrucciones", "error", err)
		return fmt.Errorf("falló la carga de instrucciones para el PID %d", pid)
	}

	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

	models.UMemoryLock.Lock()
	defer models.UMemoryLock.Unlock()

	assignedFrames := make([]int, 0, pageCount)
	// Re-verificamos la disponibilidad de frames dentro del lock de escritura
	if CountFreeFrames() < pageCount {
		return fmt.Errorf("condición de carrera detectada: no hay suficientes frames libres para el proceso PID %d", pid)
	}

	for i := 0; i < pageCount; i++ {
		frame := AllocateFrame()
		if frame == -1 {
			for _, f := range assignedFrames {
				models.FreeFrames[f] = true
			}
			return fmt.Errorf("falló la asignación de frames para PID %d", pid)
		}
		assignedFrames = append(assignedFrames, frame)
	}

	initializePageTables(pid)
	for i, frame := range assignedFrames {
		MapPageToFrame(pid, i, frame)
	}
	NewProcess(pid, size, pageCount, assignedFrames)

	slog.Debug("PCB registrado", slog.Int("pid", int(pid)), slog.Int("pages", pageCount), slog.Int("size", size))
	return nil
}

func CountFreeFrames() int {
	count := 0
	for _, isFree := range models.FreeFrames {
		if isFree {
			count++
		}
	}
	return count
}

func MapPageToFrame(pid uint, pageNumber int, frame int) {
	numLevels := models.MemoryConfig.NumberOfLevels
	entriesPerLevel := models.MemoryConfig.EntriesPerPage
	indices := getPageIndices(pageNumber, numLevels, entriesPerLevel)

	current := models.PageTables[pid]
	for level := 0; level < numLevels-1; level++ {
		idx := indices[level]
		if _, exists := current.SubTables[idx]; !exists {
			current.SubTables[idx] = &models.PageTableLevel{SubTables: make(map[int]*models.PageTableLevel)}
		}
		current = current.SubTables[idx]
	}

	lastIdx := indices[numLevels-1]
	current.SubTables[lastIdx] = &models.PageTableLevel{
		IsLeaf: true,
		Entry: &models.PageEntry{
			Frame:    frame,
			Presence: true,
		},
	}
}

// DEBUGUEAR
func initializePageTables(pid uint) {
	slog.Debug(fmt.Sprintf("initializePageTables llamada para PID %d", pid))
	if _, exists := models.PageTables[pid]; !exists {
		models.PageTables[pid] = &models.PageTableLevel{
			SubTables: make(map[int]*models.PageTableLevel),
		}
		slog.Debug(fmt.Sprintf("Nueva tabla de páginas creada para PID %d", pid))
	} else {
		slog.Debug(fmt.Sprintf("Tabla de páginas ya existe para PID %d", pid))
	}
}

func NewProcess(pid uint, size int, pageCount int, assignedFrames []int) {
	pages := make([]models.PageEntry, pageCount)
	for i := 0; i < pageCount; i++ {
		pages[i] = models.PageEntry{Frame: assignedFrames[i], Presence: true}
	}

	models.ProcessTable[pid] = &models.Process{
		Pid:     pid,
		Size:    size,
		Pages:   pages,
		Metrics: &models.Metrics{},
	}
	models.ProcessMetrics[pid] = &models.Metrics{}
	models.ProcessFramesTable[pid] = &models.ProcessFrames{PID: pid, Frames: assignedFrames}
}

func SearchFrame(pid uint, pageNumber int) int {
	slog.Debug(fmt.Sprintf("SearchFrame llamado - PID: %d, Página: %d", pid, pageNumber))
	models.ProcessDataLock.RLock()
	defer models.ProcessDataLock.RUnlock()

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

func FindPageEntry(pid uint, root *models.PageTableLevel, pageNumber int, incrementMetrics bool) (*models.PageEntry, error) {
	slog.Debug("Se accedio a FIND PAGE ENTRY")
	slog.Debug(fmt.Sprintf("Stack trace: %s", debug.Stack()))
	currentLevel := root
	indices := getPageIndices(pageNumber, models.MemoryConfig.NumberOfLevels, models.MemoryConfig.EntriesPerPage)
	slog.Debug(fmt.Sprintf("INDICES: %v", indices))

	for i, index := range indices {
		if !currentLevel.IsLeaf && incrementMetrics {
			IncrementMetric(pid, "page_table")
			slog.Debug(fmt.Sprintf("Acceso a tabla de páginas PID %d - Accesos totales: %d",
				pid, models.ProcessMetrics[pid].PageTableAccesses))
			// Delay por nivel de tabla de paginas
			time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)
		}

		nextLevel, exists := currentLevel.SubTables[index]
		if !exists {
			return nil, fmt.Errorf("nivel intermedio no encontrado")
		}

		if i == len(indices)-1 {
			if nextLevel.IsLeaf && nextLevel.Entry != nil && nextLevel.Entry.Presence {
				return nextLevel.Entry, nil
			}
			return nil, fmt.Errorf("entrada de página no presente o no es hoja")
		}
		currentLevel = nextLevel
	}
	return nil, fmt.Errorf("lógica de búsqueda de página inválida")
}

func getPageIndices(pageNumber int, levels int, entriesPerLevel int) []int {
	indices := make([]int, levels)
	tempPageNumber := pageNumber
	for i := levels - 1; i >= 0; i-- {
		indices[i] = tempPageNumber % entriesPerLevel
		tempPageNumber /= entriesPerLevel
	}
	return indices
}

func AllocateFrame() int {
	for i, free := range models.FreeFrames {
		if free {
			models.FreeFrames[i] = false
			return i
		}
	}
	slog.Error("No hay frames libres disponibles para asignar")
	return -1
}
