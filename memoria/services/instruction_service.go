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

	data := make([]byte, size)
	copy(data, models.UserMemory[physicalAddress:physicalAddress+size])

	// Actualizar métricas y bits de página fuera del lock de memoria si es posible
	// para reducir la contención, pero aquí es más simple hacerlo dentro.
	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

	if _, ok := models.ProcessTable[pid]; !ok {
		return nil, ErrProcessNotFound
	}

	pageSize := models.MemoryConfig.PageSize
	startPage := physicalAddress / pageSize
	endPage := (physicalAddress + size - 1) / pageSize
	for page := startPage; page <= endPage; page++ {
		UpdatePageBit(pid, page, "use")
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

	copy(models.UserMemory[physicalAddress:physicalAddress+len(data)], data)

	models.ProcessDataLock.Lock()
	defer models.ProcessDataLock.Unlock()

	if _, ok := models.ProcessTable[pid]; !ok {
		return ErrProcessNotFound
	}

	pageSize := models.MemoryConfig.PageSize
	startPage := physicalAddress / pageSize
	endPage := (physicalAddress + len(data) - 1) / pageSize
	for page := startPage; page <= endPage; page++ {
		UpdatePageBit(pid, page, "use")
		UpdatePageBit(pid, page, "modified")
	}
	IncrementMetric(pid, "writes")

	return nil
}

// UpdatePageBit ahora recibe pageNumber directamente
func UpdatePageBit(pid uint, pageNumber int, bit string) {
	entry, err := FindPageEntry(pid, models.PageTables[pid], pageNumber)
	if err != nil {
		slog.Warn(fmt.Sprintf("No se pudo actualizar bit '%s' para PID %d, página %d: %v", bit, pid, pageNumber, err))
		return
	}

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
