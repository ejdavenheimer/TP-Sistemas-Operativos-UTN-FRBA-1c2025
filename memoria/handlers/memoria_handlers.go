package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

func GetInstructionHandler(configPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		queryParams := r.URL.Query()
		pidStr := queryParams.Get("pid")
		pcStr := queryParams.Get("pc")

		pid, _ := strconv.ParseInt(pidStr, 10, 64)
		pc, _ := strconv.ParseInt(pcStr, 10, 64)

		instructionResult, isLast, err := services.GeInstruction(uint(pid), uint(pc))
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

func ReadMemoryHandler(w http.ResponseWriter, r *http.Request) {
	var request models.ReadRequest

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		slog.Error("Error al decodificar")
		return
	}

	// Delay de memoria
	time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)

	value, err := services.Read(uint(request.Pid), request.PhysicalAddress, request.Size)
	if err != nil {
		switch err {
		case services.ErrProcessNotFound:
			http.Error(w, "Proceso no encontrado", http.StatusNotFound)
			slog.Warn("Intento de lectura de proceso inexistente", "pid", request.Pid)
		case services.ErrMemoryViolation:
			http.Error(w, "Violación de memoria", http.StatusForbidden)
			slog.Warn("Violación de memoria detectada", "pid", request.Pid, slog.Int("direccion", request.PhysicalAddress))
		case services.ErrInvalidRead:
			http.Error(w, "Lectura inválida", http.StatusBadRequest)
			slog.Warn("Lectura inválida", "pid", request.Pid)
		default:
			http.Error(w, "Error interno leyendo memoria", http.StatusInternalServerError)
			slog.Error("Error al leer desde memoria", "pid", request.Pid, "direccion", request.PhysicalAddress, "error", err)
		}
		return
	}

	//log obligatorio
	slog.Info(fmt.Sprintf("## PID: <%d> - <Lectura> - Dir. Física: <%d> - Tamaño: <%d>", request.Pid, request.PhysicalAddress, request.Size))

	// Devolver el valor leído
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}

func SearchFrameHandler(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		PID     uint  `json:"pid"`
		Entries []int `json:"entries"`
	}

	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		slog.Error("Error al decodificar")
		return
	}

	// Delay de memoria
	time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)

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

		err = services.ExecuteDumpMemory(dumpRequest.Pid, dumpRequest.Size)
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

func WriteHandler(w http.ResponseWriter, r *http.Request) {
	//VALIDACION METODO HTTP
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Pid             uint   `json:"pid"`
		PhysicalAddress int    `json:"physical_address"`
		Data            string `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		slog.Error("Invalid WRITE request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Delay de memoria
	time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)

	//EJECUCION ESCRITURA
	dataBytes := []byte(request.Data)
	if err := services.WriteToMemory(request.Pid, request.PhysicalAddress, dataBytes); err != nil {
		slog.Error("WRITE failed", "error", err)
		http.Error(w, "Write failed", http.StatusInternalServerError)
		return
	}
	services.IncrementMetric(request.Pid, "writes")
	slog.Info(fmt.Sprintf("## PID: <%d> - <Write> - Dir. Física: <%d> - Dato: <%s>", request.Pid, request.PhysicalAddress, request.Data))
	w.WriteHeader(http.StatusOK) //RESPUESTA
}

func ReadPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Pid        uint `json:"pid"`
		PageNumber int  `json:"page_number"`
	}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		slog.Error("Error al decodificar request de CPU", "error", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Delay de memoria
	time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)

	// Validar existencia del proceso
	process, ok := models.ProcessTable[request.Pid]
	if !ok {
		slog.Warn(fmt.Sprintf("PID %d no encontrado en memoria", request.Pid))
		http.Error(w, "Process Not Found", http.StatusNotFound)
		return
	}

	// Validar que la página exista en el proceso
	if request.PageNumber < 0 || request.PageNumber >= len(process.Pages) {
		slog.Warn(fmt.Sprintf("Número de página inválido para PID %d: %d", request.Pid, request.PageNumber))
		http.Error(w, "Page Not Found", http.StatusNotFound)
		return
	}

	entry, err := services.FindPageEntry(request.Pid, models.PageTables[request.Pid], request.PageNumber)
	if err != nil {
		slog.Warn(fmt.Sprintf("Página %d no está presente en memoria para PID %d: %v", request.PageNumber, request.Pid, err))
		http.Error(w, "Page Not Present", http.StatusConflict)
		return
	}

	// Obtener el contenido del frame en memoria física
	pageSize := models.MemoryConfig.PageSize
	frameStart := entry.Frame * pageSize
	services.UpdatePageBit(request.Pid, frameStart, "use")
	services.IncrementMetric(request.Pid, "reads")
	frameEnd := frameStart + pageSize

	if frameEnd > len(models.UserMemory) {
		slog.Error("Acceso fuera de límites de memoria")
		http.Error(w, "Internal Error", http.StatusInternalServerError)
		return
	}

	content := models.UserMemory[frameStart:frameEnd]
	response := struct {
		Content []byte `json:"content"`
	}{
		Content: content,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Error al codificar la respuesta", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	slog.Debug(fmt.Sprintf("Página %d del proceso %d leída correctamente", request.PageNumber, request.Pid))
}

func FramesInUseHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	fmt.Println("=== Frames Ocupados ===")

	for pid, rootTable := range models.PageTables {
		collectFramesFromTable(pid, rootTable)
	}

	fmt.Fprintln(w, "OK") // Devuelve OK al cliente
}

func collectFramesFromTable(pid uint, table *models.PageTableLevel) {
	if table == nil {
		return
	}

	if table.IsLeaf && table.Entry != nil && table.Entry.Presence {
		line := fmt.Sprintf("PID: %d - Frame: %d\n", pid, table.Entry.Frame)
		fmt.Print(line) // También imprimo en consola
	}

	for _, sub := range table.SubTables {
		collectFramesFromTable(pid, sub)
	}
}

func FramesInUseHandlerV2(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	queryParams := r.URL.Query()
	pidStr := queryParams.Get("pid")
	pid, _ := strconv.ParseInt(pidStr, 10, 64)

	slog.Debug("Memoria: Recibida solicitud para obtener Frames en Uso.")

	// Slice para recolectar todos los FrameInfo
	allFrames := make([]models.FrameInfo, 0) // Inicializa un slice vacío

	// Itera sobre todas las tablas de páginas de los procesos
	for _, rootTable := range models.PageTables { // Asumo models.PageTables es accesible
		services.CollectFramesFromTableV2(uint(pid), rootTable, &allFrames) // Pasa el slice por referencia
	}

	groupedOutput := services.GroupFramesByPID(uint(pid), allFrames)

	// 2. Preparar la respuesta JSON
	response := map[string]interface{}{
		"status": "success",
		"data":   groupedOutput, // Incluye el slice de FrameInfo en la respuesta
	}

	server.SendJsonResponse(w, response)
}

func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	pidStr := r.URL.Query().Get("pid")
	pidInt, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil || pidInt < 0 {
		http.Error(w, "PID inválido", http.StatusBadRequest)
		return
	}
	pid := uint(pidInt)

	metrics, ok := models.ProcessMetrics[pid]
	if !ok {
		http.Error(w, "Proceso no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "=== Métricas PID: %d ===\n", pid)
	fmt.Fprintf(w, "PageTableAccesses: %d\n", metrics.PageTableAccesses)
	fmt.Fprintf(w, "InstructionFetches: %d\n", metrics.InstructionFetches)
	fmt.Fprintf(w, "SwapsOut: %d\n", metrics.SwapsOut)
	fmt.Fprintf(w, "SwapsIn: %d\n", metrics.SwapsIn)
	fmt.Fprintf(w, "Reads: %d\n", metrics.Reads)
	fmt.Fprintf(w, "Writes: %d\n", metrics.Writes)
}

func UpdatePageHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: revisar que datos necesito recibir
		//var dumpRequest models.DumpMemoryRequest
		//
		//// Decodifica el request (codificado en formato json).
		//err := json.NewDecoder(r.Body).Decode(&dumpRequest)
		//if err != nil {
		//	http.Error(w, err.Error(), http.StatusInternalServerError)
		//	return
		//}

		slog.Debug(fmt.Sprintf("## PID: <%d> Actualizando página completa"))

		services.UpdatePage()
		//if err != nil {
		//	w.WriteHeader(http.StatusNotFound)
		//	slog.Error(fmt.Sprintf("error: %s", err.Error()))
		//	return
		//}

		//response := models.DumpMemoryResponse{
		//	Result: "Ok",
		//}
		response := map[string]interface{}{
			"data": "Ok",
		}
		server.SendJsonResponse(w, response)
	}
}
