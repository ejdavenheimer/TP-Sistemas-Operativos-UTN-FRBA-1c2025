package main

import (
	"fmt"
	cpuHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

const (
	ConfigPath = "cpu/configs/cpu.json"
//	LogPath    = "cpu.log"
)

func main() {
	if len(os.Args) < 2 {
		slog.Error("Faltó el identificador de la CPU. Ejemplo: ./bin/cpu [identificador]")
		return
	}
	idCpu := os.Args[1]
	portCpu, err := strconv.Atoi(os.Args[2])
    if err != nil {
        fmt.Println("Puerto inválido:", os.Args[2])
        os.Exit(1)
    }

	config.InitConfig(ConfigPath, &models.CpuConfig)

    // Sobrescribimos el valor en el config
    models.CpuConfig.PortCpu = portCpu

	logPath, err := log.BuildLogPath("cpu_%s", idCpu)
	if err != nil {
		slog.Error("No se pudo construir el log path", "err", err)
		return
	}

	log.InitLogger(logPath, models.CpuConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port cpu: %d", models.CpuConfig.PortCpu))

	//var cpuNumber int = 1 //TODO: revisar de donde sacamos el número de CPU => nombre de archivo de config?

	http.HandleFunc("GET /", handlers.HandshakeHandler(fmt.Sprintf("Bienvenido al módulo de CPU%s", idCpu)))
	http.HandleFunc("GET /cpu", handlers.HandshakeHandler("Cpu en funcionamiento 🚀"))
	http.HandleFunc("POST /cpu/exec", cpuHandler.ExecuteHandler(models.CpuConfig))

	err = server.InitServer(models.CpuConfig.PortCpu)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
