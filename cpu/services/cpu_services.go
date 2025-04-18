package services

import (
	"encoding/json"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	kernelModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
	"io"
	"log/slog"
)

// este servicio realiza la conexión con kernel.
func GetInstruction(cpuConfig *models.Config) memoriaModel.InstructionResponse {
	//Envia la request de conexion a Kernel
	response, err := client.DoRequest(cpuConfig.PortMemory, cpuConfig.IpMemory, "GET", "memoria/instrucciones", nil)

	if err != nil {
		slog.Error("error:", err)
		panic(err)
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Info("Response: %s", string(responseBody))

	var instruction memoriaModel.InstructionResponse
	err = json.Unmarshal(responseBody, &instruction)
	if err != nil {
		slog.Error("error parseando el JSON: %v", err)
	}

	slog.Info("Instrucción recibida:", instruction.Instruction)

	return instruction
}

func ExecuteIO(ioName string, values []string, cpuConfig *models.Config) {
	//Crea y codifica la request de conexion a Kernel
	syscallRequest := kernelModel.SyscallRequest{
		Type:   ioName,
		Values: values,
	}
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
