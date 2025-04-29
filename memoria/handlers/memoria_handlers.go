package handlers

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
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
		services.GetInstructions(uint(pid), path, models.InstructionsMap)
		instruction := models.InstructionsResponse{
			Instruction: models.InstructionsMap,
		}
		slog.Debug(fmt.Sprintf("Se envierán %d instrucciones para ejecutar.", len(instruction.Instruction[uint(pid)])))
		server.SendJsonResponse(w, instruction)
	}
}

func GetInstructionHandler(configPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		pidStr := queryParams.Get("pid")
		pcStr := queryParams.Get("pc")
		pathName := queryParams.Get("pathName")

		pid, _ := strconv.ParseInt(pidStr, 10, 64)
		pc, _ := strconv.ParseInt(pcStr, 10, 64)
		path := configPath + pathName

		instructionResult, err := services.GeInstruction(uint(pid), uint(pc), path)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			slog.Error(fmt.Sprintf("error: %s", err.Error()))
			return
		}

		instruction := models.InstructionResponse{
			Instruction: instructionResult,
		}
		slog.Info(fmt.Sprintf("## PID: <%d> - Obtener instrucción: <%d> - Instrucción: %s", pid, pc, instruction.Instruction))
		server.SendJsonResponse(w, instruction)
	}
}
