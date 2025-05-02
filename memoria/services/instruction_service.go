package services

import (
	"errors"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"log/slog"
	"os"
	"strings"
)

func GeInstruction(pid uint, pc uint, path string) (string, error) {
	GetInstructions(pid, path, models.InstructionsMap)
	instructions, exists := models.InstructionsMap[pid]
	if !exists || uint32(pc) >= uint32(len(instructions)) {
		return "", errors.New("instruction not found")
	}
	instruction := instructions[pc]
	return instruction, nil
}

// Toma de a un archivo a la vez y guarda las instrucciones en un map l
func GetInstructions(pid uint, path string, instructionsMap map[uint][]string) error{
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
		slog.Error(fmt.Sprintf("Error in extracting instructions: %s", err))
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
