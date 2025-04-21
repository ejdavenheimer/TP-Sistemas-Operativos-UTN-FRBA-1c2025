package handlers

import (
	"encoding/json"
	"fmt"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
	"log/slog"
	"net/http"
)

func ConnectIoHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var device ioModel.Device
		err := json.NewDecoder(request.Body).Decode(&device)

		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		slog.Debug(fmt.Sprintf("Dispositivo conectado: %v", device))

		//Guarda el dispositivo en el map de dispositivos conectados
		models.ConnectedDevicesMap.Set(device.Name, device)
		writer.WriteHeader(http.StatusOK)
	}
}

func ExecuteSyscallHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var syscallRequest models.SyscallRequest
		err := json.NewDecoder(request.Body).Decode(&syscallRequest)
		slog.Debug(fmt.Sprintf("BODY: %v", syscallRequest))
		if err != nil {
			http.Error(writer, "Error al decodificar el cuerpo de la solicitud", http.StatusBadRequest)
			return
		}

		deviceRequested, exists := models.ConnectedDevicesMap.Get(syscallRequest.Type)
		if !exists || deviceRequested.Name == "" {
			//TODO: ver que hace cuando no encuentra la interfaz
			slog.Error(fmt.Sprintf("No se encontro al dispositivo %s", syscallRequest.Type))
			http.Error(writer, "Interfaz no conectada.", http.StatusNotFound)
			return
		}

		services.ExecuteSyscall(deviceRequested, syscallRequest.Values)
	}
}
