package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	ioHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

const (
	//TODO: revisar para que se pueda pasar cualquiera de los dos formatos
	//NO borrar el comentario de ConfigPath
	//ConfigPath = "io/configs/io.json" //"./configs/io.json"
// LogPath    = "io.log"
)

func main() {
	if len(os.Args) < 3 {
		slog.Error("no se indicó el nombre del dispositivo")
		return
	}

	models.IoName = os.Args[1]
	ioPort := os.Args[2]
    
	ConfigPath := config.IOConfigPath()
	config.InitConfig(ConfigPath, &models.IoConfig)
	models.IoConfig.PortIo, _ = strconv.Atoi(ioPort)

	logPath, err := log.BuildLogPath("io_%s", models.IoName)
	if err != nil {
		slog.Error("No se pudo construir el log path", "err", err)
		return
	}

	log.InitLogger(logPath, models.IoConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port IO: %d - IO: %s", models.IoConfig.PortIo, models.IoName))

	services.ConnectToKernel(models.IoName, models.IoConfig)

	signal.Notify(models.Shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-models.Shutdown
		slog.Debug("Señal recibida, cerrando módulo IO", "signal", sig)

		// Notifica al Kernel que se cierra este módulo
		services.NotifyDisconnection()

		os.Exit(0)
	}()

	http.HandleFunc("GET /", handlers.HandshakeHandler(fmt.Sprintf("Bienvenido al módulo de IO - Dispositivo: %s", models.IoName)))
	http.HandleFunc("GET /io", handlers.HandshakeHandler("IO en funcionamiento 🚀"))
	http.HandleFunc("POST /io", ioHandler.SleepHandler())

	err = server.InitServer(models.IoConfig.PortIo)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
