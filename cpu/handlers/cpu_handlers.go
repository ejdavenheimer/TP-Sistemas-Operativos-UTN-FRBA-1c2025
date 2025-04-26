package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/services"
	kernelModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
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

		instructionResponse := services.GetInstruction(instructionRequest, cpuConfig)
		if instructionResponse.Instruction == nil {
			http.Error(w, "No instruction found", http.StatusBadRequest)
			return
		}

		instructions, _ := instructionResponse.Instruction[1]
		var value []string
		for _, instr := range instructions {
			value = strings.Split(instr, " ")

			var syscallRequest kernelModel.SyscallRequest
			executeInstructionRequest := models.ExecuteInstructionRequest{
				Pid:    instructionRequest.Pid,
				Values: value[0:],
			}

			switch value[0] {
			case "NOOP":
				services.ExecuteNoop(executeInstructionRequest)
			case "WRITE":
				services.ExecuteWrite(executeInstructionRequest)
			case "READ":
				services.ExecuteRead(executeInstructionRequest)
			case "GOTO":
				services.ExecuteGoto(executeInstructionRequest)
			case "IO":
				syscallRequest = kernelModel.SyscallRequest{
					Pid:    instructionRequest.Pid,
					Type:   value[1],
					Values: value[0:],
				}
				services.ExecuteSyscall(syscallRequest, cpuConfig)
			case "INIT_PROC", "DUMP_MEMORY", "EXIT":
				syscallRequest = kernelModel.SyscallRequest{
					Pid:    instructionRequest.Pid,
					Values: value[0:],
				}
				services.ExecuteSyscall(syscallRequest, cpuConfig)
			default:
				slog.Error(fmt.Sprintf("Unknown instruction type %s", value[0]))
			}
		}
	}
}
