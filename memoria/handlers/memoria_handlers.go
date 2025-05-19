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

	//slog.Info("Ruta recibida para cargar instrucciones", "path", request.Path)

	err := services.ReserveMemory(request.PID, request.Size, request.Path)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al reservar memoria: %v", err), http.StatusInternalServerError)
		return
	}

	//log obligatorio
	slog.Info(fmt.Sprintf("## PID: <%d> - Proceso Creado - Tamaño: <%d>", request.PID, request.Size))

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Memoria reservada con éxito"))
}

func MemoryConfigHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]int{
		"page_size":        models.MemoryConfig.PageSize,
		"entries_per_page": models.MemoryConfig.EntriesPerPage,
		"number_of_levels": models.MemoryConfig.NumberOfLevels,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func ReadMemoryHandler(w http.ResponseWriter, r *http.Request){
	var request models.MemoryInstructionRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		slog.Error("Error al decodificar")
		return
	}
	// Obtener proceso
	process, ok := models.ProcessTable[request.Pid]
	if !ok {
		http.Error(w, "Proceso no encontrado", http.StatusNotFound)
		slog.Warn("Intento de lectura de proceso inexistente", slog.Int("pid", request.Pid))
		return
	}
	// Validar que sea el inicio de página
	if request.PhysicalAddress % models.MemoryConfig.PageSize != 0 {
		http.Error(w, "Dirección no es inicio de página", http.StatusBadRequest)
		slog.Warn("Dirección física no alineada a inicio de página", slog.Int("direccion", request.PhysicalAddress))
		return
	}
	// Validar que el tamaño solicitado no exceda el tamaño de una página
	if request.Size > models.MemoryConfig.PageSize {
		http.Error(w, "Lectura excede el tamaño de una página", http.StatusBadRequest)
		slog.Warn("Lectura mayor al tamaño de página", slog.Int("pid", request.Pid))
		return
	}

	// Validar rango del proceso
	if request.PhysicalAddress < process.BaseAddress || request.PhysicalAddress + request.Size > process.BaseAddress+process.Size {
		http.Error(w, "Violación de memoria", http.StatusForbidden)
		slog.Warn("Violación de memoria", slog.Int("pid", request.Pid), slog.Int("direccion", request.PhysicalAddress))
		return
	}
	// Leer desde la memoria
	value, err := services.Read(request.PhysicalAddress, request.Size)
	if err != nil {
		http.Error(w, "Error leyendo memoria", http.StatusInternalServerError)
		slog.Error("Error al leer desde Memoria", slog.Int("direccion", request.PhysicalAddress))
		return
	}

	//log obligatorio
	slog.Info(fmt.Sprintf("## PID: <%d> - <Lectura> - Dir. Física: <%d> - Tamaño: <%d>", request.Pid, request.PhysicalAddress, request.Size))

	// Devolver el valor leído
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}

func SearchFrameHandler(w http.ResponseWriter, r *http.Request){
	type Request struct {
		PID     uint   `json:"pid"`
		Entries []int `json:"entries"`
	}

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		slog.Error("Error al decodificar")
		return
	}

	//buscar frame
	frame := services.SearchFrame(request.PID, request.Entries)

	type Response struct {
		Frame int `json:"frame"`
	}

	resp := Response{Frame: frame}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Error codificando respuesta")
	}
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
