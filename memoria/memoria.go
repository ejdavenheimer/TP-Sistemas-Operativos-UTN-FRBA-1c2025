package main

import (
	"fmt"
	"log/slog"
	"net/http"

	memoryHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

const (
	ConfigPath = "memoria/configs/memoria.json"
	LogPath    = "./logs/memoria.log"
)

func main() {
	config.InitConfig(ConfigPath, &models.MemoryConfig)
	log.InitLogger(LogPath, models.MemoryConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Memory: %d", models.MemoryConfig.PortMemory))

	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Memoria"))
	http.HandleFunc("GET /memoria", handlers.HandshakeHandler("Memoria en funcionamiento 🚀"))
	http.HandleFunc("GET /memoria/instrucciones", memoryHandler.GetInstructionsHandler())

	//Liberar espacio  de memoria de un PCB
	http.HandleFunc("POST /memoria/liberarpcb", memoryHandler.DeleteContextHandler)

	err := server.InitServer(models.MemoryConfig.PortMemory)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
