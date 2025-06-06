package main

import (
	"fmt"
	"log/slog"
	"net/http"

	memoryHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/helpers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

const (
	//TODO: revisar para que se pueda pasar cualquiera de los dos formatos
	//NO borrar el comentario de ConfigPath
	ConfigPath = "memoria/configs/memoria.json" //"./configs/memoria.json"
	LogPath    = "./logs/memoria.log"           //"./memoria.log"
)

func main() {
	helpers.InitMemory(ConfigPath, LogPath)

	// MockUp para probar cosas de swap
	services.MockCargarProcesosEnMemoria()

	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Memoria"))
	http.HandleFunc("GET /memoria", handlers.HandshakeHandler("Memoria en funcionamiento 🚀"))
	http.HandleFunc("GET /memoria/instrucciones", memoryHandler.GetInstructionsHandler(models.MemoryConfig.ScriptsPath))
	http.HandleFunc("GET /memoria/instruccion", memoryHandler.GetInstructionHandler(models.MemoryConfig.ScriptsPath))
	http.HandleFunc("GET /config/memoria", memoryHandler.MemoryConfigHandler)
	http.HandleFunc("POST /memoria/dump-memory", memoryHandler.DumpMemoryHandler())
	http.HandleFunc("POST /memoria/leerMemoria", memoryHandler.ReadMemoryHandler)
	http.HandleFunc("POST /memoria/buscarFrame", memoryHandler.SearchFrameHandler)
	http.HandleFunc("POST /memoria/cargarpcb", memoryHandler.ReserveMemoryHandler)
	http.HandleFunc("POST /memoria/write", memoryHandler.WriteHandler)
	http.HandleFunc("POST /memoria/swapIn", memoryHandler.PutProcessInSwapHandler)
	http.HandleFunc("POST /memoria/swapOut", memoryHandler.RemoveProcessInSwapHandler)
	//Liberar espacio de memoria de un PCB
	http.HandleFunc("POST /memoria/liberarpcb", memoryHandler.DeleteContextHandler)
	slog.Debug("Memoria lista")

	err := server.InitServer(models.MemoryConfig.PortMemory)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
