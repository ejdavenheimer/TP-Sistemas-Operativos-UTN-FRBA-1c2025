package main

import (
	"fmt"
	"log/slog"
	"net/http"

	memoryHandler "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/log"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/handlers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

const (
	//TODO: revisar para que se pueda pasar cualquiera de los dos formatos
	//NO borrar el comentario de ConfigPath
	ConfigPath = "memoria/configs/memoria.json" //"./configs/memoria.json"
	LogPath    = "./logs/memoria.log" //"./memoria.log"
)

func main() {
	config.InitConfig(ConfigPath, &models.MemoryConfig)
	log.InitLogger(LogPath, models.MemoryConfig.LogLevel)

	slog.Debug(fmt.Sprintf("Port Memory: %d", models.MemoryConfig.PortMemory))
	models.InstructionsMap = make(map[uint][]string)

	models.UserMemory = make([]byte, models.MemoryConfig.MemorySize) //INICIALIZACION DE MEMORIA 
	slog.Debug("Memoria inicializada", "tamaño", len(models.UserMemory))
	


	http.HandleFunc("GET /", handlers.HandshakeHandler("Bienvenido al módulo de Memoria"))
	http.HandleFunc("GET /memoria", handlers.HandshakeHandler("Memoria en funcionamiento 🚀"))
	http.HandleFunc("GET /memoria/instrucciones", memoryHandler.GetInstructionsHandler(models.MemoryConfig.ScriptsPath))
	http.HandleFunc("GET /memoria/instruccion", memoryHandler.GetInstructionHandler(models.MemoryConfig.ScriptsPath))
	http.HandleFunc("GET /config/memoria", memoryHandler.MemoryConfigHandler)
	http.HandleFunc("POST /memoria/leerMemoria", memoryHandler.ReadMemoryHandler)
	http.HandleFunc("POST /memoria/buscarFrame", memoryHandler.SearchFrameHandler)
	
	http.HandleFunc("POST /memoria/cargarpcb", memoryHandler.ReserveMemoryHandler)
	http.HandleFunc("POST /memoria/write", memoryHandler.WriteHandler)
	slog.Info("Memoria lista")

	//Liberar espacio de memoria de un PCB
	http.HandleFunc("POST /memoria/liberarpcb", memoryHandler.DeleteContextHandler)

	err := server.InitServer(models.MemoryConfig.PortMemory)
	if err != nil {
		slog.Error(fmt.Sprintf("error initializing server: %v", err))
		panic(err)
	}
}
