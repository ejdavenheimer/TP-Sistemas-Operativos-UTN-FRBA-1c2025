package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
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
		PseudocodePath: pseudocodeName,
		Size:           processSize,
		RafagaEstimada: float32(models.KernelConfig.InitialEstimate),
	}

	TransitionState(pcb, models.EstadoNew)

	models.QueueNew.Add(pcb)
	StartLongTermScheduler()
	slog.Info(fmt.Sprintf("## PID %d Se crea el proceso - Estado : NEW", pid))

	return pcb, nil
}

// Envía una solicitud a Memoria para asignar espacio y cargar instrucciones
func requestMemorySpace(pid uint, processSize int) (bool, error) {
	// Extraigo solo el nombre del archivo, por si viene una ruta
	//pseudocodeName := filepath.Base(pseudocodePath)
	request := models.MemoryRequest{
		PID:  pid,
		Size: processSize,
	}

	body, err := json.Marshal(request)
	if err != nil {
		return false, fmt.Errorf("error al serializar MemoryRequest: %v", err)
	}

	resp, err := client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/capacidadUserMemory", body)
	if err != nil {
		return false, fmt.Errorf("error enviando request a Memoria: %v", err)
	}

	if resp.StatusCode == http.StatusOK {
		return true, nil
	}

	if resp.StatusCode == http.StatusInsufficientStorage {
		return false, nil
	}

	return false, fmt.Errorf("respuesta inesperada de Memoria: código %d", resp.StatusCode)
}

func generatePID() uint {
	pidMutex.Lock()
	defer pidMutex.Unlock()

	pid := nextPID
	nextPID++
	return pid
}
