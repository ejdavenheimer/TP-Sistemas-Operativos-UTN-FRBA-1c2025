package services

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"log/slog"
	"sync"
	"math"
	"fmt"
)

var memoryLock sync.Mutex
var frameCounter = 0 //cada vez que se consulta frame se incrementa, las direcciones no se repiten

func ReserveMemory(pid uint, size int, path string) error {
	//Calcula cuantas páginas necesita
	// Calcular cuántas páginas necesita
	pageCount := int(math.Ceil(float64(size) / float64(models.MemoryConfig.PageSize)))
	totalMemoryNeeded := pageCount * models.MemoryConfig.PageSize

	// Verificar si hay espacio suficiente en memoria
	if models.NextFreeAddress+totalMemoryNeeded > models.MemoryConfig.MemorySize {
		err := fmt.Errorf("no hay suficiente memoria para el proceso PID %d", pid)
		slog.Error(err.Error())
		return err
	}
	// Cargar instrucciones del script
	err := GetInstructions(pid, path, models.InstructionsMap)
	if err != nil {
		slog.Error("Error al cargar instrucciones", "error", err)
		return fmt.Errorf("falló la carga de instrucciones para el PID %d", pid)
	}
    
	// Dirección base contigua
	baseAddress := models.NextFreeAddress
	models.NextFreeAddress += totalMemoryNeeded

	// Guardar PCB en la tabla de procesos
	models.ProcessTable[int(pid)] = models.Process{
		Pid:         int(pid),
		BaseAddress: baseAddress,
		Size:        totalMemoryNeeded,
	}

	// Crear estructuras de paginación (iniciales vacías, se puede expandir esto)
	err = initializePageTables(pid)
	if err != nil {
		slog.Error("Error inicializando tablas de páginas", "pid", pid, "error", err)
		return fmt.Errorf("falló la creación de estructuras administrativas para el PID %d", pid)
	}

	// Inicializar métricas para el proceso
	initializeMetrics(pid)

	slog.Debug("PCB registrado", slog.Int("pid", int(pid)), slog.Int("base_address", baseAddress), slog.Int("size", size))

	return nil
}

func initializePageTables(pid uint) error {
	// Verificar que no exista una tabla ya creada para este proceso
    if _, exists := models.PageTables[pid]; exists {
        return fmt.Errorf("ya existe una tabla de páginas para el PID %d", pid)
    }

    // Crear raíz
	root := make(map[int]interface{})
	current := root

	for level := 0; level < models.MemoryConfig.NumberOfLevels-1; level++ {
        for i := 0; i < models.MemoryConfig.EntriesPerPage; i++ {
            current[i] = make(map[int]interface{})
        }
        nextLevel, ok := current[0].(map[int]interface{})
        if !ok {
            return fmt.Errorf("fallo al crear el nivel %d para el PID %d", level, pid)
        }
        current = nextLevel
    }

    // Último nivel: asignar marcos iniciales (-1)
    for i := 0; i < models.MemoryConfig.EntriesPerPage; i++ {
        current[i] = -1
    }

    // Registrar en la estructura global
    models.PageTables[pid] = root

    return nil
}

func initializeMetrics(pid uint) {
	models.ProcessMetrics[pid] = &models.Metrics{
		PageTableAccesses:  0,
		InstructionFetches: 0,
		SwapsOut:           0,
		SwapsIn:            0,
		Reads:              0,
		Writes:             0,
	}
}

func SearchFrame(pid uint, entries []int)int{
    frameCounter++
	return frameCounter
}