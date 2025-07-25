package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
)

// ConnectIoHandler maneja la conexión de un nuevo dispositivo de I/O.
func ConnectIoHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var device ioModel.Device
		if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
			http.Error(w, "Datos de dispositivo inválidos", http.StatusBadRequest)
			return
		}
		device.IsFree = true
		models.ConnectedDeviceManager.Add(&device)
		slog.Info("Dispositivo de I/O conectado", "nombre", device.Name, "puerto", device.Port)

		// Al conectarse un nuevo dispositivo, intentamos despachar un proceso que pudiera estar esperando.
		services.TryToDispatchNextIO(device.Name)

		w.WriteHeader(http.StatusOK)
	}
}

// FinishIoHandler maneja la notificación de fin de I/O de un dispositivo.
func FinishIoHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var response ioModel.DeviceResponse
		if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
			http.Error(w, "Respuesta de dispositivo inválida", http.StatusBadRequest)
			return
		}

		slog.Info("## (%d) - Fin de IO", response.Pid)

		// Liberamos el dispositivo.
		device, found := models.ConnectedDeviceManager.MarkAsFreeByPort(response.Port)
		if !found {
			slog.Warn("Se recibió fin de I/O de un dispositivo no registrado.", "puerto", response.Port)
		}

		// Desbloqueamos el proceso.
		services.UnblockProcess(response.Pid)

		// Intentamos despachar al siguiente proceso en la cola de espera para este tipo de dispositivo.
		if found {
			services.TryToDispatchNextIO(device.Name)
		}

		w.WriteHeader(http.StatusOK)
	}
}
