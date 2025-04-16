package main

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
)

const (
	ConfigPath = "./configs/cpu.json"
	LogPath    = "cpu.log"
)

func main() {
	config.InitConfig(ConfigPath, &models.CpuConfig)
	log.InitLogger(LogPath, models.CpuConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port cpu: %d", models.CpuConfig.PortCpu))

	var cpuNumber int = 1 //TODO: revisar de donde sacamos el número de CPU => nombre de archivo de config?

	http.HandleFunc("GET /", handlers.HandshakeHandler(fmt.Sprintf("Bienvenido al módulo de CPU%d", cpuNumber)))
	http.HandleFunc("GET /cpu", handlers.HandshakeHandler("Cpu en funcionamiento 🚀"))

	err := server.InitServer(models.CpuConfig.PortCpu)
	if err != nil {
		slog.Error("error initializing server: %v", err)
		panic(err)
	}
}
