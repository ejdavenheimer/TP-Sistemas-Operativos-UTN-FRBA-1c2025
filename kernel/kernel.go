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

func main() {
	if len(os.Args) < 4 {
		slog.Error("Faltan los parámetros necesarios: [archivo_pseudocódigo] [tamanio_proceso] [archivo_config]")
		return
	}

	// --- 1. Inicialización ---
	ConfigPath := os.Args[3]
	LogPath, err := log.BuildLogPath("kernel")
	if err != nil {
		slog.Error(fmt.Sprintf("No se pudo preparar el archivo de log: %v", err))
		return
	}

	config.InitConfig(ConfigPath, &models.KernelConfig)
	log.InitLogger(LogPath, models.KernelConfig.LogLevel)

	slog.Info(fmt.Sprintf("Kernel escuchando en el puerto: %d", models.KernelConfig.PortKernel))

	// --- 2. Inicio de Planificadores ---
	go services.StartScheduler()      // Inicia el PLP (esperará el Enter).
	go services.ShortTermScheduler()  // Inicia el PCP (esperará notificaciones).
	go services.MediumTermScheduler() // Inicia el PMP (esperará notificaciones y timers).

	// --- 3. Creación del Proceso Inicial ---
	pseudocodeFile := os.Args[1]
	processSize, err := strconv.Atoi(os.Args[2])
	if err != nil {
		slog.Error(fmt.Sprintf("Error al convertir el tamaño del proceso: %v", err))
		return
	}
	additionalArgs := os.Args[4:]

	_, err = services.InitProcess(pseudocodeFile, processSize, additionalArgs)
	if err != nil {
		slog.Error("Error al iniciar el primer proceso", "err", err)
		return
	}

	// --- 4. Registro de Endpoints HTTP ---
	// Handshakes básicos
	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Kernel"))

	// Conexión de recursos
	http.HandleFunc("POST /kernel/cpus", kernelHandler.ConnectCpuHandler())
	http.HandleFunc("POST /kernel/dispositivos", kernelHandler.ConnectIoHandler())

	// Syscalls y notificaciones
	http.HandleFunc("POST /kernel/syscall/init_proc", kernelHandler.InitProcSyscallHandler())
	http.HandleFunc("POST /kernel/informar-io-finalizada", kernelHandler.FinishIoHandler())
	// Endpoint para manejar la desconexión de un dispositivo de I/O
	http.HandleFunc("POST /kernel/dispositivo-finalizado", kernelHandler.DisconnectIoHandler())

	// --- 5. Arranque del Servidor ---
	err = server.InitServer(models.KernelConfig.PortKernel)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al iniciar el servidor del Kernel: %v", err))
		panic(err)
	}
}
