package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

var nextPID int

func InitProcess(pseudocodeFile string, processSize int, additionalArgs []string) (*models.PCB, error) {
	pid := generatePID()
	absPath, err := filepath.Abs(pseudocodeFile)
    if err != nil {
        slog.Error("Error al obtener ruta absoluta", "err", err)
        return nil, fmt.Errorf("Error al obtener ruta absoluta: %v", err)
    }

    err = requestMemorySpace(pid, processSize, absPath)
	if err != nil {
		slog.Error("Error al solicitar espacio en memoria", "err", err)
		return nil, fmt.Errorf("Error al solicitar espacio en memoria: %v", err)
	}

	pcb := &models.PCB{
		PID:          pid,
		PC:           0,
		ME:           make(map[models.Estado]int),
		MT:           make(map[models.Estado]time.Duration),
		EstadoActual: models.EstadoNew,
		UltimoCambio: time.Now(),
	}

	updateStateMetrics(pcb, models.EstadoNew)
	slog.Info("##",
	"PID", pcb.PID,
	"Se crea el proceso - Estado", string(pcb.EstadoActual))
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