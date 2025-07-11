package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

type UserMemoryRequest struct {
	PID  uint `json:"PID"`
	Size int  `json:"Size"`
}

// Envía una solicitud a Memoria para verificar capacidad disponible
func CheckUserMemoryCapacity(pid uint, processSize int) error {
	slog.Debug("Verificando capacidad de memoria para proceso", "PID", pid, "Size", processSize)

	request := UserMemoryRequest{
		PID:  pid,
		Size: processSize,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error al serializar UserMemoryRequest: %v", err)
	}

	resp, err := client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/capacidadUserMemory", body)
	if err != nil {
		return fmt.Errorf("error enviando request a Memoria: %v", err)
	}
	defer resp.Body.Close()

	// DoRequest ya maneja el StatusCode internamente, pero si retorna la respuesta
	// significa que hubo un error de status diferente a 200
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("memoria insuficiente para proceso PID %d, status: %d", pid, resp.StatusCode)
	}

	slog.Debug("Memoria confirmó capacidad disponible", "PID", pid)
	return nil
}
