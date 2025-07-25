package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// CheckUserMemoryCapacity envía una solicitud a Memoria para verificar si hay
// espacio suficiente para un proceso antes de intentar cargarlo.
func CheckUserMemoryCapacity(pid uint, processSize int) error {
	slog.Debug("Verificando capacidad de memoria para proceso", "PID", pid, "Size", processSize)

	request := struct {
		PID  uint `json:"PID"`
		Size int  `json:"Size"`
	}{
		PID:  pid,
		Size: processSize,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("error al serializar la solicitud de capacidad: %v", err)
	}

	url := fmt.Sprintf("http://%s:%d/memoria/capacidadUserMemory", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("error enviando la solicitud a Memoria: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Usamos StatusNoContent (204) como señal de "no hay espacio", no como un error.
		if resp.StatusCode == http.StatusNoContent {
			return fmt.Errorf("memoria insuficiente")
		}
		return fmt.Errorf("respuesta inesperada de Memoria: código %d", resp.StatusCode)
	}

	slog.Debug("Memoria confirmó capacidad disponible", "PID", pid)
	return nil
}
