package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"log/slog"
	"net/http"
	"strconv"
)

func GetProcessHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		queryParams := request.URL.Query()
		pidStr := queryParams.Get("pid")

		pidInt, err := strconv.ParseInt(pidStr, 10, 64)
		if err != nil || pidInt < 0 {
			http.Error(writer, "Parámetro pid inválido", http.StatusBadRequest)
			return
		}
		
		pid := uint(pidInt)

		processResponse := services.GetProcess(pid)
		slog.Debug(fmt.Sprintf("PID: %d - Estado: : %s", processResponse.Pid, processResponse.EstadoActual))
		response := map[string]interface{}{
			"status": "success",
			"data":   processResponse,
		}

		server.SendJsonResponse(writer, response)
	}
}

func GetAllHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		slog.Debug("Kernel: Solicitud recibida para obtener el estado de las colas de planificación.")

		// Recolectar el estado actual de todas las colas
		queuesState := services.GetQueuesState()

		// Preparar la respuesta JSON
		response := map[string]interface{}{
			"status": "success",
			"data":   queuesState,
		}

		// Enviar la respuesta JSON
		server.SendJsonResponse(writer, response)
		slog.Debug("Kernel: Estado de las colas enviado exitosamente.")
	}
}

func AddProcessHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var processRequest models.ProcessRequest
		err := json.NewDecoder(request.Body).Decode(&processRequest)

		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		isSuccess, _ := services.AddProcessToQueue(processRequest.Pid, processRequest.EstadoActual)

		if !isSuccess {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		server.SendJsonResponse(writer, fmt.Sprintf("Proceso <%d> creado correctamente", processRequest.Pid))
	}
}

func UpdateProcessHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var processRequest models.ProcessRequest
		err := json.NewDecoder(request.Body).Decode(&processRequest)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		_, isSuccess, _ := services.MoveProcessToState(processRequest.Pid, processRequest.EstadoActual)

		if !isSuccess {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		server.SendJsonResponse(writer, fmt.Sprintf("Proceso <%d> modificado correctamente", processRequest.Pid))
	}
}
