package services

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"log/slog"
	"os"
	"strings"
)

func GetIoInstruction(pid uint, path string) string {
	GetInstructions(pid, path, models.InstructionsMap)
	var result string = ""
	for _, instruction := range models.InstructionsMap[pid] {
		if strings.HasPrefix(instruction, "IO ") {
			result = strings.TrimSpace(instruction)
		}
	}
	return result
}

// Toma de a un archivo a la vez y guarda las instrucciones en un map l
func GetInstructions(pid uint, path string, instructionsMap map[uint][]string) {
	data := ExtractInstructions(path)
	InsertData(pid, instructionsMap, data)
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
