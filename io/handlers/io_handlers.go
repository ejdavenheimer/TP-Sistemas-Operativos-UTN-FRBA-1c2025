package handlers

import (
	"encoding/json"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"net/http"
)

func SleepHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		//mx_interfaz.Lock()

		//--------- RECIBE ---------

		var deviceRequest models.DeviceRequest

		// Decodifica el request (codificado en formato json).
		err := json.NewDecoder(request.Body).Decode(&deviceRequest)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		//--------- EJECUTA ---------
		services.Sleep(deviceRequest.Pid, deviceRequest.SuspensionTime)

		//--------- RESPUESTA ---------

		writer.WriteHeader(http.StatusOK)
		//mx_interfaz.Unlock()
	}
}
