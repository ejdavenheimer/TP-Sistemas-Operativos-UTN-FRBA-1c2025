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
	ConfigPath = "io/configs/io.json"
//	LogPath    = "io.log"
)

func main() {
	if len(os.Args) < 3 {
		slog.Error("no se indicó el nombre del dispositivo")
		return
	}

	ioName := os.Args[1]
	ioPort := os.Args[2]

	config.InitConfig(ConfigPath, &models.IoConfig)
	models.IoConfig.PortIo, _ = strconv.Atoi(ioPort)

	logPath, err := log.BuildLogPath("io_%s", ioName)
	if err != nil {
		slog.Error("No se pudo construir el log path", "err", err)
		return
	}

	log.InitLogger(logPath, models.IoConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port IO: %d", models.IoConfig.PortIo))

	services.ConnectToKernel(ioName, models.IoConfig)

	http.HandleFunc("GET /", handlers.HandshakeHandler(fmt.Sprintf("Bienvenido al módulo de IO - Dispositivo: %s", ioName)))
	http.HandleFunc("GET /io", handlers.HandshakeHandler("IO en funcionamiento 🚀"))
	http.HandleFunc("POST /io", ioHandler.SleepHandler())

	err = server.InitServer(models.IoConfig.PortIo)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
