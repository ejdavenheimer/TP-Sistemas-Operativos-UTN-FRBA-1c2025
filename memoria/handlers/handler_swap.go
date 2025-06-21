package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
)

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

	services.PutProcessInSwap(req.PID)

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

	services.RemoveProcessInSwap(req.PID)

	w.WriteHeader(http.StatusOK) //RESPUESTA
}
