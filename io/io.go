package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time" // Importa el paquete time

	ioHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

func main() {
	if len(os.Args) < 3 {
		slog.Error("no se indicó el nombre del dispositivo")
		return
	}

	models.IoName = strings.ToUpper(os.Args[1])
	ioPort := os.Args[2]

	ConfigPath := config.IOConfigPath()
	config.InitConfig(ConfigPath, &models.IoConfig)
	models.IoConfig.PortIo, _ = strconv.Atoi(ioPort)

	logPath, err := log.BuildLogPath("io_%s_%s", models.IoName, ioPort)
	if err != nil {
		slog.Error("No se pudo construir el log path", "err", err)
		return
	}

	log.InitLogger(logPath, models.IoConfig.LogLevel)
	slog.Info(fmt.Sprintf("Port IO: %d - IO: %s", models.IoConfig.PortIo, models.IoName))

	// 1. Definir los handlers ANTES de iniciar el servidor
	http.HandleFunc("GET /", handlers.HandshakeHandler(fmt.Sprintf("Bienvenido al módulo de IO - Dispositivo: %s", models.IoName)))
	http.HandleFunc("GET /io", handlers.HandshakeHandler("IO en funcionamiento 🚀"))
	http.HandleFunc("POST /io", ioHandler.SleepHandler())

	// 2. Iniciar el servidor en una goroutine para que no bloquee
	go func() {
		err_server := server.InitServer(models.IoConfig.PortIo)
		if err_server != nil {
			slog.Error(fmt.Sprintf("error initializing server: %v", err_server))
			panic(err_server)
		}
	}()

	// Damos una pequeña pausa para asegurar que la goroutine del servidor haya iniciado
	time.Sleep(100 * time.Millisecond)

	// 3. AHORA, con el servidor ya escuchando, nos conectamos al Kernel
	services.ConnectToKernel(models.IoName, models.IoConfig)

	// 4. Mantenemos el proceso principal vivo para manejar señales de cierre
	signal.Notify(models.Shutdown, syscall.SIGINT, syscall.SIGTERM)

	sig := <-models.Shutdown
	slog.Debug("Señal recibida, cerrando módulo IO", "signal", sig)

	// Notifica al Kernel que se cierra este módulo
	services.NotifyDisconnection()

	os.Exit(0)
}
