package helpers

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
)

// CreateDirectory crea un directorio en el path especificado.
func CreateDirectory(dir string) {
	err := os.MkdirAll(dir, os.ModePerm)

	if err != nil {
		slog.Error(fmt.Sprintf("Error al crear el directorio %s: %v", dir, err))
		return
	}

	slog.Debug(fmt.Sprintf("Directorio %s creado o ya existía.", dir))
}

// CreateFile crea el archivo
func CreateFile(fileName string, size int) error {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0666)

	if err != nil {
		slog.Error(fmt.Sprintf("Error al crear el archivo: %v", err))
		return err
	}

	defer file.Close()

	// Se ajusta el tamaño del archivo
	err = file.Truncate(int64(size))
	if err != nil {
		slog.Error(fmt.Sprintf("Error al ajustar el tamaño del archivo: %v", err))
		return err
	}

	return nil
}

// InitMemory se encarga de hacer la configuración inicial
func InitMemory(configPath string, logPath string) {
	config.InitConfig(configPath, &models.MemoryConfig)
	log.InitLogger(logPath, models.MemoryConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Memory: %d", models.MemoryConfig.PortMemory))
	models.InstructionsMap = make(map[uint][]string)

	models.UserMemory = make([]byte, models.MemoryConfig.MemorySize) //INICIALIZACIÓN DE MEMORIA
	slog.Debug("Memoria inicializada", "tamaño", len(models.UserMemory))

	// Inicializar FrameTable
	pageSize := models.MemoryConfig.PageSize
	memSize := models.MemoryConfig.MemorySize
	framesCount := memSize / pageSize
	models.FrameTable = make([]models.MemoryFrame, framesCount)
	for i := 0; i < framesCount; i++ {
		models.FrameTable[i] = models.MemoryFrame{
			StartAddr: i * pageSize,
			IsFree:    true,
		}
	}
	slog.Debug("FrameTable inicializado", "cantidad_frames", framesCount)

	CreateDirectory(models.MemoryConfig.DumpPath)
	slog.Debug(fmt.Sprintf("Swap: %s", models.MemoryConfig.SwapFilePath))
	_ = CreateFile(models.MemoryConfig.SwapFilePath, 0) //TODO: revisar, inicialmente va arrancar con tamaño 0
}

func GetDumpName(pid uint) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%d-%s.dmp", pid, timestamp)
}
