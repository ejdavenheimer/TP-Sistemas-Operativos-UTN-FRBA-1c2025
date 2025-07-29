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
	defer models.ProcessDataLock.RUnlock()

	instructions, exists := models.InstructionsMap[pid]
	if !exists || pc >= uint(len(instructions)) {
		return "", false, errors.New("instruction not found or PC out of bounds")
	}
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

	models.UMemoryLock.RLock()
	defer models.UMemoryLock.RUnlock()

	if physicalAddress < 0 || physicalAddress+size > len(models.UserMemory) {
		return nil, ErrMemoryViolation
	}

	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

	process, ok := models.ProcessTable[pid]
	if !ok {
		return nil, ErrProcessNotFound
	}

	data := make([]byte, size)
	copy(data, models.UserMemory[physicalAddress:physicalAddress+size])

	pageSize := models.MemoryConfig.PageSize
	startFrame := physicalAddress / pageSize
	endFrame := (physicalAddress + size - 1) / pageSize

	// CORRECCIÓN: Iteramos sobre los frames afectados para encontrar el nro de página lógico correspondiente a cada uno.
	for frame := startFrame; frame <= endFrame; frame++ {
		// Búsqueda DIRECTA en las páginas del proceso
		for i, page := range process.Pages {
			if page.Frame == frame {
				// Acceso directo a la entrada de página - SIN métricas ni delays
				if pageEntry := getPageEntryDirect(pid, i); pageEntry != nil {
					UpdatePageBit(pageEntry, "use")
				}
				break // Encontrado, siguiente frame
			}
		}
	}
	IncrementMetric(pid, "reads")

	return data, nil
}

func WriteToMemory(pid uint, physicalAddress int, data []byte) error {
	models.UMemoryLock.Lock()
	defer models.UMemoryLock.Unlock()

	if physicalAddress < 0 || physicalAddress+len(data) > len(models.UserMemory) {
		return ErrMemoryViolation
	}

	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

	process, ok := models.ProcessTable[pid]
	if !ok {
		return ErrProcessNotFound
	}

	copy(models.UserMemory[physicalAddress:physicalAddress+len(data)], data)

	pageSize := models.MemoryConfig.PageSize
	startFrame := physicalAddress / pageSize
	endFrame := (physicalAddress + len(data) - 1) / pageSize

	// CORRECCIÓN: Iteramos sobre los frames afectados para encontrar el nro de página lógico correspondiente a cada uno.
	for frame := startFrame; frame <= endFrame; frame++ {
		// Búsqueda DIRECTA en las páginas del proceso
		for i, page := range process.Pages {
			if page.Frame == frame {
				// Acceso directo a la entrada de página - SIN métricas ni delays
				if pageEntry := getPageEntryDirect(pid, i); pageEntry != nil {
					UpdatePageBit(pageEntry, "use")
					UpdatePageBit(pageEntry, "modified")
				}
				break // Encontrado, siguiente frame
			}
		}
	}
	IncrementMetric(pid, "writes")

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
