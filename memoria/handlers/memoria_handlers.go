package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
	"strconv"
)

// TODO: deprecado, borrar
func GetInstructionsHandler(configPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		pidStr := queryParams.Get("pid")
		pathName := queryParams.Get("pathName")

		pid, _ := strconv.ParseInt(pidStr, 10, 64)

		path := configPath + pathName
		//services.GetInstructions(uint(pid), path, models.InstructionsMap)
		//instruction := models.InstructionsResponse{

		err := services.GetInstructions(uint(pid), path, models.InstructionsMap)
		if err != nil {
			slog.Error(fmt.Sprintf("Error obteniendo instrucciones: %v", err))
			http.Error(w, "Error obteniendo instrucciones", http.StatusInternalServerError)
			return
		}

		instruction := models.InstructionsResponse{
			Instruction: models.InstructionsMap,
		}
		slog.Debug(fmt.Sprintf("Se envierán %d instrucciones para ejecutar.", len(instruction.Instruction[uint(pid)])))
		server.SendJsonResponse(w, instruction)
	}
}

func GetInstructionHandler(configPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		pidStr := queryParams.Get("pid")
		pcStr := queryParams.Get("pc")

		pid, _ := strconv.ParseInt(pidStr, 10, 64)
		pc, _ := strconv.ParseInt(pcStr, 10, 64)

		instructionResult, isLast, err := services.GeInstruction(uint(pid), uint(pc), configPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			slog.Error(fmt.Sprintf("error: %s", err.Error()))
			return
		}

		instruction := models.InstructionResponse{
			Instruction: instructionResult,
			IsLast:      isLast,
		}
		slog.Info(fmt.Sprintf("## PID: <%d> - Obtener instrucción: <%d> - Instrucción: %s", pid, pc, instruction.Instruction))
		server.SendJsonResponse(w, instruction)
	}
}

func ReserveMemoryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var request models.MemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, fmt.Sprintf("Error al parsear el cuerpo de la solicitud: %v", err), http.StatusBadRequest)
		return
	}

	slog.Info("Ruta recibida para cargar instrucciones", "path", request.Path)

	err := reserveMemory(request.PID, request.Size, request.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al reservar memoria: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Memoria reservada con éxito"))
}

func reserveMemory(pid uint, size int, path string) error {
	if size > models.MemoryConfig.MemorySize {
		return fmt.Errorf("No hay suficiente memoria disponible")
	}

	// Intentar cargar las instrucciones
	err := services.GetInstructions(uint(pid), path, models.InstructionsMap)
	if err != nil {
		return fmt.Errorf("Error al cargar instrucciones: %v", err)
	}

	// Decrementar el espacio disponible en memoria
	models.MemoryConfig.MemorySize -= size

	// Solo mostrar el mensaje de éxito si las instrucciones se cargaron correctamente
	slog.Debug(fmt.Sprintf("Memoria reservada: %d bytes. Instrucciones cargadas para PID %d.", size, pid))

	return nil
}

func DumpMemoryHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var dumpRequest models.DumpMemoryRequest

		// Decodifica el request (codificado en formato json).
		err := json.NewDecoder(r.Body).Decode(&dumpRequest)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Debug(fmt.Sprintf("## PID: <%d> Dump Memory", dumpRequest.Pid))

		err = services.ExecuteDumpMemory(dumpRequest.Pid, dumpRequest.Size, models.MemoryConfig.DumpPath)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			slog.Error(fmt.Sprintf("error: %s", err.Error()))
			return
		}

		response := models.DumpMemoryResponse{
			Result: "Ok",
		}
		server.SendJsonResponse(w, response)
	}
}
