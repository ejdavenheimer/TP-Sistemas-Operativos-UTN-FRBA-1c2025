package handlers

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/services"
	"log/slog"
	"net/http"
	"strings"
)

func ExecuteHandler(cpuConfig *models.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		instruction := services.GetInstruction(cpuConfig)
		value := strings.Split(instruction.Instruction, " ")

		switch value[0] {
		case "IO":
			services.ExecuteIO(value[0], value[0:], cpuConfig)
		default:
			slog.Error("error: instrucción inválida")
		}
	}
}
