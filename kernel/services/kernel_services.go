package services

import (
	"encoding/json"
	"errors"
	"fmt"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// este servicio le solicita al dispositivo que duerme por el tiempo que le pasemos.
func SleepDevice(pid int, timeSleep int, device ioModel.Device) error {
	//Crea y codifica la request de conexion a Kernel
	var request = models.DeviceRequest{Pid: pid, SuspensionTime: timeSleep}
	body, err := json.Marshal(request)

	if err != nil {
		slog.Error("error", slog.String("message", err.Error()))
		return err
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

		return errors.New("dispositivo desconectado")
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	err = json.Unmarshal(responseBody, &deviceResponse)
	if err != nil {
		slog.Error(fmt.Sprintf("error parseando el JSON: %v", err))
		return err
	}

	slog.Debug(fmt.Sprintf("Response: %s", deviceResponse.Reason))
	return nil
}

func ExecuteSyscall(syscallRequest models.SyscallRequest, writer http.ResponseWriter) {
	syscallName := syscallRequest.Type
	slog.Info(fmt.Sprintf("## %d - Solicitó syscall: %s", syscallRequest.Pid, syscallName))
	switch syscallName {
	case "IO":
		deviceRequested, index, exists := models.ConnectedDeviceList.Find(func(d ioModel.Device) bool {
			return syscallRequest.Values[0] == d.Name && d.IsFree
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

		device := ioModel.Device{Ip: deviceRequested.Ip, Port: deviceRequested.Port, Name: syscallRequest.Values[0]}
		sleepTime, _ := strconv.Atoi(syscallRequest.Values[1])
		err = SleepDevice(syscallRequest.Pid, sleepTime, device)
		if err != nil {
			slog.Error(fmt.Sprintf("error: %v", err))
			return
		}

		deviceRequested, index, _ = models.ConnectedDeviceList.Find(func(d ioModel.Device) bool {
			return syscallRequest.Values[0] == d.Name && !d.IsFree
		})
		deviceRequested.IsFree = true
		err = models.ConnectedDeviceList.Set(index, deviceRequested)
		if err != nil {
			slog.Error(fmt.Sprintf("error: %v", err))
			return
		}
	case "INIT_PROC":
		if len(syscallRequest.Values) < 2 {
			slog.Error("INIT_PROC necesita 2 parametros: path y tamaño")
			http.Error(writer, "Parametros insuficientes", http.StatusBadRequest)
			return
		}

		parentPID := syscallRequest.Pid

		pseudocodeFile := syscallRequest.Values[0]
		processSize, err := strconv.Atoi(syscallRequest.Values[1])
		if err != nil {
			slog.Error("Tamaño de proceso inválido", "valor", syscallRequest.Values[1])
			http.Error(writer, "Tamaño de proceso inválido", http.StatusBadRequest)
			return
		}

		// Paso en pid del padre como primer argumento
		additionalArgs := []string{strconv.Itoa(parentPID)}
		pcb, err := InitProcess(pseudocodeFile, processSize, additionalArgs)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Info("Proceso inicializado correctamente", "PID", pcb.PID)

		server.SendJsonResponse(writer, map[string]interface{}{
			"action":     "continue",
		})
	case "DUMP_MEMORY":
		slog.Warn("DUMP_MEMORY") //TODO: implementar
		server.SendJsonResponse(writer, map[string]string{
			"action": "continue",
		})
	case "EXIT":
		slog.Warn("EXIT") //TODO: implementar
		server.SendJsonResponse(writer, map[string]interface{}{
			"action": "exit",
		})
	default:
		slog.Error("Invalid syscall type", slog.String("type", syscallName))
		http.Error(writer, fmt.Sprintf("Tipo de syscall inválido: %s", syscallName), http.StatusBadRequest)
		//panic(fmt.Sprintf("Invalid syscall type: %s", syscallName))
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
