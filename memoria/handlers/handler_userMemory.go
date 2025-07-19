package handlers

import (
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
)

var uMemoryLock sync.Mutex

// Estructura para recibir la request del kernel
type UserMemoryRequest struct {
	PID  uint `json:"PID"`
	Size int  `json:"Size"`
}

func UserMemoryCapacityHandler(w http.ResponseWriter, r *http.Request) {
	// Validación método HTTP
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Decodificar la request
	var req UserMemoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Invalid request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	pid := req.PID
	size := req.Size

	slog.Debug("Verificando capacidad de memoria", "PID", pid, "Size", size)

	// Calcula cuántas páginas necesita
	pageSize := models.MemoryConfig.PageSize
	pageCount := int(math.Ceil(float64(size) / float64(pageSize)))
	slog.Debug("Page count calculado", "pid", pid, "size", size, "pageCount", pageCount)

	// Verifico que haya frames libres suficientes
	uMemoryLock.Lock()
	freeFramesCount := 0
	for _, free := range models.FreeFrames {
		if free {
			freeFramesCount++
		}
	}
	uMemoryLock.Unlock()

	if freeFramesCount < pageCount {
		slog.Debug("Memoria insuficiente", "PID", pid, "necesita", pageCount, "libres", freeFramesCount)
		w.WriteHeader(http.StatusNoContent) // o 204: no hay contenido, pero no es error
		return
	}

	slog.Debug("Capacidad de memoria verificada exitosamente", "PID", pid, "framesLibres", freeFramesCount, "framesNecesarios", pageCount)
	w.WriteHeader(http.StatusOK)
}
