package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	kernelHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

const (
	ConfigPath = "kernel/configs/kernel.json"
	LogPath    = "./logs/kernel.log"
)

var pcb *models.PCB

func main() {
	if len(os.Args) < 3 {
		slog.Error("Faltan los parametros necesarios [archivo_pseudocódigo] y [tamanio_proceso]")
		return
	}

	//Parametros
	pseudocodeFile := os.Args[1]
	processSize, err := strconv.Atoi(os.Args[2])
	if err != nil {
		slog.Error(fmt.Sprintf("Error al convertir el tamaño del proceso: %v", err))
		return
	}
	additionalArgs := os.Args[3:]

	//Test de funciones
	//services.TestQueueNew()
	//services.TestFinalizarProceso()

	config.InitConfig(ConfigPath, &models.KernelConfig)
	log.InitLogger(LogPath, models.KernelConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Kernel: %d", models.KernelConfig.PortKernel))
    
	go services.StartScheduler()
	
	// Iniciar el proceso
	pcb, err = services.InitProcess(pseudocodeFile, processSize, additionalArgs)
	if err != nil {
		slog.Error("Error al iniciar proceso", "err", err)
		return
	}

	//TODO: borrar esto
	//device := models2.Device{Ip: "127.0.0.1", Port: 8003, Name: "Test"}
	//services.SleepDevice(0, 25000, device)

	/* ----------> ENDPOINTS <----------*/
	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Kernel"))
	http.HandleFunc("GET /kernel", handlers.HandshakeHandler("Kernel en funcionamiento 🚀"))
	http.HandleFunc("GET /kernel/dispositivos-conectados", kernelHandler.GetDevicesMapHandlers())
	http.HandleFunc("POST /kernel/dispositivos", kernelHandler.ConnectIoHandler())
	http.HandleFunc("POST /kernel/syscall", kernelHandler.ExecuteSyscallHandler())

	//Planificador de larzo plazo
	http.HandleFunc("POST /kernel/finalizarProceso", kernelHandler.FinishProcessHandler)

	//Iniciacialización del servidor
	err = server.InitServer(models.KernelConfig.PortKernel)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
