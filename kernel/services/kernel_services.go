package services

import (
	"encoding/json"
	"fmt"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
	"io"
	"log/slog"
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

func ExecuteSyscall(device ioModel.Device, values []string) {
	ioName := values[0]
	switch ioName {
	case "IO":
		device := ioModel.Device{Ip: device.Ip, Port: device.Port, Name: values[1]}
		sleepTime, _ := strconv.Atoi(values[2])
		SleepDevice(0, sleepTime, device)
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
