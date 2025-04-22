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
		slog.Error("error:", err)
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
		models.ConnectedDevicesMap.Delete(device.Name)

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
		deviceRequested, exists := models.ConnectedDevicesMap.Get(syscallRequest.Type)
		if !exists || deviceRequested.Name == "" {
			slog.Error(fmt.Sprintf("No se encontro al dispositivo %s", syscallRequest.Type))
			http.Error(writer, "Dispositivo no conectado.", http.StatusNotFound)
			EndProcess(ioModel.DeviceResponse{Pid: syscallRequest.Pid, Reason: fmt.Sprintf("No se encontro al dispositivo %s", syscallRequest.Type)})
			return
		}
		device := ioModel.Device{Ip: deviceRequested.Ip, Port: deviceRequested.Port, Name: syscallRequest.Values[1]}
		sleepTime, _ := strconv.Atoi(syscallRequest.Values[2])
		SleepDevice(syscallRequest.Pid, sleepTime, device)
	case "INIT_PROC":
		slog.Warn("INIT_PROC") //TODO: implementar
	case "DUMP_MEMORY":
		slog.Warn("DUMP_MEMORY") //TODO: implementar
	case "EXIT":
		slog.Warn("EXIT") //TODO: implementar
	default:
		slog.Error("Invalid syscall type:", ioName)
		panic(fmt.Sprintf("Invalid syscall type: %s", ioName))
	}
}

func EndProcess(response ioModel.DeviceResponse) {
	slog.Debug(fmt.Sprintf("[%d] Finaliza el proceso - Motivo: %s", response.Pid, response.Reason))
	//TODO: implementar lógica para finalizar proceso
	slog.Info(fmt.Sprintf("## (<%d>) - Finaliza el proceso", response.Pid))
}
