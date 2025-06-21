package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/services"
)

func DeleteContextHandler(w http.ResponseWriter, r *http.Request) {
	//VALIDACION METODO HTTP
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	//Recibe el PID del proceso a finalizar
	var req uint
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		slog.Error("Invalid request", "error", err)
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	services.ClearMemoryOfProcess(req)

	w.WriteHeader(http.StatusOK) //RESPUESTA
	//TODO: Más adelante, debe enviar el PCB a una función que realmente se encargue de borrar el contexto
}
