package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
)

func ExecuteProcessHandler(resp http.ResponseWriter, req *http.Request) {
	var pcb models.PCB
	err := json.NewDecoder(req.Body).Decode(&pcb)

	if err != nil {
		http.Error(resp, "Error al decodificar el cuerpo de la solicitud", http.StatusBadRequest)
		return
	}

	services.SelectToExecute(pcb)
}
