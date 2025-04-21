package handlers

import (
	"encoding/json"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/services"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"log/slog"
	"net/http"
	"strings"
)

func ExecuteHandler(cpuConfig *models.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var instructionRequest memoriaModel.InstructionRequest

		// Decodifica el request (codificado en formato json).
		err := json.NewDecoder(r.Body).Decode(&instructionRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		instruction := services.GetInstruction(instructionRequest, cpuConfig)
		value := strings.Split(instruction.Instruction, " ")

		switch value[0] {
		case "IO":
			services.ExecuteIO(value[1], value[0:], cpuConfig)
		default:
			slog.Error("error: instrucción inválida")
		}
	}
}
