package services

import (
	"encoding/json"
	"fmt"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
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
	_, err = client.DoRequest(device.Port, device.Ip, "POST", "io", body)

	if err != nil {
		slog.Error("error:", err)
		return
	}
}

func ExecuteSyscall(syscallType string, values []string) {
	ioName := values[0]
	switch ioName {
	case "IO":
		//TODO: ver de donde sacamos el IP y Port de IO
		device := ioModel.Device{Ip: "127.0.0.1", Port: 8003, Name: ioName}
		sleepTime, _ := strconv.Atoi(values[1])
		SleepDevice(0, sleepTime, device)
	default:
		slog.Error("Invalid syscall type:", ioName)
		panic(fmt.Sprintf("Invalid syscall type: %s", ioName))
	}
}
