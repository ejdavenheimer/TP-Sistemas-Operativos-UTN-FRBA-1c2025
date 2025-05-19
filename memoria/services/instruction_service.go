package services

import (
	"errors"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func GeInstruction(pid uint, pc uint, path string) (string, bool, error) {
	err := GetInstructionsByPid(pid, path, models.InstructionsMap)
	if err != nil {
		return "", false, errors.New("instruction not found")
	}

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
		return fmt.Errorf("No se pudieron cargar las instrucciones desde el archivo")
	}

	InsertData(pid, instructionsMap, data)
	return nil
}

func GetInstructionsByPid(pid uint, path string, instructionsMap map[uint][]string) error {
	path, err := FindScriptByID(path, fmt.Sprintf("%d", pid))
	if err != nil {
		slog.Error(fmt.Sprintf("No se encontró archivo para el ID %d: %v", pid, err))
		return nil
	}
	data := ExtractInstructions(path)
	if data == nil {
		return fmt.Errorf("No se pudieron cargar las instrucciones desde el archivo")
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

func Read(physicalAddress int, size int) (string, error) {
	//if physicalAddress < 0 || physicalAddress+size > len(models.UserMemory) {
	//	return "", fmt.Errorf("acceso fuera de los límites de memoria")
	//}
	//return string(models.UserMemory[physicalAddress : physicalAddress+size]), nil
	return "DATOS FIJOS PARA PRUEBA", nil
}