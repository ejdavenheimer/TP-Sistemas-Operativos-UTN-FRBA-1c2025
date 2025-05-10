package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/services"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

func ExecuteProcessHandler(cpuConfig *models.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var instructionRequest memoriaModel.InstructionRequest

		// Decodifica el request (codificado en formato json).
		err := json.NewDecoder(r.Body).Decode(&instructionRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		request := memoriaModel.InstructionRequest{
			Pid:      instructionRequest.Pid,
			PC:       instructionRequest.PC,
			PathName: instructionRequest.PathName,
		}
		models.CpuRegisters.PC = uint(request.PC)
		var isFinished bool = false
		for !models.InterruptControl.InterruptPending && !isFinished {
			fetchResult := services.Fetch(request, cpuConfig)
			if fetchResult == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			services.DecodeAndExecute(instructionRequest.Pid, fetchResult, cpuConfig, &isFinished)
			request.PC = int(models.CpuRegisters.PC)
		}

		models.InterruptControl.InterruptPending = false
		//w.WriteHeader(http.StatusOK)
	}
}

// TODO: deprecado, borrar!
func ExecuteHandler(cpuConfig *models.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var instructionRequest memoriaModel.InstructionRequest

		// Decodifica el request (codificado en formato json).
		err := json.NewDecoder(r.Body).Decode(&instructionRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		instructionResponse := services.GetInstruction(instructionRequest, cpuConfig)
		if instructionResponse.Instruction == nil {
			http.Error(w, "No instruction found", http.StatusBadRequest)
			return
		}

		instructions, _ := instructionResponse.Instruction[uint(instructionRequest.Pid)]
		models.CpuRegisters.PC = 0
		var isFinished bool = false
		for _, instr := range instructions {
			services.DecodeAndExecute(instructionRequest.Pid, instr, cpuConfig, &isFinished)
		}
	}
}

func InterruptProcessHandler(cpuConfig *models.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var pid int
		if err := json.NewDecoder(r.Body).Decode(&pid); err != nil {
			http.Error(w, "PID inválido", http.StatusBadRequest)
			return
		}

		slog.Info("Interrupción recibida", slog.Int("pid", pid))

		if pid == models.InterruptControl.PID {
			slog.Info("Interrupción informada al cpu", slog.Int("pid", pid))
			models.InterruptControl.InterruptPending = true
			w.WriteHeader(http.StatusOK)
		} else {
			slog.Error("No existe ese proceso ejecutandose en esta cpu para interrumpirlo", slog.Int("pid", pid))
			w.WriteHeader(http.StatusBadRequest)
		}

	}
}
