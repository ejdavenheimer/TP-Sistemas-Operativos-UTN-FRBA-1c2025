package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

var ProcessTableLock sync.RWMutex

func GetInstructionHandler(configPath string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// --- CORRECCIÓN APLICADA ---
		// Se agrega el retardo de memoria para la búsqueda de instrucciones.
		time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)
		// --------------------------

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
	time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)

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
	slog.Info(fmt.Sprintf(
		"## PID: <%d> - <Lectura> - Dir. Física: <%d> - Tamaño: <%d> - LEIDO <%s>",
		request.Pid, request.PhysicalAddress, request.Size, value),
	)
	response := struct {
		Content []byte `json:"content"`
	}{
		Content: value,
	}

	// Devolver el valor leído
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Error al codificar respuesta de lectura", "error", err)
		http.Error(w, "Error codificando respuesta", http.StatusInternalServerError)
	}
}

func SearchFrameHandler(w http.ResponseWriter, r *http.Request) {
	type Request struct {
		PID        uint `json:"pid"`
		PageNumber int  `json:"pageNumber"`
	}
	var request Request
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		slog.Error("Error al decodificar SearchFrameHandler request")
		return
	}

	// Delay de memoria
	//time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)

	//buscar frame
	frame := services.SearchFrame(request.PID, request.PageNumber)

	type Response struct {
		Frame int `json:"frame"`
	}

	resp := Response{Frame: frame}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Error codificando respuesta SearchFrameHandler")
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

	var request models.WriteRequest
	body, _ := io.ReadAll(r.Body)
	slog.Debug("Cuerpo recibido", "json", string(body))
	r.Body = io.NopCloser(bytes.NewReader(body))

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		slog.Error("Invalid WRITE request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Delay de memoria
	time.Sleep(time.Duration(models.MemoryConfig.MemoryDelay) * time.Millisecond)

	// Validar existencia del proceso
	ProcessTableLock.RLock()
	_, ok := models.ProcessTable[request.Pid]
	slog.Debug(fmt.Sprintf("Busca a proceso %d en ProcessTable con dirección fisica %d", request.Pid, request.PhysicalAddress))
	ProcessTableLock.RUnlock()
	if !ok {
		slog.Warn(fmt.Sprintf("PID %d no encontrado en memoria", request.Pid))
		http.Error(w, "Process Not Found", http.StatusNotFound)
		return
	}

	//EJECUCION ESCRITURA
	//slog.Debug("Antes de llamar WriteToMemory", "PID", request.Pid, "PhysicalAddress", request.PhysicalAddress, "DataLen", len(dataBytes))
	if err := services.WriteToMemory(request.Pid, request.PhysicalAddress, []byte(request.Data)); err != nil {
		slog.Error("WRITE failed", "error", err)
		http.Error(w, "Write failed", http.StatusInternalServerError)
		return
	}
	//services.IncrementMetric(request.Pid, "writes")
	dataBytes := []byte(request.Data)
	if idx := bytes.IndexByte(dataBytes, 0); idx != -1 {
		dataBytes = dataBytes[:idx]
	}

	slog.Info(fmt.Sprintf("## PID: <%d> - <Escritura> - Dir. Física: <%d> - Tamaño: <%d> - Dato: <%s>", request.Pid, request.PhysicalAddress, len(dataBytes), string(dataBytes)))
	w.WriteHeader(http.StatusOK) //RESPUESTA
}

func ReadPageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		PID             uint   `json:"pid"`
		PageNumber      int    `json:"page_number"`
		PhysicalAddress int    `json:"physicalAddress"`
		Operacion       string `json:"operacion"`
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

	pageSize := models.MemoryConfig.PageSize
	// Calcular el byte 0 del frame (inicio de la página física)
	frameStart := (request.PhysicalAddress / pageSize) * pageSize
	content, err := services.Read(request.PID, frameStart, pageSize)
	if err != nil {
		switch err {
		case services.ErrProcessNotFound:
			slog.Warn(fmt.Sprintf("PID %d no encontrado", request.PID))
			http.Error(w, "Process Not Found", http.StatusNotFound)
		case services.ErrMemoryViolation:
			slog.Error(fmt.Sprintf("Violación de memoria para PID %d en dirección %d", request.PID, frameStart))
			http.Error(w, "Memory Violation", http.StatusBadRequest)
		case services.ErrInvalidRead:
			slog.Error("Tamaño de lectura inválido")
			http.Error(w, "Invalid Read", http.StatusBadRequest)
		default:
			slog.Error("Error interno en lectura", "error", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	switch request.Operacion {
	case "Lectura":
		slog.Info(fmt.Sprintf("## PID: <%d> - <Lectura Página por LECTURA> - Dir. Física: <%d> - Tamaño: <%d> - VALOR:<%s>", request.PID, frameStart, pageSize, string(content)))
	case "Escritura":
		slog.Info(fmt.Sprintf("## PID: <%d> - <Lectura Página por ESCRITURA> - Dir. Física: <%d> - Tamaño: <%d> - VALOR:<%s>", request.PID, frameStart, pageSize, string(content)))
	default:
		slog.Info(fmt.Sprintf("## PID: <%d> - <Lectura Página por OPERACIÓN DESCONOCIDA> - Dir. Física: <%d> - Tamaño: <%d> - VALOR:<%s>", request.PID, frameStart, pageSize, string(content)))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(struct {
		Content []byte `json:"content"`
	}{
		Content: content,
	})

	slog.Debug(fmt.Sprintf("Página %d del proceso %d leída correctamente", request.PageNumber, request.PID))
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
