package main

import (
	"fmt"
	ioHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
	"os"
)

const (
	ConfigPath = "./configs/io.json"
	LogPath    = "io.log"
)

func main() {
	config.InitConfig(ConfigPath, &models.IoConfig)
	log.InitLogger(LogPath, models.IoConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port IO: %d", models.IoConfig.PortIo))

	ioName := os.Args[1]

	if len(os.Args) < 2 {
		slog.Error("no se indicó el nombre del dispositivo")
		return
	}

	services.ConnectToKernel(ioName, models.IoConfig)

	http.HandleFunc("GET /", handlers.HandshakeHandler(fmt.Sprintf("Bienvenido al módulo de IO - Dispositivo: %s", ioName)))
	http.HandleFunc("GET /io", handlers.HandshakeHandler("IO en funcionamiento 🚀"))
	http.HandleFunc("POST /io", ioHandler.SleepHandler())

	err := server.InitServer(models.IoConfig.PortIo)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
