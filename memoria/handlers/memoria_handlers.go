package handlers

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"net/http"
	"strconv"
)

func GetInstructionsHandler(configPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		pidStr := queryParams.Get("pid")
		pathName := queryParams.Get("pathName")

		pid, _ := strconv.ParseInt(pidStr, 10, 64)

		path := configPath + pathName
		instruction := models.InstructionResponse{
			Instruction: services.GetIoInstruction(uint(pid), path),
		}
		server.SendJsonResponse(w, instruction)
	}
}
