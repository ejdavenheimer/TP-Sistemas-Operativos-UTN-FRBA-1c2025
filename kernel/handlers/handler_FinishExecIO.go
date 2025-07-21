package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	ioModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/helpers"
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
		//isSuccess, errorMessage := services.UnblockSyscallBlocked(uint(pid))
		//
		//if !isSuccess {
		//	slog.Error(errorMessage)
		//	http.Error(writer, errorMessage, http.StatusBadRequest)
		//	return
		//}
		//helpers.AddPidForDevice(req.Name, pid)

		pids, _ := helpers.GetPidsForDevice(req.Name)
		if pids != nil && len(pids) > 0 {
			responseSent := ProcessNextWaitingDevice(req, writer)
			if responseSent { // Si ProcessNextWaitingDevice ya envió una respuesta, salimos
				return
			}
		}

		writer.WriteHeader(http.StatusOK)
		services.NotifyToReady()
	}
}

func ProcessNextWaitingDevice(request ioModels.DeviceResponse, writer http.ResponseWriter) bool {
	for i := 0; i < models.ConnectedDeviceList.Size(); i++ {
		pidWaiting, isSuccess := helpers.GetAndRemoveOnePidForDevice(request.Name)

		if isSuccess {
			isSuccess, errorMessage := services.UnblockSyscallBlocked(uint(pidWaiting))

			if !isSuccess {
				slog.Error(errorMessage)
				http.Error(writer, errorMessage, http.StatusInternalServerError)
				return true
			}
		}
	}
	return false
}
