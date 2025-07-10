package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// este servicio realiza la conexión con kernel.
func ConnectToKernel(ioName string, ioConfig *models.Config) {
	//Crea y codifica la request de conexion a Kernel
	var request = models.Device{Name: ioName, Ip: ioConfig.IpIo, Port: ioConfig.PortIo}
	body, err := json.Marshal(request)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}

	//Envia la request de conexion a Kernel
	_, err = client.DoRequest(ioConfig.PortKernel, ioConfig.IpKernel, "POST", "kernel/dispositivos", body)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}

	slog.Debug("Dispositivo registrado exitosamente con el Kernel", "nombre", ioName)
	slog.Debug(fmt.Sprintf("%%# IO: %s - Registrado en Kernel - IO: %s:%d - Kernel: %s:%d",
		ioName,
		ioConfig.IpIo, ioConfig.PortIo,
		ioConfig.IpKernel, ioConfig.PortKernel))
}

func notifyKernel(pid uint, message string, ioConfig *models.Config) {
	//Crea y codifica la request de conexion a Kernel
	var request = models.DeviceResponse{Pid: pid, Reason: message, Port: ioConfig.PortIo, Name: models.IoName}
	body, err := json.Marshal(request)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}

	slog.Debug("Se envía notificación de finalización de dispositivo.")
	//Envia la request de conexion a Kernel
	_, err = client.DoRequest(ioConfig.PortKernel, ioConfig.IpKernel, "POST", "kernel/informar-io-finalizada", body)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}
}

func Sleep(pid uint, suspensionTime int) {
	slog.Debug("Inicio de operación IO", "pid", pid, "duración_ms", suspensionTime)
	slog.Debug(fmt.Sprintf("[%d] zzzzzzzzzz", pid))
	time.Sleep(time.Duration(suspensionTime) * time.Millisecond)
	slog.Debug("quién me desperto?? (mirada que juzga)")
	slog.Debug("Fin de operación IO", "pid", pid)
	notifyKernel(pid, "Fin de IO", models.IoConfig)
}

func NotifyDisconnection() {
	const InvalidPid uint = 10000000
	var request = models.DeviceResponse{Pid: InvalidPid, Reason: "KILL", Port: models.IoConfig.PortIo}

	body, err := json.Marshal(request)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}

	_, err = client.DoRequest(models.IoConfig.PortKernel, models.IoConfig.IpKernel, "POST", "kernel/dispositivo-finalizado", body)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}

}
