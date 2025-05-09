package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	cpuHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

const (
	//TODO: revisar para que se pueda pasar cualquiera de los dos formatos
	//NO borrar el comentario de ConfigPath
	ConfigPath = "cpu/configs/cpu.json" //"./configs/cpu.json"
// LogPath    = "cpu.log"
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

	//CPU debe avisar que está disponible al Kernel, así se arma una lista para ver cuál usará
	cpuId, err := strconv.Atoi(idCpu) //Pasa a entero
	if err != nil {
		fmt.Println("Error al convertir a int:", err)
		return
	}
	services.ConnectToKernel(cpuId, models.CpuConfig)

	http.HandleFunc("GET /", handlers.HandshakeHandler(fmt.Sprintf("Bienvenido al módulo de CPU%s", idCpu)))
	http.HandleFunc("GET /cpu", handlers.HandshakeHandler("Cpu en funcionamiento 🚀"))
	http.HandleFunc("POST /cpu/process", cpuHandler.ExecuteHandler(models.CpuConfig)) //TODO: deprecado, borrar EP
	http.HandleFunc("POST /cpu/exec", cpuHandler.ExecuteProcessHandler(models.CpuConfig))
	http.HandleFunc("POST /cpu/interrupt", cpuHandler.InterruptProcessHandler(models.CpuConfig))

	err = server.InitServer(models.CpuConfig.PortCpu)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
