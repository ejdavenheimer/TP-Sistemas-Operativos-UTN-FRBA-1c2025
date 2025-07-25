package services

import (
	"log/slog"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

var (
	nextPID  uint = 0
	pidMutex sync.Mutex
)

// generatePID crea un ID de proceso único de forma segura.
func generatePID() uint {
	pidMutex.Lock()
	defer pidMutex.Unlock()
	pid := nextPID
	nextPID++
	return pid
}

// InitProcess se encarga de crear la estructura PCB, asignarle un PID único
// y moverlo al estado NEW para que el Planificador de Largo Plazo lo gestione.
func InitProcess(pseudocodeFile string, processSize int, additionalArgs []string) (*models.PCB, error) {

	pseudocodeName := filepath.Base(pseudocodeFile)

	parentPID := -1
	if len(additionalArgs) > 0 {
		parentPIDVal, err := strconv.Atoi(additionalArgs[0])
		if err == nil {
			parentPID = parentPIDVal
		} else {
			slog.Warn("No se pudo parsear ParentPID, utilizando valor por defecto -1")
		}
	}

	pcb := &models.PCB{
		PID:            generatePID(),
		ParentPID:      parentPID,
		PC:             0,
		ME:             make(map[models.Estado]int),
		MT:             make(map[models.Estado]time.Duration),
		PseudocodePath: pseudocodeName,
		Size:           processSize,
		RafagaEstimada: float32(models.KernelConfig.InitialEstimate),
		SwapRequested:  false, // Inicializamos el flag
	}

	// Usamos la nueva función para mover el proceso al estado NEW y a su cola.
	TransitionProcessState(pcb, models.EstadoNew)

	// Damos una señal al planificador de largo plazo para que se active y evalúe la cola NEW.
	StartLongTermScheduler()

	return pcb, nil
}
