package helpers

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"log/slog"
	"os"
	"time"
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
func CreateFile(file string) error {
	_, err := os.OpenFile(file, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)

	if err != nil {
		slog.Error(fmt.Sprintf("Error al crear el archivo: %v", err))
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

	CreateDirectory(models.MemoryConfig.DumpPath)
	slog.Debug(fmt.Sprintf("Swap: %s", models.MemoryConfig.SwapFilePath))
	_ = CreateFile(models.MemoryConfig.SwapFilePath)
}

func GetDumpName(pid uint) string {
	timestamp := time.Now().Format("20060102-150405")
	return fmt.Sprintf("%d-%s.dmp", pid, timestamp)
}
