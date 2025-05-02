package services

import (
	"encoding/json"
	"fmt"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// este servicio le solicita al dispositivo que duerme por el tiempo que le pasemos.
func SleepDevice(pid int, timeSleep int, device ioModel.Device) {
	//Crea y codifica la request de conexion a Kernel
	var request = models.DeviceRequest{Pid: pid, SuspensionTime: timeSleep}
	body, err := json.Marshal(request)

	if err != nil {
		slog.Error("error", slog.String("message", err.Error()))
		return
	}

	//Envia la request de conexion a Kernel
	response, err := client.DoRequest(device.Port, device.Ip, "POST", "io", body)
	var deviceResponse ioModel.DeviceResponse

	if err != nil {
		deviceResponse := ioModel.DeviceResponse{
			Pid:    pid,
			Reason: "Dispositivo desconectado",
		}
		EndProcess(deviceResponse)
		result, index, _ := models.ConnectedDeviceList.Find(func(d ioModel.Device) bool {
			return device.Port == d.Port
		})
		slog.Debug(fmt.Sprintf("Se va a desconectar el dispositivo %s.", result.Name))
		models.ConnectedDeviceList.Remove(index)

		return
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	err = json.Unmarshal(responseBody, &deviceResponse)
	if err != nil {
		slog.Error(fmt.Sprintf("error parseando el JSON: %v", err))
	}

	slog.Debug(fmt.Sprintf("Response: %s", deviceResponse.Reason))
}

func ExecuteSyscall(syscallRequest models.SyscallRequest, writer http.ResponseWriter) {
	ioName := syscallRequest.Values[0]
	switch ioName {
	case "IO":
		deviceRequested, index, exists := models.ConnectedDeviceList.Find(func(d ioModel.Device) bool {
			return syscallRequest.Type == d.Name && d.IsFree
		})

		if index == -1 {
			slog.Debug("El dispositivo se encuentra ocupado...")
			//TODO: revisar que pasa en este caso, entiendo que se bloqueo
			BlokedProcess(ioModel.DeviceResponse{Pid: syscallRequest.Pid, Reason: fmt.Sprintf("El dispositivo %s no se encuentra disponible", syscallRequest.Type)})
			return
		}

		if !exists || deviceRequested.Name == "" {
			slog.Error(fmt.Sprintf("No se encontro al dispositivo %s", syscallRequest.Type))
			http.Error(writer, "Dispositivo no conectado.", http.StatusNotFound)
			EndProcess(ioModel.DeviceResponse{Pid: syscallRequest.Pid, Reason: fmt.Sprintf("No se encontro al dispositivo %s", syscallRequest.Type)})
			return
		}

		deviceRequested.IsFree = false
		err := models.ConnectedDeviceList.Set(index, deviceRequested)
		if err != nil {
			slog.Error(fmt.Sprintf("error: %v", err))
			return
		}

		device := ioModel.Device{Ip: deviceRequested.Ip, Port: deviceRequested.Port, Name: syscallRequest.Values[1]}
		sleepTime, _ := strconv.Atoi(syscallRequest.Values[2])
		SleepDevice(syscallRequest.Pid, sleepTime, device)
		deviceRequested.IsFree = true
		err = models.ConnectedDeviceList.Set(index, deviceRequested)
		if err != nil {
			slog.Error(fmt.Sprintf("error: %v", err))
			return
		}
	case "INIT_PROC":
		slog.Warn("INIT_PROC") //TODO: implementar
	case "DUMP_MEMORY":
		slog.Warn("DUMP_MEMORY") //TODO: implementar
	case "EXIT":
		slog.Warn("EXIT") //TODO: implementar
	default:
		slog.Error("Invalid syscall type", slog.String("type", ioName))
        panic(fmt.Sprintf("Invalid syscall type: %s", ioName))
	}
}

func EndProcess(response ioModel.DeviceResponse) {
	slog.Debug(fmt.Sprintf("[%d] Finaliza el proceso - Motivo: %s", response.Pid, response.Reason))
	//TODO: implementar lógica para finalizar proceso
	slog.Info(fmt.Sprintf("## (<%d>) - Finaliza el proceso", response.Pid))
}

func BlokedProcess(response ioModel.DeviceResponse) {
	slog.Debug(fmt.Sprintf("[%d] Se bloquea el proceso el proceso - Motivo: %s", response.Pid, response.Reason))
	//TODO: implementar lógica para bloquear proceso
}
