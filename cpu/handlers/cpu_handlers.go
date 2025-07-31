package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/services"
	kernelModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

func ExecuteProcessHandler(cpuConfig *models.Config) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var instructionRequest memoriaModel.InstructionRequest

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

		executionStartTime := time.Now()

		var isFinished, isBlocked, isSyscall bool = false, false, false
		var syscallRequest kernelModel.SyscallRequest

		models.InterruptControl.PID = int(instructionRequest.Pid)
		for !models.InterruptControl.InterruptPending && !isFinished && !isBlocked {
			fetchResult := services.Fetch(request, cpuConfig)

			if fetchResult.Instruction == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}

			services.DecodeAndExecute(instructionRequest.Pid, fetchResult.Instruction, cpuConfig, &isFinished, &isBlocked, &isSyscall, &syscallRequest)

			if !isFinished && fetchResult.IsLast {
				isFinished = fetchResult.IsLast
			}

			request.PC = int(models.CpuRegisters.PC)
		}

		executionTime := float32(time.Since(executionStartTime).Milliseconds())

		response := kernelModel.PCBExecuteRequest{
			PID:           request.Pid,
			PC:            request.PC,
			ExecutionTime: executionTime,
		}

		if models.InterruptControl.InterruptPending {
			response.StatusCodePCB = kernelModel.NeedInterrupt
			slog.Debug("ExecuteProcessHandler need interrupt")
		}

		if isBlocked && !isSyscall {
			response.StatusCodePCB = kernelModel.NeedReplan
			slog.Debug("ExecuteProcessHandler need re-plan")
		}

		if isSyscall && !isFinished {
			response.StatusCodePCB = kernelModel.NeedExecuteSyscall
			response.SyscallRequest = syscallRequest
			slog.Debug("ExecuteProcessHandler need execute syscall")
		}

		if isFinished && !isBlocked && !models.InterruptControl.InterruptPending {
			if isBlocked {
				response.SyscallRequest = syscallRequest
			}
			response.StatusCodePCB = kernelModel.NeedFinish
			slog.Debug("ExecuteProcessHandler need finish")
		}

		models.InterruptControl.InterruptPending = false
		server.SendJsonResponse(w, response)
	}
}

func InterruptProcessHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var pid int
		if err := json.NewDecoder(r.Body).Decode(&pid); err != nil {
			http.Error(w, "PID inválido", http.StatusBadRequest)
			return
		}

		slog.Debug("Interrupción recibida", slog.Int("pid", pid))
		slog.Info("##Llega interrupción al puerto Interrupt")

		if pid == models.InterruptControl.PID {
			slog.Debug("Interrupción informada al cpu", slog.Int("pid", pid))
			models.InterruptControl.InterruptPending = true
			w.WriteHeader(http.StatusOK)
		} else {
			slog.Error("No existe ese proceso ejecutandose en esta cpu para interrumpirlo", slog.Int("pid", pid))
			w.WriteHeader(http.StatusBadRequest)
		}

	}
}
