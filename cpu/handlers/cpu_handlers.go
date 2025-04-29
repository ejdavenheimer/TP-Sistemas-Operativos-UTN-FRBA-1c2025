package handlers

import (
	"encoding/json"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/services"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"net/http"
)

func ExecuteHandlerV2(cpuConfig *models.Config) func(http.ResponseWriter, *http.Request) {
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
		for !models.InterruptPending && !isFinished {
			fetchResult := services.Fetch(request, cpuConfig)
			if fetchResult == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			services.DecodeAndExecute(instructionRequest.Pid, fetchResult, cpuConfig, &isFinished)
			request.PC = int(models.CpuRegisters.PC)
		}

		models.InterruptPending = false
		//w.WriteHeader(http.StatusOK)
	}
}

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

		instructions, _ := instructionResponse.Instruction[1]
		models.CpuRegisters.PC = 0
		var isFinished bool = false
		for _, instr := range instructions {
			services.DecodeAndExecute(instructionRequest.Pid, instr, cpuConfig, &isFinished)
		}
	}
}
