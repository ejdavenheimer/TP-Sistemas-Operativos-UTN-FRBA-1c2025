package services

import (
	"encoding/json"
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
	"log/slog"
	"time"
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

	slog.Info("Dispositivo registrado exitosamente con el Kernel", "nombre", ioName)
	slog.Info(fmt.Sprintf("%%# IO: %s - Registrado en Kernel - IO: %s:%d - Kernel: %s:%d",
		ioName,
		ioConfig.IpIo, ioConfig.PortIo,
		ioConfig.IpKernel, ioConfig.PortKernel))
}

func Sleep(pid int, suspensionTime int) {
	slog.Info("Inicio de operación IO", "pid", pid, "duración_ms", suspensionTime)
	slog.Debug(fmt.Sprintf("[%d] zzzzzzzzzz", pid))
	time.Sleep(time.Duration(suspensionTime) * time.Millisecond)
	slog.Debug("quién me desperto?? (mirada que juzga)")
	slog.Info("Fin de operación IO", "pid", pid)
}
