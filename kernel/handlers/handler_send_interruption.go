package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
)

type loNecesario struct {
	PID    uint
	Puerto int
	IP     string
}

func SendInterruptionHandler(resp http.ResponseWriter, req *http.Request) {
	slog.Info("Me meti al SendInterruption")
	var pcb loNecesario
	err := json.NewDecoder(req.Body).Decode(&pcb)

	if err != nil {
		http.Error(resp, "Error al decodificar el cuerpo de la solicitud", http.StatusBadRequest)
		return
	}

	// services.SelectToExecute() ----DESCOMENTAR!!!!!!!
	services.SendInterruption(pcb.PID, pcb.Puerto, pcb.IP)
}
