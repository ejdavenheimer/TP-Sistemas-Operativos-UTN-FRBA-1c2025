package main

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
)

const (
	ConfigPath = "./configs/memoria.json"
	LogPath    = "memoria.log"
)

func main() {
	config.InitConfig(ConfigPath, &models.MemoryConfig)
	log.InitLogger(LogPath, models.MemoryConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Memory: %d", models.MemoryConfig.PortMemory))

	http.HandleFunc("/memoria", handlers.HandshakeHandler("Memoria en funcionamiento 🚀"))

	err := server.InitServer(models.MemoryConfig.PortMemory)
	if err != nil {
		slog.Error("error initializing server: %v", err)
		panic(err)
	}
}
