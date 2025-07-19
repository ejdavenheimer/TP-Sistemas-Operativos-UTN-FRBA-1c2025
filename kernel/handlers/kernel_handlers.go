package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	cpuModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

func ConnectIoHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var device ioModel.Device
		err := json.NewDecoder(request.Body).Decode(&device)

		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		device.IsFree = true

		slog.Info(fmt.Sprintf("Dispositivo conectado: %v", device))

		//Guarda el dispositivo en el map de dispositivos conectados
		models.ConnectedDeviceList.Add(device)
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

		services.ExecuteSyscall(syscallRequest, writer)
		//writer.WriteHeader(http.StatusOK)
	}
}

func GetDevicesMapHandlers() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		models.ConnectedDeviceList.ForEach(func(device ioModel.Device) {
			slog.Debug(fmt.Sprintf("Device: %v", device))
		})
		writer.WriteHeader(http.StatusOK)
	}
}

func GetCpuMapHandlers() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var cpus []cpuModel.CpuN

		for _, cpu := range models.ConnectedCpuMap.M {
			cpus = append(cpus, *cpu)
			slog.Debug(fmt.Sprintf("CPU: %v", cpu))
		}

		writer.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(writer).Encode(cpus)
		if err != nil {
			slog.Error("No se pudo codificar la lista de CPUs", "error", err)
			http.Error(writer, "Error interno", http.StatusInternalServerError)
			return
		}
	}
}

func FinishDeviceHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var device ioModel.DeviceResponse
		err := json.NewDecoder(request.Body).Decode(&device)

		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		isSuccess, pid := services.FinishDevice(device.Port)

		if !isSuccess {
			slog.Error("Qué rompimos? :(")
			http.Error(writer, "Qué rompimos? :(", http.StatusBadRequest)
			return
		}

		pcb, _, isSuccess := services.FindPCBInAnyQueue(uint(pid))

		if !isSuccess {
			slog.Error(fmt.Sprintf("No se encontre el proceso <%d>", pid))
			http.Error(writer, "o se encontre el proceso :(", http.StatusBadRequest)
			return
		}

		//var state models.Estado = models.EstadoReady
		//
		//if pcb.EstadoActual == models.EstadoSuspendidoBlocked {
		//	state = models.EstadoSuspendidoReady
		//}

		if device.Reason == "KILL" {
			slog.Debug("Se murió :(")
			//state = models.EstadoExit
			models.ConnectedDeviceList.RemoveWhere(func(d ioModel.Device) bool {
				return d.Port == device.Port
			})
		}

		_, isSuccess, err = services.MoveProcessToState(pcb.PID, models.EstadoExit, pcb.EstadoActual == models.EstadoReady)

		if !isSuccess || err != nil {
			slog.Error("Qué rompimos? :(")
			http.Error(writer, "Qué rompimos? :(", http.StatusBadRequest)
			return
		}

		ProcessNextWaitingDevice(device, writer)
		//models.NotifyBlocked <- 1
		//services.NotifyToBlocked()

		services.StartLongTermScheduler()
		server.SendJsonResponse(writer, device)
	}
}
