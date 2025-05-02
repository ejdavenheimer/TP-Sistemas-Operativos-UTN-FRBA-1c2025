package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	"os"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

var nextPID int

func InitProcess(pseudocodeFile string, processSize int, additionalArgs []string) (*models.PCB, error) {
	_, err := os.Stat(pseudocodeFile)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Error("El archivo pseudocódigo no existe", "path", pseudocodeFile)
			return nil, fmt.Errorf("El archivo pseudocódigo no existe: %v", err)
		}
		slog.Error("Error al verificar archivo pseudocódigo", "path", pseudocodeFile, "err", err)
		return nil, fmt.Errorf("Error al verificar archivo pseudocódigo: %v", err)
	}

	pid := generatePID()

	pcb := &models.PCB{
		PID:           pid,
		PC:            0,
		ME:            make(map[models.Estado]int),
		MT:            make(map[models.Estado]time.Duration),
		EstadoActual:  models.EstadoNew,
		UltimoCambio:  time.Now(),
		PseudocodePath: pseudocodeFile,
		Size:          processSize,
	}

	models.QueueNew.Add(*pcb)
	slog.Info("Proceso agregado a cola NEW", "PID", pid)

	return pcb, nil
}
// Envía una solicitud a Memoria para asignar espacio y cargar instrucciones
func requestMemorySpace(pid int, processSize int, pseudocodePath string) error {
    request := models.MemoryRequest{
        PID: pid,
        Size: processSize,
        Path: pseudocodePath,
    }

    body, err := json.Marshal(request)
    if err != nil {
        return fmt.Errorf("Error al serializar MemoryRequest: %v", err)
    }

    _, err = client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/cargarpcb", body)
    if err != nil {
        return fmt.Errorf("Error enviando request a Memoria: %v", err)
    }

    return nil
}


func generatePID() int {
    nextPID++
    return nextPID - 1
}
func updateStateMetrics(pcb *models.PCB, estado models.Estado) {
    // Incrementa el contador de veces que el proceso estuvo en ese estado
    pcb.ME[estado]++

    // Actualiza el tiempo que el proceso ha estado en este estado
    duration := time.Since(pcb.UltimoCambio)
    pcb.MT[estado] += duration
}