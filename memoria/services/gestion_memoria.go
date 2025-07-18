package services

import (
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

var memoryLock sync.Mutex

func ReserveMemory(pid uint, size int, path string) error {
	// Validar tamaño
	if size < 0 {
		return fmt.Errorf("el tamaño del proceso debe ser mayor a 0 (PID %d)", pid)
	}
	// Calcula cuantas páginas necesita
	pageSize := models.MemoryConfig.PageSize
	pageCount := int(math.Ceil(float64(size) / float64(pageSize)))
	//slog.Debug("Page count calculado", "pid", pid, "size", size, "pageCount", pageCount)

	// Verifico que haya frames libres suficientes
	memoryLock.Lock()
	freeFramesCount := 0
	for _, free := range models.FreeFrames {
		if free {
			freeFramesCount++
		}
	}
	memoryLock.Unlock()
	if freeFramesCount < pageCount {
		err := fmt.Errorf("no hay suficientes frames libres para el proceso PID %d", pid)
		slog.Error(err.Error())
		return err
	}

	// Cargar instrucciones del script
	err := GetInstructionsByName(pid, path, models.InstructionsMap, models.MemoryConfig.ScriptsPath)
	if err != nil {
		slog.Error("Error al cargar instrucciones", "error", err)
		return fmt.Errorf("falló la carga de instrucciones para el PID %d", pid)
	}

	// Inicializar la tabla multinivel vacía para el proceso
	err = initializePageTables(pid)
	if err != nil {
		slog.Error("Error inicializando tablas de páginas", "pid", pid, "error", err)
		return fmt.Errorf("falló la creación de estructuras administrativas para el PID %d", pid)
	}

	// Asignar frames a la cantidad de páginas necesarias
	assignedFrames := make([]int, 0, pageCount)
	for i := 0; i < pageCount; i++ {
		frame := AllocateFrame()
		if frame == -1 {
			slog.Error("Error asignando frame", "pid", pid)
			releaseFrames(pid, assignedFrames)
			return fmt.Errorf("falló la asignación de frames para PID %d", pid)
		}
		assignedFrames = append(assignedFrames, frame) // si fue asignado bien se guarda en la lista assignedFrames

		err = MapPageToFrame(pid, i, frame)
		if err != nil {
			slog.Error("Error mapeando página a frame", "pid", pid, "error", err)
			releaseFrames(pid, assignedFrames)
			return fmt.Errorf("falló el mapeo de páginas para PID %d", pid)
		}
	}

	NewProcess(pid, size, pageCount, assignedFrames)

	slog.Debug("PCB registrado", slog.Int("pid", int(pid)), slog.Int("pages", pageCount), slog.Int("size", size))
	if _, exists := models.ProcessTable[pid]; !exists {
		slog.Warn("Proceso no registrado en ProcessTable al finalizar ReserveMemory", "pid", pid)
	}
	return nil
}

// Crea los niveles necesarios en la estructura multinivel hasta insertar una entrada en el último nivel.
func MapPageToFrame(pid uint, pageNumber int, frame int) error {
	// Obtener configuración
	numLevels := models.MemoryConfig.NumberOfLevels
	entriesPerLevel := models.MemoryConfig.EntriesPerPage
	// Obtener los índices que se usan para navegar por cada nivel de la tabla
	indices := getPageIndices(pageNumber, numLevels, entriesPerLevel)
	memoryLock.Lock()
	defer memoryLock.Unlock()

	current := models.PageTables[pid]
	if current == nil {
		return fmt.Errorf("tabla de páginas no inicializada para PID %d", pid)
	}

	// Recorrer niveles excepto el último
	for level := 0; level < numLevels-1; level++ {
		idx := indices[level]

		if current.SubTables == nil {
			current.SubTables = make(map[int]*models.PageTableLevel)
		}
		if _, exists := current.SubTables[idx]; !exists {
			current.SubTables[idx] = &models.PageTableLevel{
				IsLeaf:    false,
				SubTables: make(map[int]*models.PageTableLevel),
				Entry:     nil,
			}
		}

		// Avanza al siguiente nivel
		current = current.SubTables[idx]
	}

	// Último nivel (hoja): insertar PageEntry
	lastIdx := indices[numLevels-1]
	if _, exists := current.SubTables[lastIdx]; exists {
		return fmt.Errorf("entrada de página ya mapeada para página %d (PID %d)", pageNumber, pid)
	}
	current.SubTables[lastIdx] = &models.PageTableLevel{
		IsLeaf: true,
		Entry: &models.PageEntry{
			Frame:    frame,
			Presence: true,
			Use:      false,
			Modified: false,
		},
	}
	//slog.Debug(fmt.Sprintf("Tabla de páginas %v",current))
	slog.Debug("MapPageToFrame", "pid", pid, "page", pageNumber, "frame", frame, "direccion_base", frame*models.MemoryConfig.PageSize)
	return nil
}

func initializePageTables(pid uint) error {
	memoryLock.Lock()
	defer memoryLock.Unlock()
	// Verificar que no exista una tabla ya creada para este proceso
	if _, exists := models.PageTables[pid]; exists {
		return fmt.Errorf("ya existe una tabla de páginas para el PID %d", pid)
	}
	// Crear recursivamente la raíz de la tabla multinivel
	root := createPageTableLevel(1, models.MemoryConfig.NumberOfLevels)
	if root == nil {
		return fmt.Errorf("falló la creación de la tabla de páginas para PID %d", pid)
	}
	models.PageTables[pid] = root
	return nil
}

// Crea recursivamente una tabla de páginas multinivel.
func createPageTableLevel(currentLevel, maxLevels int) *models.PageTableLevel {
	level := &models.PageTableLevel{
		IsLeaf:    currentLevel == maxLevels,
		Entry:     nil,
		SubTables: make(map[int]*models.PageTableLevel),
	}

	return level
}

func NewProcess(pid uint, size int, pageCount int, assignedFrames []int) {
	memoryLock.Lock()
	defer memoryLock.Unlock()
	pages := make([]models.PageEntry, pageCount)
	for i := 0; i < pageCount; i++ {
		pages[i] = models.PageEntry{
			Frame:    assignedFrames[i],
			Presence: true,
			Use:      false,
			Modified: false,
		}
	}
	ProcessTableLock.Lock()
	models.ProcessTable[pid] = &models.Process{
		Pid:     pid,
		Size:    size,
		Pages:   pages,
		Metrics: &models.Metrics{},
	}
	ProcessTableLock.Unlock()
	models.ProcessMetrics[pid] = &models.Metrics{}
	models.ProcessFramesTable[pid] = &models.ProcessFrames{
		PID:    pid,
		Frames: assignedFrames,
	}
	slog.Info(fmt.Sprintf("Proceso %d agregado a ProcessTable con %d páginas y tamaño %d", pid, pageCount, size))
}

func releaseFrames(pid uint, frames []int) {
	memoryLock.Lock()
	defer memoryLock.Unlock()

	for _, f := range frames {
		models.FreeFrames[f] = true
	}

	delete(models.PageTables, pid)
	delete(models.InstructionsMap, pid)
	delete(models.ProcessFramesTable, pid)
	delete(models.ProcessTable, pid)
	delete(models.ProcessMetrics, pid)

	slog.Warn("Rollback de proceso ejecutado", "pid", pid)
}

func SearchFrame(pid uint, pageNumber int) int {
	memoryLock.Lock()
	defer memoryLock.Unlock()

	// Obtener la raíz de la tabla multinivel para el PID
	pageTableRoot, exists := models.PageTables[pid]
	if !exists {
		slog.Warn("Tabla de páginas no encontrada para PID", "pid", pid)
		return -1
	}

	slog.Debug("SearchFrame recibido", "pid", pid, "pageNumber", pageNumber)

	frame, err := getFrameFromPageNumber(pid, pageTableRoot, pageNumber)
	if err == nil && frame != -1 {
		slog.Debug(fmt.Sprintf("Frame enviado a CPU: %d", frame))
		return frame
	}
	slog.Debug("NO SE ENCONTRO EL FRAME")
	// Si no se encontró ningún frame para las páginas consultadas
	return -1
}

func getFrameFromPageNumber(pid uint, root *models.PageTableLevel, pageNumber int) (int, error) {
	entry, err := FindPageEntry(pid, root, pageNumber)
	if err != nil {
		return -1, err
	}
	return entry.Frame, nil
}

func FindPageEntry(pid uint, root *models.PageTableLevel, pageNumber int) (*models.PageEntry, error) {
	currentLevel := root
	indices := getPageIndices(pageNumber, models.MemoryConfig.NumberOfLevels, models.MemoryConfig.EntriesPerPage)

	for i, index := range indices {
		IncrementMetric(pid, "page_table")
		// Si estamos en el último índice, deberíamos encontrar una hoja
		if i == len(indices)-1 {
			nextLevel, exists := currentLevel.SubTables[index]
			if !exists || nextLevel == nil || !nextLevel.IsLeaf {
				return nil, fmt.Errorf("entrada de página no encontrada o no es hoja")
			}
			if nextLevel.Entry != nil && nextLevel.Entry.Presence {
				return nextLevel.Entry, nil
			}
			return nil, fmt.Errorf("entrada de página no presente o no asignada")
		}

		// Navegamos niveles intermedios
		nextLevel, exists := currentLevel.SubTables[index]
		if !exists {
			return nil, fmt.Errorf("nivel %d, índice %d no existe", i, index)
		}
		currentLevel = nextLevel
	}

	return nil, fmt.Errorf("estructura incompleta o sin hoja final")
}

// De acuerdo a la cantidad de niveles de la tabla y la cantidad de entradas por nivel.
func getPageIndices(pageNumber int, levels int, entriesPerLevel int) []int {
	indices := make([]int, levels)
	for i := levels - 1; i >= 0; i-- {
		// Obtener el índice correspondiente al nivel actual
		indices[i] = pageNumber % entriesPerLevel
		pageNumber /= entriesPerLevel
	}
	//slog.Debug("getPageIndices", "pageNumber", pageNumber, "indices", indices)
	return indices
}

func AllocateFrame() int {
	memoryLock.Lock()
	defer memoryLock.Unlock()

	pageSize := models.MemoryConfig.PageSize
	for i, free := range models.FreeFrames {
		if free {
			if i*pageSize >= len(models.UserMemory) {
				// Este frame está marcado como libre pero no cabe en la memoria real
				slog.Warn("Frame libre fuera de rango físico", "frame", i, "direccion_base", i*pageSize)
				continue
			}
			models.FreeFrames[i] = false
			return i
		}
	}
	slog.Error("No hay frames libres disponibles para asignar")
	return -1
}

func UpdatePage() {
	slog.Warn("TODO: implementar lógica")
}
