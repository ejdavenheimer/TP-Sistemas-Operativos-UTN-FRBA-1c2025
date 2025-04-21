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
)

// este servicio realiza la conexión con kernel.
func GetInstruction(request memoriaModel.InstructionRequest, cpuConfig *models.Config) memoriaModel.InstructionResponse {
	//Envia la request de conexion a Kernel
	query := fmt.Sprintf("memoria/instrucciones?pid=%d&pathName=%s", request.Pid, request.PathName)
	response, err := client.DoRequest(cpuConfig.PortMemory, cpuConfig.IpMemory, "GET", query, nil)

	if err != nil {
		slog.Error("error:", err)
		panic(err)
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	var instruction memoriaModel.InstructionResponse
	err = json.Unmarshal(responseBody, &instruction)
	if err != nil {
		slog.Error("error parseando el JSON: %v", err)
	}

	slog.Debug(fmt.Sprintf("Instrucción recibida: %s", instruction.Instruction))

	return instruction
}

func ExecuteIO(syscallRequest kernelModel.SyscallRequest, cpuConfig *models.Config) {
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
