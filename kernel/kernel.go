package main

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
)

const (
	ConfigPath = "./configs/kernel.json"
	LogPath    = "kernel.log"
)

func main() {
	config.InitConfig(ConfigPath, &models.KernelConfig)
	log.InitLogger(LogPath, models.KernelConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Kernel: %d", models.KernelConfig.PortKernel))

	http.HandleFunc("/kernel", handlers.HandshakeHandler("Kernel en funcionamiento 🚀"))

	err := server.InitServer(models.KernelConfig.PortKernel)
	if err != nil {
		slog.Error("error initializing server: %v", err)
		panic(err)
	}
}
