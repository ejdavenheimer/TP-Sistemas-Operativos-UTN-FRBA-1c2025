package handlers

import (
	"encoding/json"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/services"
	kernelModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"net/http"
)

func SleepHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		ioModel.DeviceMutex.Lock()

		//--------- RECIBE ---------

		var deviceRequest kernelModel.DeviceRequest

		// Decodifica el request (codificado en formato json).
		err := json.NewDecoder(request.Body).Decode(&deviceRequest)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		//--------- EJECUTA ---------
		services.Sleep(deviceRequest.Pid, deviceRequest.SuspensionTime)

		//--------- RESPUESTA ---------

		response := ioModel.DeviceResponse{
			Pid:    deviceRequest.Pid,
			Reason: "Fin de IO",
		}

		server.SendJsonResponse(writer, response)
		ioModel.DeviceMutex.Unlock()
	}
}
