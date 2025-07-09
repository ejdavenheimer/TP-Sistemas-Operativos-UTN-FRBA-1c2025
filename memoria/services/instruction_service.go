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

var WriteToMemoryMock = WriteToMemory //TODO: lo agregue para los test, chequear si es necesario

func GeInstruction(pid uint, pc uint) (string, bool, error) {
	instructions, exists := models.InstructionsMap[pid]
	if !exists || uint32(pc) >= uint32(len(instructions)) {
		return "", false, errors.New("instruction not found")
	}
	instruction := instructions[pc]
	isLast := pc == uint(len(instructions))-1
	return instruction, isLast, nil
}

// Toma de a un archivo a la vez y guarda las instrucciones en un map l
func GetInstructions(pid uint, path string, instructionsMap map[uint][]string) error {
	data := ExtractInstructions(path)
	if data == nil {
		return fmt.Errorf("no se pudieron cargar las instrucciones desde el archivo")
	}

	InsertData(pid, instructionsMap, data)
	return nil
}

func GetInstructionsByPid(pid uint, path string, instructionsMap map[uint][]string) error {
	path, err := FindScriptByName(path, fmt.Sprintf("%d", pid))
	if err != nil {
		slog.Error(fmt.Sprintf("No se encontró archivo para el ID %d: %v", pid, err))
		return nil
	}
	data := ExtractInstructions(path)
	if data == nil {
		return fmt.Errorf("no se pudieron cargar las instrucciones desde el archivo")
	}

	InsertData(pid, instructionsMap, data)
	return nil
}

// Abre el archivo especificado por la ruta 'path' y guarda su contenido en un slice de bytes.
// Retorna el contenido del archivo como un slice de bytes.
func ExtractInstructions(path string) []byte {
	// Lee el archivo
	file, err := os.ReadFile(path)
	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return nil
	}

	// Ahora 'file' es un slice de bytes con el contenido del archivo
	return file
}

// insertData inserta las instrucciones en la memoria de instrucciones asociadas al PID especificado
func InsertData(pid uint, instructionsMap map[uint][]string, data []byte) {
	// Separar las instrucciones por medio de tokens
	instructions := strings.Split(string(data), "\n")
	cleaned := make([]string, 0, len(instructions))
	for _, instr := range instructions {
		instr = strings.TrimSpace(instr) // elimina \r y espacios sobrantes
		cleaned = append(cleaned, instr)
		IncrementMetric(pid, "fetch")
	}
	// Insertar las instrucciones en la memoria de instrucciones
	instructionsMap[pid] = cleaned
}

func FindScriptByID(dir string, pid string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), pid) {
			return filepath.Join(dir, file.Name()), nil
		}
	}
	return "", fmt.Errorf("no se encontró archivo con ID %s no encontrado", pid)
}

// Busca un archivo por nombre exacto en el directorio dado
func FindScriptByName(dir string, scriptName string) (string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if !file.IsDir() && file.Name() == scriptName {
			return filepath.Join(dir, file.Name()), nil
		}
	}
	return "", fmt.Errorf("no se encontró archivo con nombre %s", scriptName)
}

// Modifico GetInstructions para buscar por nombre en scripts_path
func GetInstructionsByName(pid uint, scriptName string, instructionsMap map[uint][]string, scriptsPath string) error {
	path, err := FindScriptByName(scriptsPath, scriptName)
	if err != nil {
		slog.Error(fmt.Sprintf("No se encontró archivo para el nombre %s: %v", scriptName, err))
		return err
	}
	data := ExtractInstructions(path)
	if data == nil {
		return fmt.Errorf("no se pudieron cargar las instrucciones desde el archivo")
	}

	InsertData(pid, instructionsMap, data)
	return nil
}

func Read(pid uint, physicalAddress int, size int) ([]byte, error) {
	ProcessTableLock.RLock() 
	process, ok := models.ProcessTable[pid]
	ProcessTableLock.RUnlock()
	if !ok {
		return nil, ErrProcessNotFound
	}
	slog.Debug(fmt.Sprintf("PID: %d - Dirección física solicitada: %d - Tamaño: %d\n", pid, physicalAddress, size))

	if size <= 0 || size > models.MemoryConfig.PageSize {
		return nil, ErrInvalidRead
	}

	maxAddress := len(process.Pages) * models.MemoryConfig.PageSize
	if physicalAddress < 0 || physicalAddress+size > maxAddress {
		return nil, ErrMemoryViolation
	}

	data, err := readFromMemory(physicalAddress, size)
	if err != nil {
		return nil, err
	}

	UpdatePageBit(pid, physicalAddress, "use")
	IncrementMetric(pid, "reads")

	return data, nil
}

func readFromMemory(physicalAddress int, size int) ([]byte, error) {
	// Verifico que la dirección y el tamaño estén dentro del rango válido
	if physicalAddress < 0 || physicalAddress+size > len(models.UserMemory) {
		return nil, fmt.Errorf("dirección fuera de rango")
	}

	// Copio la porción de memoria solicitada
	data := make([]byte, size)
	copy(data, models.UserMemory[physicalAddress:physicalAddress+size])

	return data, nil
}

func WriteToMemory(pid uint, physicalAddress int, data []byte) (int,error) {
	slog.Debug(fmt.Sprintf("WriteToMemory solicitado - PID: %d - Dirección física: %d - Bytes: %d", pid, physicalAddress, len(data)))
	// Verificar que el proceso existe
	ProcessTableLock.RLock() 
	_, ok := models.ProcessTable[pid]
	ProcessTableLock.RUnlock()
	if !ok {
		return -1, fmt.Errorf("proceso %d no encontrado", pid)
	}

	memoryLock.Lock()
	defer memoryLock.Unlock()

	// Validación límites de memoria
	if physicalAddress < 0 || physicalAddress+len(data) > len(models.UserMemory) {
		return -1, fmt.Errorf("violación de memoria física en dirección %d", physicalAddress)
	}

	// Escribir en memoria física
	copy(models.UserMemory[physicalAddress:physicalAddress+len(data)], data)

	UpdatePageBit(pid, physicalAddress, "use")
	UpdatePageBit(pid, physicalAddress, "modified")
	IncrementMetric(pid, "writes")

	frame := physicalAddress / models.MemoryConfig.PageSize
	slog.Debug(fmt.Sprintf("WriteToMemory - PID: %d - Dir: %d - Bytes: %d", pid, physicalAddress, len(data)))
	return frame, nil
}

func UpdatePageBit(pid uint, physicalAddress int, bit string) {
	pageNumber := physicalAddress / models.MemoryConfig.PageSize
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
		slog.Warn(fmt.Sprintf("El bit es desconocido: %s", bit))
	}
}

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
