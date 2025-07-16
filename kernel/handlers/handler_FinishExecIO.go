package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/helpers"
	"log/slog"
	"net/http"

	ioModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
)

func FinishExecIOHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var req ioModels.DeviceResponse
		err := json.NewDecoder(request.Body).Decode(&req)
		if err != nil {
			slog.Warn("Error al decodificar PID desde IO", "error", err)
			http.Error(writer, "PID inválido", http.StatusBadRequest)
			return
		}

		isSuccess, pid := services.FinishDevice(req.Port)

		if !isSuccess {
			slog.Error("Error al quitar el dispositivo. Qué rompimos? :(")
			http.Error(writer, "Error al quitar el dispositivo. Qué rompimos? :(", http.StatusBadRequest)
			return
		}

		slog.Debug("Solicitud de finalización de IO recibida", "pid", pid)
		slog.Debug(fmt.Sprintf("Motivo de desalojo: %s", req.Reason))

		//chequeo si hay un otro proceso esperando por el dispostivo
		go ProcessNextWaitingDevice(req, writer)

		isSuccess, errorMessage := services.UnblockSyscallBlocked(uint(pid))

		if !isSuccess {
			slog.Error(errorMessage)
			http.Error(writer, errorMessage, http.StatusBadRequest)
			return
		}

		writer.WriteHeader(http.StatusOK)
	}
}

func ProcessNextWaitingDevice(request ioModels.DeviceResponse, writer http.ResponseWriter) {
	pidWaiting, isSuccess := helpers.GetAndRemoveOnePidForDevice(request.Name)

	if isSuccess {
		processNextWaiting(uint(pidWaiting), request, writer)
	}
}

func processNextWaiting(pidWaiting uint, request ioModels.DeviceResponse, writer http.ResponseWriter) {
	slog.Debug(fmt.Sprintf("Se encontró un proceso esperando por el dispositivo [%s]", request.Name))

	pcb, _, isSuccess := services.FindPCBInAnyQueue(uint(pidWaiting))

	if !isSuccess {
		slog.Error(fmt.Sprintf("No se encontró el proceso <%d>. Qué rompimos? :(", pidWaiting))
		http.Error(writer, fmt.Sprintf("No se encontró el proceso <%d>. Qué rompimos? :(", pidWaiting), http.StatusBadRequest)
		return
	}

	var state models.Estado = models.EstadoExit

	if pcb.EstadoActual == models.EstadoBlocked {
		state = models.EstadoReady
	}

	if pcb.EstadoActual == models.EstadoReady {
		state = models.EstadoSuspendidoReady
	}

	_, isSuccess, err := services.MoveProcessToState(pcb.PID, state, false)

	if !isSuccess || err != nil {
		slog.Error(fmt.Sprintf("Se produjo un error al mover el proceso <%d> a la cola <%s>. Qué rompimos? :(", pcb.PID, state))
		http.Error(writer, fmt.Sprintf("Se produjo un error al mover el proceso <%d> a la cola <%s>. Qué rompimos? :(", pcb.PID, state), http.StatusBadRequest)
		return
	}
}
