package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strconv"
	"time"

	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

var (
	nextPID  uint = 0
	pidMutex sync.Mutex
)

func InitProcess(pseudocodeFile string, processSize int, additionalArgs []string) (*models.PCB, error) {
	// Extraigo solo el nombre del archivo, por si viene una ruta
	pseudocodeName := filepath.Base(pseudocodeFile)

	// Ya no verifico existencia local, la verificación será en Memoria

	parentPID := -1 // Valor por defecto (para el primer proceso o proceso raíz)
	if len(additionalArgs) > 0 {
		parentPIDVal, err := strconv.Atoi(additionalArgs[0])
		if err != nil {
			slog.Warn("No se pudo parsear ParentPID, utilizando valor por defecto -1")
			parentPID = -1
		} else {
			parentPID = parentPIDVal
		}
	}

	pid := generatePID()

	pcb := &models.PCB{
		PID:            pid,
		ParentPID:      parentPID,
		PC:             0,
		ME:             make(map[models.Estado]int),
		MT:             make(map[models.Estado]time.Duration),
		EstadoActual:   models.EstadoNew,
		UltimoCambio:   time.Now(),
		PseudocodePath: pseudocodeName,
		Size:           processSize,
		RafagaEstimada: float32(models.KernelConfig.InitialEstimate),
	}

	models.QueueNew.Add(pcb)
	StartLongTermScheduler()
	slog.Info(fmt.Sprintf("## PID %d Se crea el proceso - Estado : NEW", pid))

	return pcb, nil
}

// Envía una solicitud a Memoria para asignar espacio y cargar instrucciones
func requestMemorySpace(pid uint, processSize int, pseudocodePath string) error {
	// Extraigo solo el nombre del archivo, por si viene una ruta
	pseudocodeName := filepath.Base(pseudocodePath)
	request := models.MemoryRequest{
		PID:  pid,
		Size: processSize,
		Path: pseudocodeName,
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

func generatePID() uint {
	pidMutex.Lock()
	defer pidMutex.Unlock()

	pid := nextPID
	nextPID++
	return pid
}
