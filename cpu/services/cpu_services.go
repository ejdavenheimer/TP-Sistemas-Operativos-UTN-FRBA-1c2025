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
	"os"
	"strconv"
	"strings"
	"bytes"
	"encoding/base64"
)

func GetInstruction(request memoriaModel.InstructionRequest, cpuConfig *models.Config) memoriaModel.InstructionsResponse {
	//Envia la request de conexion a Kernel
	query := fmt.Sprintf("memoria/instrucciones?pid=%d&pathName=%s", request.Pid, request.PathName)
	response, err := client.DoRequest(cpuConfig.PortMemory, cpuConfig.IpMemory, "GET", query, nil)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	var instruction memoriaModel.InstructionsResponse
	err = json.Unmarshal(responseBody, &instruction)
	if err != nil {
		slog.Error(fmt.Sprintf("error parseando el JSON: %v", err))
		return memoriaModel.InstructionsResponse{}
	}

	slog.Debug(fmt.Sprintf("Instrucción recibida: %v", instruction.Instruction))

	return instruction
}

func ExecuteSyscall(syscallRequest kernelModel.SyscallRequest, cpuConfig *models.Config) string {
	//Crea y codifica la request de conexion a Kernel
	body, err := json.Marshal(syscallRequest)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return "error"
	}

	//Envia la request de conexion a Kernel
	response, err := client.DoRequest(cpuConfig.PortKernel, cpuConfig.IpKernel, "POST", "kernel/syscall", body)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return "error"
	}

	defer response.Body.Close()
	responseBody, _ := io.ReadAll(response.Body)

	var kernelResp struct {
		Action string `json:"action"` // e.g., "continue", "block", "exit"
	}

	err = json.Unmarshal(responseBody, &kernelResp)
	if err != nil {
		slog.Error(fmt.Sprintf("error parseando respuesta del kernel: %v", err))
		return "error"
	}

	return kernelResp.Action
}

func ExecuteNoop(request models.ExecuteInstructionRequest) {
	//log obligatorio
	slog.Info(fmt.Sprintf("## PID: %d - Ejecutando: %s", request.Pid, request.Values[0]))
	slog.Debug(fmt.Sprintf("[%d] Instrucción %s", request.Pid, request.Values[0]))
	slog.Debug(fmt.Sprintf("Valor inicial de PC: %d", models.CpuRegisters.PC))

	increase_PC()
}

func ExecuteWrite(request models.ExecuteInstructionRequest) {
	slog.Debug(fmt.Sprintf("[%d] Instrucción %s(%s, %s)", request.Pid, request.Values[0], request.Values[1], request.Values[2]))
	//TODO: implementar lógica para WRITE
	//CONVERSION DIR LOG A FISICA
	logicalAddress, err := strconv.Atoi(request.Values[1])
	if err != nil {
		slog.Error("Dirección lógica inválida")
		return
	}


	if err != nil {
		slog.Error("Error al convertir dirección lógica", "error", err)
		return
	}

	physicalAddress := TranslateAddress(request.Pid, logicalAddress)
	if physicalAddress == -1{
		slog.Warn("Instrucción READ no puede continuar: diección invalida")
		increase_PC()
		return
	}

	//SOLICITUD DE ESCRITURA
	writeReq := models.WriteRequest{
		PID:             request.Pid,
		PhysicalAddress: physicalAddress,
		Data:            request.Values[2],
	}

	body, err := json.Marshal(writeReq)
	if err != nil {
		slog.Error("Error al serializar WriteRequest", "error", err)
		return
	}

	//PETICION HTTP A MEMORIA
	_, err = client.DoRequest(
		models.CpuConfig.PortMemory,
		models.CpuConfig.IpMemory,
		"POST",
		"memoria/write",
		body,
	)
	if err != nil {
		slog.Error("Fallo la escritura en Memoria", "error", err)
		return
	}

	slog.Info(fmt.Sprintf("## PID: %d - ACCIÓN: ESCRIBIR - DIRECCIÓN FISICA: %d - Valor: %s", request.Pid, physicalAddress, writeReq.Data))
	
	increase_PC()
}

func ExecuteRead(request models.ExecuteInstructionRequest) {
	//log obligatorio
	slog.Info(fmt.Sprintf("## PID: %d - Ejecutando: %s - %s %s", request.Pid, request.Values[0], request.Values[1], request.Values[2]))
	slog.Debug(fmt.Sprintf("[%d] Instrucción %s(%s, %s)", request.Pid, request.Values[0], request.Values[1], request.Values[2]))

	logicalAddress, err := strconv.Atoi(request.Values[1])
	if err != nil {
		slog.Error("Dirección lógica inválida")
		return
	}

	size, err := strconv.Atoi(request.Values[2])
	if err != nil {
		slog.Error("Tamaño inválido")
		return
	}

	physicalAddress := TranslateAddress(request.Pid, logicalAddress)
	if physicalAddress == -1{
		slog.Warn("Instrucción READ no puede continuar: diección invalida")
		increase_PC()
		return
	}

	readRequest := models.MemoryReadRequest{
		Pid:             request.Pid,
		PhysicalAddress: physicalAddress,
		Size:            size,
	}

	jsonBody, err := json.Marshal(readRequest)
	if err != nil {
		slog.Error("No se pudo serializar la request a Memoria")
		return
	}

	response, err := client.DoRequest(models.CpuConfig.PortMemory, models.CpuConfig.IpMemory, "POST", "memoria/leerMemoria", jsonBody)
	if err != nil {
		slog.Error("Error al comunicarse con Memoria")
		return
	}

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		slog.Error("Error al leer desde Memoria", "direccion", physicalAddress, "status", response.StatusCode, "body", string(body))
		os.Exit(1)
	}

	var memoryValue []byte
    defer response.Body.Close()
    if err := json.NewDecoder(response.Body).Decode(&memoryValue); err != nil {
	   slog.Error("No se pudo leer el valor de memoria")
	   return
    }
    
    cleanData := bytes.Trim(memoryValue, "\x00")
	dataBase64 := base64.StdEncoding.EncodeToString(memoryValue)
    
    slog.Info(fmt.Sprintf("## PID: %d - ACCIÓN: LEER - DIRECCIÓN FISICA: %d - Valor: %s", request.Pid, physicalAddress, string(cleanData)))
    slog.Debug(fmt.Sprintf("Valor (hex): %x", memoryValue))
	slog.Debug(fmt.Sprintf("Valor (base64): %s", dataBase64))
	increase_PC()
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

func Fetch(request memoriaModel.InstructionRequest, cpuConfig *models.Config) memoriaModel.InstructionResponse {
	query := fmt.Sprintf("memoria/instruccion?pid=%d&pc=%d", request.Pid, request.PC)
	response, err := client.DoRequest(cpuConfig.PortMemory, cpuConfig.IpMemory, "GET", query, nil)

	var instructionResponse memoriaModel.InstructionResponse
	if err != nil || response.StatusCode != http.StatusOK {
		slog.Error(fmt.Sprintf("error: %v", err))
		return instructionResponse
	}

	if response.StatusCode != http.StatusOK {
		slog.Error(fmt.Sprintf("error: %d", response.StatusCode))
		return instructionResponse
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	err = json.Unmarshal(responseBody, &instructionResponse)
	if err != nil {
		slog.Error(fmt.Sprintf("error parseando el JSON: %v", err))
	}

	slog.Debug(fmt.Sprintf("Instrucción recibida: %s", instructionResponse.Instruction))
	return instructionResponse
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
	case "INIT_PROC", "DUMP_MEMORY", "IO":
		syscallRequest = kernelModel.SyscallRequest{
			Pid:    pid,
			Type:   value[0],
			Values: value[1:],
		}
		action := ExecuteSyscall(syscallRequest, cpuConfig)
		switch action {
		case "continue":
			increase_PC()
			*isFinished = false
		case "block", "exit":
			*isFinished = true
		default:
			slog.Error(fmt.Sprintf("acción desconocida de syscall: %v", action))
			*isFinished = true
		}
	case "EXIT":
		syscallRequest = kernelModel.SyscallRequest{
			Pid:  pid,
			Type: value[0],
		}
		action := ExecuteSyscall(syscallRequest, cpuConfig)
		switch action {
		case "exit":
			*isFinished = true
		default:
			slog.Error(fmt.Sprintf("acción desconocida de syscall: %v", action))
			*isFinished = true
		}

	default:
		slog.Error(fmt.Sprintf("Unknown instruction type %s", value[0]))
	}

	//if value[0] != "GOTO" {
	//	models.CpuRegisters.PC++
	//}

	//slog.Debug(fmt.Sprintf("Valor del PC: %d", models.CpuRegisters.PC))
}

func increase_PC() {
	models.CpuRegisters.PC += 1
	slog.Debug(fmt.Sprintf("Valor actual de PC: %d", models.CpuRegisters.PC))
}
