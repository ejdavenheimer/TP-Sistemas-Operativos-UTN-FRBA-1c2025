package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	kernelModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// --- Funciones de Ciclo de Instrucción ---

func Fetch(request memoriaModel.InstructionRequest, cpuConfig *models.Config) memoriaModel.InstructionResponse {
	query := fmt.Sprintf("memoria/instruccion?pid=%d&pc=%d", request.Pid, request.PC)
	slog.Info(fmt.Sprintf("## PID: <%d> - FETCH - <%d>", request.Pid, request.PC))
	response, err := client.DoRequest(cpuConfig.PortMemory, cpuConfig.IpMemory, "GET", query, nil)

	var instructionResponse memoriaModel.InstructionResponse
	if err != nil || response.StatusCode != http.StatusOK {
		slog.Error("Error en Fetch al comunicarse con Memoria", "error", err)
		return instructionResponse
	}
	defer response.Body.Close()

	responseBody, _ := io.ReadAll(response.Body)
	json.Unmarshal(responseBody, &instructionResponse)
	slog.Debug(fmt.Sprintf("Instrucción recibida: %s", instructionResponse.Instruction))
	return instructionResponse
}

func DecodeAndExecute(pid uint, instruction string, cpuConfig *models.Config, isFinished *bool, isBlocked *bool, isSyscall *bool, syscallRequest *kernelModel.SyscallRequest) {
	parts := strings.Split(instruction, " ")
	instructionType := parts[0]

	executeReq := models.ExecuteInstructionRequest{
		Pid:    pid,
		Values: parts,
	}

	switch instructionType {
	case "NOOP":
		ExecuteNoop(executeReq)
	case "WRITE":
		ExecuteWrite(executeReq)
	case "READ":
		ExecuteRead(executeReq)
	case "GOTO":
		ExecuteGoto(executeReq)
	case "INIT_PROC":
		slog.Info(fmt.Sprintf("## PID: <%d> - Ejecutando: <%s> - <%s> <%s>", pid, parts[0], parts[1], parts[2]))
		syncSyscallReq := kernelModel.SyscallRequest{
			Pid:    pid,
			Type:   "INIT_PROC",
			Values: parts[1:],
		}
		body, _ := json.Marshal(syncSyscallReq)
		_, err := client.DoRequest(cpuConfig.PortKernel, cpuConfig.IpKernel, "POST", "kernel/syscall/init_proc", body)
		if err != nil {
			slog.Error("Fallo al ejecutar syscall INIT_PROC", "error", err)
		}
		increase_PC()

	case "IO", "DUMP_MEMORY":
		slog.Info(fmt.Sprintf("## PID: <%d> - Ejecutando: <%s>", pid, instruction))
		syscallRequest.Pid = pid
		syscallRequest.Type = instructionType
		syscallRequest.Values = parts[1:]
		Cache.RemoveProcessFromCache(pid)
		RemoveTLBEntriesByPID(pid)
		*isBlocked = true
		*isSyscall = true
		increase_PC()

	case "EXIT":
		slog.Info(fmt.Sprintf("## PID: <%d> - Ejecutando: <%s>", pid, instruction))
		RemoveTLBEntriesByPID(pid)
		Cache.RemoveProcessFromCache(pid)
		*isFinished = true

	default:
		slog.Error(fmt.Sprintf("Instrucción desconocida: %s", instructionType))
		increase_PC()
	}
}

// --- Implementación de Instrucciones ---

func ExecuteNoop(request models.ExecuteInstructionRequest) {
	slog.Info(fmt.Sprintf("## PID: <%d> - Ejecutando: <%s>", request.Pid, request.Values[0]))
	increase_PC()
}

func ExecuteWrite(request models.ExecuteInstructionRequest) {
	slog.Info(fmt.Sprintf("## PID: <%d> - Ejecutando: <%s> - <%s> <%s>", request.Pid, request.Values[0], request.Values[1], request.Values[2]))
	logicalAddress, err := strconv.Atoi(request.Values[1])
	if err != nil {
		slog.Error("Dirección lógica inválida en WRITE", "error", err)
		increase_PC()
		return
	}
	value := request.Values[2]
	physicalAddress := TranslateAddress(request.Pid, logicalAddress)
	if physicalAddress == -1 {
		slog.Warn("Instrucción WRITE no puede continuar: dirección inválida.")
		increase_PC()
		return
	}

	if IsEnabled() {
		pageNumber := logicalAddress / models.MemConfig.PageSize
		offset := logicalAddress % models.MemConfig.PageSize
		_, found := Cache.Get(request.Pid, pageNumber)
		if !found {
			content := getPageFromMemory(request.Pid, pageNumber, physicalAddress, "Escritura")
			if content == nil {
				increase_PC()
				return
			}
			frame := physicalAddress / models.MemConfig.PageSize
			Cache.Put(request.Pid, pageNumber, frame, content)
		}
		entryKey := getEntryKey(request.Pid, pageNumber)
		idx := Cache.PageMap[entryKey]
		entry := &Cache.Entries[idx]
		copy(entry.Content[offset:], []byte(value))
		entry.ModifiedBit = true
		entry.UseBit = true
		slog.Info(fmt.Sprintf("## PID: <%d> - ACCIÓN: <ESCRIBIR> - DIRECCIÓN FISICA: <%d> - Valor: <%s>", request.Pid, physicalAddress, value))
		increase_PC()
		return
	}

	writeReq := memoriaModel.WriteRequest{
		Pid:             request.Pid,
		PhysicalAddress: physicalAddress,
		Data:            []byte(value),
	}
	body, _ := json.Marshal(writeReq)
	_, err = client.DoRequest(models.CpuConfig.PortMemory, models.CpuConfig.IpMemory, "POST", "memoria/write", body)
	if err != nil {
		slog.Error("Fallo la escritura en Memoria", "error", err)
	}
	increase_PC()
}

func ExecuteRead(request models.ExecuteInstructionRequest) {
	slog.Info(fmt.Sprintf("## PID: <%d> - Ejecutando: <%s> - <%s> <%s>", request.Pid, request.Values[0], request.Values[1], request.Values[2]))
	logicalAddress, err := strconv.Atoi(request.Values[1])
	if err != nil {
		slog.Error("Dirección lógica inválida en READ", "error", err)
		increase_PC()
		return
	}
	size, err := strconv.Atoi(request.Values[2])
	if err != nil {
		slog.Error("Tamaño inválido en READ", "error", err)
		increase_PC()
		return
	}

	physicalAddress := TranslateAddress(request.Pid, logicalAddress)
	if physicalAddress == -1 {
		slog.Warn("Instrucción READ no puede continuar: dirección inválida.")
		increase_PC()
		return
	}

	if IsEnabled() {
		pageNumber := logicalAddress / models.MemConfig.PageSize
		offset := logicalAddress % models.MemConfig.PageSize
		content, found := Cache.Get(request.Pid, pageNumber)
		if !found {
			content = getPageFromMemory(request.Pid, pageNumber, physicalAddress, "Lectura")
			if content == nil {
				increase_PC()
				return
			}
			frame := physicalAddress / models.MemConfig.PageSize
			Cache.Put(request.Pid, pageNumber, frame, content)
		}
		data := content[offset : offset+size]
		cleanData := bytes.Trim(data, "\x00")
		slog.Info(fmt.Sprintf("## PID: <%d> - ACCIÓN: <LEER> - DIRECCIÓN FISICA: <%d> - Valor: <%s>", request.Pid, physicalAddress, string(cleanData)))
		increase_PC()
		return
	}

	readRequest := memoriaModel.ReadRequest{
		Pid:             request.Pid,
		PhysicalAddress: physicalAddress,
		Size:            size,
	}
	jsonBody, _ := json.Marshal(readRequest)
	response, err := client.DoRequest(models.CpuConfig.PortMemory, models.CpuConfig.IpMemory, "POST", "memoria/leerMemoria", jsonBody)
	if err != nil {
		increase_PC()
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		slog.Error("Error al leer desde Memoria", "status", response.StatusCode, "body", string(body))
		increase_PC()
		return
	}

	var memoryResponse struct {
		Content []byte `json:"content"`
	}
	json.NewDecoder(response.Body).Decode(&memoryResponse)
	cleanData := bytes.Trim(memoryResponse.Content, "\x00")
	slog.Info(fmt.Sprintf("## PID: %d - ACCIÓN: LEER - DIRECCIÓN FISICA: %d - Valor: %s", request.Pid, physicalAddress, string(cleanData)))
	increase_PC()
}

func ExecuteGoto(request models.ExecuteInstructionRequest) {
	slog.Info(fmt.Sprintf("## PID: <%d> - Ejecutando: <%s> - <%s>", request.Pid, request.Values[0], request.Values[1]))
	value, _ := strconv.Atoi(request.Values[1])
	if value > 0 {
		models.CpuRegisters.PC = uint(value - 1)
	} else {
		models.CpuRegisters.PC = 0
	}
}

// --- Funciones Auxiliares ---

func increase_PC() {
	models.CpuRegisters.PC++
	slog.Debug(fmt.Sprintf("Valor actual de PC: %d", models.CpuRegisters.PC))
}

func getPageFromMemory(pid uint, pageNumber int, physicalAddress int, operacion string) []byte {
	type PageRequest struct {
		PID             uint   `json:"pid"`
		PageNumber      int    `json:"page_number"`
		PhysicalAddress int    `json:"physicalAddress"`
		Operacion       string `json:"operacion"`
	}
	type PageResponse struct {
		Content []byte `json:"content"`
	}

	req := PageRequest{PID: pid, PhysicalAddress: physicalAddress, PageNumber: pageNumber, Operacion: operacion}
	body, err := json.Marshal(req)
	if err != nil {
		slog.Error("Error serializando request JSON", "error", err)
		return nil
	}

	resp, err := client.DoRequest(models.CpuConfig.PortMemory, models.CpuConfig.IpMemory, "POST", "memoria/leerPagina", body)
	if err != nil {
		slog.Error("Error solicitando página completa", "error", err)
		return nil
	}
	defer resp.Body.Close()

	var res PageResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		slog.Error("Error decodificando página", "error", err)
		return nil
	}

	return res.Content
}
