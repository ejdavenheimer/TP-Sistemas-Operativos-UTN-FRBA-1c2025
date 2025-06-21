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
	//TODO: revisar para que se pueda pasar cualquiera de los dos formatos
	//NO borrar el comentario de ConfigPath
	ConfigPath = "kernel/configs/kernel.json" //"./configs/kernel.json"
	LogPath    = "./logs/kernel.log"          //"./kernel.log"
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

	config.InitConfig(ConfigPath, &models.KernelConfig)
	log.InitLogger(LogPath, models.KernelConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Kernel: %d", models.KernelConfig.PortKernel))

	go services.MediumTermScheduler()
	go services.StartShortTermScheduler()
	go services.StartScheduler()

	// Iniciar el proceso
	pcb, err = services.InitProcess(pseudocodeFile, processSize, additionalArgs)
	if err != nil {
		slog.Error("Error al iniciar proceso", "err", err)
		return
	}

	/* ----------> ENDPOINTS <----------*/
	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Kernel"))
	http.HandleFunc("GET /kernel", handlers.HandshakeHandler("Kernel en funcionamiento 🚀"))
	http.HandleFunc("GET /kernel/dispositivos-conectados", kernelHandler.GetDevicesMapHandlers())
	http.HandleFunc("GET /kernel/cpus-conectadas", kernelHandler.GetCpuMapHandlers())
	http.HandleFunc("GET /kernel/proceso", kernelHandler.GetProcessHandler())
	http.HandleFunc("GET /kernel/procesos", kernelHandler.GetAllHandler())
	http.HandleFunc("PUT /kernel/proceso", kernelHandler.UpdateProcessHandler())
	http.HandleFunc("POST /kernel/proceso", kernelHandler.AddProcessHandler())
	http.HandleFunc("POST /kernel/dispositivos", kernelHandler.ConnectIoHandler())
	http.HandleFunc("POST /kernel/dispositivo-finalizado", kernelHandler.FinishDeviceHandler())
	http.HandleFunc("POST /kernel/syscall", kernelHandler.ExecuteSyscallHandler())
	http.HandleFunc("POST /kernel/cpus", kernelHandler.ConnectCpuHandler())
	http.HandleFunc("POST /kernel/informar-io-finalizada", kernelHandler.FinishExecIOHandler())
	http.HandleFunc("POST /kernel/mandar-interrupcion-a-cpu", kernelHandler.SendInterruptionHandler)

	//Planificador de larzo plazo
	http.HandleFunc("POST /kernel/finalizarProceso", kernelHandler.FinishProcessHandler)
	http.HandleFunc("POST /kernel/ejecutarProceso", kernelHandler.ExecuteProcessHandler)

	//Inicialización del servidor
	err = server.InitServer(models.KernelConfig.PortKernel)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
