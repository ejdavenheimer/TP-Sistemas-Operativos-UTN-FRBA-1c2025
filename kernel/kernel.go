package main

import (
	"fmt"
	"log/slog"
	"net/http"

	kernelHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

const (
	ConfigPath = "kernel/configs/kernel.json"
	LogPath    = "./logs/kernel.log"
)

func main() {
	//TODO: Test de funciones, despues se deben borrar
	//services.TestQueueNew()
	//services.TestFinalizarProceso()

	config.InitConfig(ConfigPath, &models.KernelConfig)
	log.InitLogger(LogPath, models.KernelConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Kernel: %d", models.KernelConfig.PortKernel))

	/* ----------> ENDPOINTS <----------*/
	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Kernel"))
	http.HandleFunc("GET /kernel", handlers.HandshakeHandler("Kernel en funcionamiento 🚀"))
	http.HandleFunc("GET /kernel/dispositivos-conectados", kernelHandler.GetDevicesMapHandlers())
	http.HandleFunc("POST /kernel/dispositivos", kernelHandler.ConnectIoHandler())
	http.HandleFunc("POST /kernel/syscall", kernelHandler.ExecuteSyscallHandler())

	//Planificador de larzo plazo
	http.HandleFunc("POST /kernel/finalizarProceso", kernelHandler.FinishProcessHandler)

	//Iniciacialización del servidor
	err := server.InitServer(models.KernelConfig.PortKernel)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
