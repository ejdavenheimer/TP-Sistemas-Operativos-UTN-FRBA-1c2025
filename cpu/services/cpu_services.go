package services

import (
	"encoding/json"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	kernelModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// este servicio realiza la conexión con kernel.
func GetInstruction(request memoriaModel.InstructionRequest, cpuConfig *models.Config) memoriaModel.InstructionsResponse {
	//Envia la request de conexion a Kernel
	query := fmt.Sprintf("memoria/instrucciones?pid=%d&pathName=%s", request.Pid, request.PathName)
	response, err := client.DoRequest(cpuConfig.PortMemory, cpuConfig.IpMemory, "GET", query, nil)

	if err != nil {
		slog.Error("error:", err)
		panic(err)
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	var instruction memoriaModel.InstructionsResponse
	err = json.Unmarshal(responseBody, &instruction)
	if err != nil {
		slog.Error("error parseando el JSON: %v", err)
		return memoriaModel.InstructionsResponse{}
	}

	slog.Debug(fmt.Sprintf("Instrucción recibida: %v", instruction.Instruction))

	return instruction
}

func ExecuteSyscall(syscallRequest kernelModel.SyscallRequest, cpuConfig *models.Config) {
	//Crea y codifica la request de conexion a Kernel
	body, err := json.Marshal(syscallRequest)

	if err != nil {
		slog.Error("error:", err)
		return
	}

	//Envia la request de conexion a Kernel
	_, err = client.DoRequest(cpuConfig.PortKernel, cpuConfig.IpKernel, "POST", "kernel/syscall", body)

	if err != nil {
		slog.Error("error:", err)
		return
	}
}

func ExecuteNoop(request models.ExecuteInstructionRequest) {
	slog.Debug(fmt.Sprintf("[%d] Instrucción %s", request.Pid, request.Values[0]))
	//TODO: implementar lógica para NOOP
}

func ExecuteWrite(request models.ExecuteInstructionRequest) {
	slog.Debug(fmt.Sprintf("[%d] Instrucción %s(%s, %s)", request.Pid, request.Values[0], request.Values[1], request.Values[2]))
	//TODO: implementar lógica para WRITE
}

func ExecuteRead(request models.ExecuteInstructionRequest) {
	slog.Debug(fmt.Sprintf("[%d] Instrucción %s(%s, %s)", request.Pid, request.Values[0], request.Values[1], request.Values[2]))
	//TODO: implementar lógica para READ
}

func ExecuteGoto(request models.ExecuteInstructionRequest) {
	slog.Debug(fmt.Sprintf("[%d] Instrucción %s %s", request.Pid, request.Values[0], request.Values[1]))
	slog.Debug(fmt.Sprintf("Valor anterior de PC: %d", models.CpuRegisters.PC))

	value, _ := strconv.Atoi(request.Values[1])
	result := value - 1

	if value < 0 {
		slog.Error(fmt.Sprintf("El valor de parámetros de GOTO no puede ser negativo"))
		return
	}

	if value == 0 {
		result = 0
	}

	models.CpuRegisters.PC = uint(result)
	slog.Debug(fmt.Sprintf("Valor actual de PC: %d", models.CpuRegisters.PC))
}

func Fetch(request memoriaModel.InstructionRequest, cpuConfig *models.Config) string {
	query := fmt.Sprintf("memoria/instruccion?pid=%d&pc=%d&pathName=%s", request.Pid, request.PC, request.PathName)
	response, err := client.DoRequest(cpuConfig.PortMemory, cpuConfig.IpMemory, "GET", query, nil)

	if err != nil || response.StatusCode != http.StatusOK {
		slog.Error("error:", err)
		return ""
	}

	if response.StatusCode != http.StatusOK {
		slog.Error(fmt.Sprintf("error: %d", response.StatusCode))
		return ""
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	var instructionResponse memoriaModel.InstructionResponse
	err = json.Unmarshal(responseBody, &instructionResponse)
	if err != nil {
		slog.Error(fmt.Sprintf("error parseando el JSON: %v", err))
	}

	slog.Debug(fmt.Sprintf("Instrucción recibida: %s", instructionResponse.Instruction))
	return instructionResponse.Instruction
}

func DecodeAndExecute(pid int, instructions string, cpuConfig *models.Config, isFinished *bool) {
	value := strings.Split(instructions, " ")

	var syscallRequest kernelModel.SyscallRequest
	executeInstructionRequest := models.ExecuteInstructionRequest{
		Pid:    pid,
		Values: value[0:],
	}

	switch value[0] {
	case "NOOP":
		ExecuteNoop(executeInstructionRequest)
	case "WRITE":
		ExecuteWrite(executeInstructionRequest)
	case "READ":
		ExecuteRead(executeInstructionRequest)
	case "GOTO":
		ExecuteGoto(executeInstructionRequest)
	case "IO":
		syscallRequest = kernelModel.SyscallRequest{
			Pid:    pid,
			Type:   value[0],
			Values: value[1:],
		}
		*isFinished = true
		ExecuteSyscall(syscallRequest, cpuConfig)
	case "INIT_PROC", "DUMP_MEMORY", "EXIT":
		syscallRequest = kernelModel.SyscallRequest{
			Pid:    pid,
			Type:   value[0],
			Values: value[1:],
		}
		*isFinished = true
		ExecuteSyscall(syscallRequest, cpuConfig)
	default:
		slog.Error(fmt.Sprintf("Unknown instruction type %s", value[0]))
	}

	if value[0] != "GOTO" {
		models.CpuRegisters.PC++
	}

	slog.Debug(fmt.Sprintf("Valor del PC: %d", models.CpuRegisters.PC))
}
