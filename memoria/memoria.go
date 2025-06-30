package main

import (
	"fmt"
	"log/slog"
	"net/http"

	memoryHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/helpers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
)

const (
	//TODO: revisar para que se pueda pasar cualquiera de los dos formatos
	//NO borrar el comentario de ConfigPath
	//ConfigPath = "memoria/configs/memoria.json" //"./configs/memoria.json"
	//LogPath    = "./logs/memoria.log"           //"./memoria.log"
)

func main() {
    ConfigPath := config.MemoriaConfigPath()
    LogPath, err := log.BuildLogPath("memoria")
    if err != nil {
        slog.Error(fmt.Sprintf("No se pudo preparar el archivo de log: %v", err))
        return
    }
	helpers.InitMemory(ConfigPath, LogPath)
	// MockUp para probar cosas de swap
	//services.MockCargarProcesosEnMemoria()

	//Inicios
	http.HandleFunc("GET /config/memoria", memoryHandler.MemoryConfigHandler)
	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Memoria"))
	http.HandleFunc("GET /memoria", handlers.HandshakeHandler("Memoria en funcionamiento 🚀"))

	//Manejo de memoria del sistema
	http.HandleFunc("GET /memoria/instruccion", memoryHandler.GetInstructionHandler(models.MemoryConfig.ScriptsPath))

	//Acceso a tabla de paginas
	http.HandleFunc("POST /memoria/buscarFrame", memoryHandler.SearchFrameHandler)

	//Acceso a espacio de usuario
	http.HandleFunc("POST /memoria/leerPagina", memoryHandler.ReadPageHandler)
	http.HandleFunc("POST /memoria/write", memoryHandler.WriteHandler)

	//Leer página completa
	http.HandleFunc("POST /memoria/leerMemoria", memoryHandler.ReadMemoryHandler)

	//Memory Dump
	http.HandleFunc("POST /memoria/dump-memory", memoryHandler.DumpMemoryHandler())

	//Actualizar página completa
	http.HandleFunc("GET /memoria/framesOcupados", memoryHandler.FramesInUseHandler)
	http.HandleFunc("GET /memoria/v2/framesOcupados", memoryHandler.FramesInUseHandlerV2)
	http.HandleFunc("GET /memoria/metrics", memoryHandler.MetricsHandler)

	//Manejo de swap
	http.HandleFunc("POST /memoria/swapOut", memoryHandler.PutProcessInSwapHandler)
	http.HandleFunc("POST /memoria/swapIn", memoryHandler.RemoveProcessInSwapHandler)

	//Ocupar o Liberar espacio de memoria de un PCB
	http.HandleFunc("POST /memoria/cargarpcb", memoryHandler.ReserveMemoryHandler)
	http.HandleFunc("POST /memoria/liberarpcb", memoryHandler.EndProcessHandler)

	slog.Debug("Memoria lista")

	err = server.InitServer(models.MemoryConfig.PortMemory)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
