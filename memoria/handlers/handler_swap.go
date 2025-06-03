package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
)

func SwapHandler(w http.ResponseWriter, r *http.Request) {
	//VALIDACION METODO HTTP
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	//Recibe el PID del proceso a suspender
	var pid int
	if err := json.NewDecoder(r.Body).Decode(&pid); err != nil {
		slog.Error("Invalid request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	services.MoveToSwap(pid)

	w.WriteHeader(http.StatusOK) //RESPUESTA
}
