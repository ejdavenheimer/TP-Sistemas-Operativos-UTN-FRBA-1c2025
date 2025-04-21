package main

import (
	"fmt"
	kernelHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
)

const (
	ConfigPath = "kernel/configs/kernel.json"
	LogPath    = "./logs/kernel.log"
)

func main() {
	config.InitConfig(ConfigPath, &models.KernelConfig)
	log.InitLogger(LogPath, models.KernelConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Kernel: %d", models.KernelConfig.PortKernel))

	//device := models2.Device{Ip: "127.0.0.1", Port: 8003, Name: "Test"}
	//services.SleepDevice(0, 25000, device)

	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Kernel"))
	http.HandleFunc("GET /kernel", handlers.HandshakeHandler("Kernel en funcionamiento 🚀"))
	http.HandleFunc("GET /kernel/dispositivos-conectados", kernelHandler.GetDevicesMap())
	http.HandleFunc("POST /kernel/dispositivos", kernelHandler.ConnectIoHandler())
	http.HandleFunc("POST /kernel/syscall", kernelHandler.ExecuteSyscallHandler())

	err := server.InitServer(models.KernelConfig.PortKernel)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
