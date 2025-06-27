package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/helpers"
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

		slog.Debug(fmt.Sprintf("Dispositivo conectado: %v", device))

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
		var state models.Estado = models.EstadoReady

		if device.Reason == "KILL" {
			slog.Debug("Se murió :(")
			state = models.EstadoExit
		}

		_, isSuccess, err = services.MoveProcessToState(uint(pid), state)

		if !isSuccess || err != nil {
			slog.Error("Qué rompimos? :(")
			http.Error(writer, "Qué rompimos? :(", http.StatusBadRequest)
			return
		}

		//chequeo si hay un otro proceso esperando
		pidWaiting, isSuccess := helpers.GetAndRemoveOnePidForDevice(device.Name)

		if isSuccess {
			slog.Debug(fmt.Sprintf("Se encontró un proceso esperando por el dispositivo [%s]", device.Name))
			_, isSuccess, err = services.MoveProcessToState(uint(pidWaiting), state)

			if !isSuccess || err != nil {
				slog.Error("Qué rompimos? :(")
				http.Error(writer, "Qué rompimos? :(", http.StatusBadRequest)
				return
			}
		}

		server.SendJsonResponse(writer, device)
	}
}

//func DeviceRegisterHandler() http.HandlerFunc {
//	return func(w http.ResponseWriter, r *http.Request) {
//		var request models.Device
//		err := json.NewDecoder(r.Body).Decode(&request)
//		if err != nil {
//			slog.Error("Error al decodificar dispositivo IO", "error", err)
//			http.Error(w, "Bad request", http.StatusBadRequest)
//			return
//		}
//
//		models.ConnectedDevicesMap.Set(request.Name, ioModel.Device{
//			Name: request.Name,
//			Ip:   request.Ip,
//			Port: request.Port,
//		})
//
//		slog.Info(fmt.Sprintf("Dispositivo IO conectado: %s (%s:%d)", request.Name, request.Ip, request.Port))
//
//		w.Header().Set("Content-Type", "application/json")
//		json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
//	}
//}

// EJECUTA UNA SYSCALL IO
func ExecuteSyscallIOHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var syscallRequest models.SyscallRequest
		err := json.NewDecoder(request.Body).Decode(&syscallRequest)
		if err != nil {
			http.Error(writer, "Error al decodificar el cuerpo de la solicitud", http.StatusBadRequest)
			return
		}

		slog.Debug(fmt.Sprintf("BODY: %v", syscallRequest))
		//BUSCA AL DISP SOLICITADO
		deviceRequested, exists := models.ConnectedDevicesMap.Get(syscallRequest.Type)
		if !exists || deviceRequested.Name == "" {
			slog.Error(fmt.Sprintf("No se encontro al dispositivo %s", syscallRequest.Type))
			http.Error(writer, "Dispositivo no conectado.", http.StatusNotFound)
			services.EndProcess(syscallRequest.Pid, fmt.Sprintf("No se encontro al dispositivo %s", syscallRequest.Type))
			return
		}

		services.ExecuteSyscall(models.SyscallRequest{
			Type:   syscallRequest.Type,
			Pid:    syscallRequest.Pid,
			Values: syscallRequest.Values,
		}, writer)
	}
}

func GetDevicesMap() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		models.ConnectedDevicesMap.GetAll()
		writer.WriteHeader(http.StatusOK)
	}
}
