package handlers

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"net/http"
)

func GetInstructionsHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		instruction := models.InstructionResponse{
			Instruction: services.GetIoInstruction(),
		}
		server.SendJsonResponse(w, instruction)
	}
}
