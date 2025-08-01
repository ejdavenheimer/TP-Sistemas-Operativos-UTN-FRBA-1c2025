package services

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

var (
	ErrProcessNotFound = errors.New("proceso no encontrado")
	ErrMemoryViolation = errors.New("violacion de memoria")
	ErrInvalidRead     = errors.New("lectura invalida")
)

func GeInstruction(pid uint, pc uint) (string, bool, error) {
	models.ProcessDataLock.RLock()
	instructions, exists := models.InstructionsMap[pid]
	if !exists || pc >= uint(len(instructions)) {
		return "", false, errors.New("instruction not found or PC out of bounds")
	}
	defer models.ProcessDataLock.RUnlock()
	instruction := instructions[pc]
	isLast := pc == uint(len(instructions))-1
	IncrementMetric(pid, "fetch")
	return instruction, isLast, nil
}

func GetInstructionsByName(pid uint, scriptName string, instructionsMap map[uint][]string, scriptsPath string) error {
	path, err := FindScriptByName(scriptsPath, scriptName)
	if err != nil {
		slog.Error(fmt.Sprintf("No se encontró archivo de script '%s': %v", scriptName, err))
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error(fmt.Sprintf("No se pudo leer el archivo de script '%s': %v", path, err))
		return err
	}

	instructions := strings.Split(string(data), "\n")
	cleaned := make([]string, 0, len(instructions))
	for _, instr := range instructions {
		instr = strings.TrimSpace(instr)
		if instr != "" { // Evitar añadir líneas vacías
			cleaned = append(cleaned, instr)
		}
	}

	models.ProcessDataLock.Lock()
	instructionsMap[pid] = cleaned
	models.ProcessDataLock.Unlock()

	return nil
}

func FindScriptByName(dir string, scriptName string) (string, error) {
	filePath := filepath.Join(dir, scriptName)
	if _, err := os.Stat(filePath); err == nil {
		return filePath, nil
	}
	return "", fmt.Errorf("no se encontró archivo con nombre %s", scriptName)
}

func Read(pid uint, physicalAddress int, size int) ([]byte, error) {
	if size <= 0 {
		return nil, ErrInvalidRead
	}

	pageSize := models.MemoryConfig.PageSize
	startFrame := physicalAddress / pageSize
	endFrame := (physicalAddress + size - 1) / pageSize

	models.ProcessDataLock.RLock()
	process, processExists := models.ProcessTable[pid]
	if !processExists {
		models.ProcessDataLock.RUnlock()
		return nil, ErrProcessNotFound
	}
	pagesCopy := make([]models.PageEntry, len(process.Pages))
	copy(pagesCopy, process.Pages)
	models.ProcessDataLock.RUnlock()

	var data []byte
	models.UMemoryLock.RLock()
	if physicalAddress < 0 || physicalAddress+size > len(models.UserMemory) {
		models.UMemoryLock.RUnlock()
		return nil, ErrMemoryViolation
	}

	// Lectura inmediata - operación más crítica
	data = make([]byte, size)
	copy(data, models.UserMemory[physicalAddress:physicalAddress+size])
	models.UMemoryLock.RUnlock()

	affectedPages := make([]int, 0, endFrame-startFrame+1)

	// Buscar páginas afectadas usando la copia local (sin locks)
	for frame := startFrame; frame <= endFrame; frame++ {
		for i, page := range pagesCopy {
			if page.Frame == frame {
				affectedPages = append(affectedPages, i)
				break // Encontrado, continuar con siguiente frame
			}
		}
	}

	// Actualizar bits de uso para todas las páginas afectadas
	for _, pageNumber := range affectedPages {
		if pageEntry := getPageEntryDirect(pid, pageNumber); pageEntry != nil {
			UpdatePageBit(pageEntry, "use")
		}
	}

	models.ProcessDataLock.Lock()
	IncrementMetric(pid, "reads")
	models.ProcessDataLock.Unlock()

	return data, nil
}

func WriteToMemory(pid uint, physicalAddress int, data []byte) error {
	if len(data) == 0 {
		return nil // Nada que escribir
	}
	pageSize := models.MemoryConfig.PageSize
	startFrame := physicalAddress / pageSize
	endFrame := (physicalAddress + len(data) - 1) / pageSize

	models.UMemoryLock.Lock()
	slog.Debug("UMemoryLock lockeado WRITE")
	if physicalAddress < 0 || physicalAddress+len(data) > len(models.UserMemory) {
		models.UMemoryLock.Unlock()
		return ErrMemoryViolation
	}
	// Escritura inmediata - operación más crítica
	copy(models.UserMemory[physicalAddress:physicalAddress+len(data)], data)
	models.UMemoryLock.Unlock()

	models.ProcessDataLock.RLock() // Solo lectura para verificar existencia
	process, ok := models.ProcessTable[pid]
	if !ok {
		models.ProcessDataLock.RUnlock()
		return ErrProcessNotFound
	}

	// Copiar páginas localmente para minimizar tiempo con lock
	pagesCopy := make([]models.PageEntry, len(process.Pages))
	copy(pagesCopy, process.Pages)
	models.ProcessDataLock.RUnlock()

	affectedPages := make([]int, 0, endFrame-startFrame+1)

	// Buscar páginas afectadas usando la copia local
	for frame := startFrame; frame <= endFrame; frame++ {
		for i, page := range pagesCopy {
			if page.Frame == frame {
				affectedPages = append(affectedPages, i)
				break
			}
		}
	}

	// 5. ACTUALIZAR BITS DE PÁGINAS (operaciones rápidas)
	for _, pageNumber := range affectedPages {
		if pageEntry := getPageEntryDirect(pid, pageNumber); pageEntry != nil {
			UpdatePageBit(pageEntry, "use")
			UpdatePageBit(pageEntry, "modified")
		}
	}

	// 6. ACTUALIZAR MÉTRICAS (lock mínimo)
	models.ProcessDataLock.Lock()
	IncrementMetric(pid, "writes")
	models.ProcessDataLock.Unlock()

	return nil
}

func getPageEntryDirect(pid uint, pageNumber int) *models.PageEntry {
	pageTableRoot, exists := models.PageTables[pid]
	if !exists {
		return nil
	}

	// Calcular índices pero navegar DIRECTAMENTE sin delays ni métricas
	indices := getPageIndices(pageNumber, models.MemoryConfig.NumberOfLevels, models.MemoryConfig.EntriesPerPage)
	currentLevel := pageTableRoot

	// Navegación rápida SIN delays
	for i, index := range indices {
		nextLevel, exists := currentLevel.SubTables[index]
		if !exists {
			return nil
		}

		if i == len(indices)-1 {
			if nextLevel.IsLeaf && nextLevel.Entry != nil && nextLevel.Entry.Presence {
				return nextLevel.Entry
			}
			return nil
		}

		currentLevel = nextLevel
	}

	return nil
}

// UpdatePageBit ahora recibe el número de página lógico correcto.
func UpdatePageBit(entry *models.PageEntry, bit string) {
	switch bit {
	case "presence_on":
		entry.Presence = true
	case "presence_off":
		entry.Presence = false
	case "use":
		entry.Use = true
	case "modified":
		entry.Modified = true
	default:
		slog.Warn(fmt.Sprintf("Intento de actualizar bit desconocido: %s", bit))
	}
}

// IncrementMetric debe ser llamado dentro de un lock de ProcessDataLock
func IncrementMetric(pid uint, metric string) {
	if m, ok := models.ProcessMetrics[pid]; ok {
		switch metric {
		case "reads":
			m.Reads++
		case "writes":
			m.Writes++
		case "swap_out":
			m.SwapsOut++
		case "swap_in":
			m.SwapsIn++
		case "page_table":
			m.PageTableAccesses++
		case "fetch":
			m.InstructionFetches++
		default:
			slog.Warn(fmt.Sprintf("Métrica desconocida: %s", metric))
		}
	}
}

// **NUEVA FUNCIÓN AUXILIAR**
// findPageNumberByFrame realiza la búsqueda inversa: dado un frame, encuentra a qué página lógica pertenece para un PID.
// Esta función debe ser llamada dentro de un lock de ProcessDataLock.
//func findPageNumberByFrame(pid uint, frameIndex int) (int, bool) {
//	process, exists := models.ProcessTable[pid]
//	if !exists {
//		return -1, false
//	}
// Esta es una búsqueda lineal, pero dado el bajo número de páginas por proceso en las pruebas,
// es suficientemente eficiente y mucho más simple que mantener un mapa inverso.
//		if page.Frame == frameIndex {
//			return i, true
//		}
//	}
//	return -1, false
//}
