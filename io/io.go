package main

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
)

const (
	ConfigPath = "./configs/io.json"
	LogPath    = "io.log"
)

func main() {
	config.InitConfig(ConfigPath, &models.IoConfig)
	log.InitLogger(LogPath, models.IoConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port IO: %d", models.IoConfig.PortIo))

	http.HandleFunc("/io", handlers.HandshakeHandler("IO en funcionamiento 🚀"))

	err := server.InitServer(models.IoConfig.PortIo)
	if err != nil {
		slog.Error("error initializing server: %v", err)
		panic(err)
	}
}
