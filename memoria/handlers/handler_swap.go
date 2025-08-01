package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
)

var memoryOperationMutex sync.RWMutex
var activeMemoryOperations = make(map[uint]bool)

func isProcessBeingProcessed(pid uint) bool {
	memoryOperationMutex.RLock()
	defer memoryOperationMutex.RUnlock()
	return activeMemoryOperations[pid]
}

func setProcessBeingProcessed(pid uint, processing bool) {
	memoryOperationMutex.Lock()
	defer memoryOperationMutex.Unlock()
	if processing {
		activeMemoryOperations[pid] = true
	} else {
		delete(activeMemoryOperations, pid)
	}
}

func PutProcessInSwapHandler(w http.ResponseWriter, r *http.Request) {
	//VALIDACION METODO HTTP
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	//Recibe el PID del proceso a suspender
	var req models.PIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Invalid request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	slog.Debug("Iniciando PUT PROCESS IN SWAP", "PID", req.PID)

	// Verificar si el proceso ya está siendo procesado
	if isProcessBeingProcessed(req.PID) {
		slog.Warn("Proceso ya está siendo procesado en memoria", "PID", req.PID)
		http.Error(w, "Process already being processed", http.StatusConflict)
		return
	}

	// Marcar proceso como siendo procesado
	setProcessBeingProcessed(req.PID, true)
	defer setProcessBeingProcessed(req.PID, false)

	// Ejecutar operación de forma sincronizada
	if err := services.PutProcessInSwap(req.PID); err != nil {
		slog.Error("Error en PUT PROCESS IN SWAP", "PID", req.PID, "error", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("PUT PROCESS IN SWAP completado exitosamente", "PID", req.PID)
	w.WriteHeader(http.StatusOK) //RESPUESTA
}

func RemoveProcessInSwapHandler(w http.ResponseWriter, r *http.Request) {
	//VALIDACION METODO HTTP
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	//Recibe el PID del proceso a suspender
	var req models.PIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Invalid request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	slog.Debug("Iniciando REMOVE PROCESS IN SWAP", "PID", req.PID)

	// Verificar si el proceso ya está siendo procesado
	if isProcessBeingProcessed(req.PID) {
		slog.Warn("Proceso ya está siendo procesado en memoria", "PID", req.PID)
		http.Error(w, "Process already being processed", http.StatusConflict)
		return
	}

	// Marcar proceso como siendo procesado
	setProcessBeingProcessed(req.PID, true)
	defer setProcessBeingProcessed(req.PID, false)

	// Ejecutar operación de forma sincronizada
	if err := services.RemoveProcessInSwap(req.PID); err != nil {
		slog.Error("Error en REMOVE PROCESS IN SWAP", "PID", req.PID, "error", err)
		http.Error(w, "Internal Server Error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Debug("REMOVE PROCESS IN SWAP completado exitosamente", "PID", req.PID)
	w.WriteHeader(http.StatusOK) //RESPUESTA
}

func HandleCheckSwapStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		PID uint `json:"pid"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		slog.Error("Error decodificando request para check swap status", "error", err)
		http.Error(w, "Request inválido", http.StatusBadRequest)
		return
	}

	isInSwap := services.IsProcessInSwap(request.PID)

	response := struct {
		PID     uint `json:"pid"`
		InSwap  bool `json:"in_swap"`
		Success bool `json:"success"`
	}{
		PID:     request.PID,
		InSwap:  isInSwap,
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	slog.Debug("Memoria: Consultado estado de swap", "PID", request.PID, "in_swap", isInSwap)
}
