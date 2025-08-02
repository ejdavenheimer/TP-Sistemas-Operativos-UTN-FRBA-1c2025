package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"

	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	kernelModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// executeIOSyscall maneja la petición de I/O de un proceso.
func executeIOSyscall(pcb *kernelModels.PCB, request kernelModels.SyscallRequest) {
	deviceName := request.Values[0]
	time, err := strconv.Atoi(request.Values[1])
	if err != nil {
		slog.Error("Syscall IO con tiempo inválido. Finalizando proceso.", "PID", pcb.PID)
		TransitionProcessState(pcb, kernelModels.EstadoExit)
		StartLongTermScheduler()
		return
	}

	TransitionProcessState(pcb, kernelModels.EstadoBlocked)
	slog.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", pcb.PID, deviceName))

	device, exists := kernelModels.ConnectedDeviceManager.GetFreeByName(deviceName)

	if !exists {
		slog.Error("Syscall IO para un tipo de dispositivo inexistente. Finalizando proceso.", "dispositivo", deviceName, "PID", pcb.PID)
		TransitionProcessState(pcb, kernelModels.EstadoExit)
		StartLongTermScheduler()
		return
	}

	if device == nil {
		slog.Info("Dispositivo ocupado. Proceso encolado.", "dispositivo", deviceName, "PID", pcb.PID)
		// Guardamos la request en el PCB antes de encolarlo.
		pcb.PendingIoRequest = &request
		kernelModels.WaitingForDeviceManager.Enqueue(deviceName, pcb)
	} else {
		slog.Info("Dispositivo libre encontrado. Enviando proceso a I/O.", "dispositivo", deviceName, "PID", pcb.PID)
		dispatchToDevice(pcb, device, time)
	}
}

// dispatchToDevice envía la solicitud de I/O al módulo correspondiente.
func dispatchToDevice(pcb *kernelModels.PCB, device *ioModel.Device, time int) {
	device.PID = pcb.PID

	go func() {
		request := struct {
			Pid            uint `json:"pid"`
			SuspensionTime int  `json:"suspensionTime"`
		}{
			Pid:            pcb.PID,
			SuspensionTime: time,
		}

		body, err := json.Marshal(request)
		if err != nil {
			slog.Error("Error al serializar petición de I/O. Finalizando proceso.", "PID", pcb.PID)
			TransitionProcessState(pcb, kernelModels.EstadoExit)
			StartLongTermScheduler()
			return
		}

		_, err = client.DoRequest(device.Port, device.Ip, "POST", "io", body)
		if err != nil {
			slog.Error("Error de comunicación con el módulo de I/O. Finalizando proceso.", "dispositivo", device.Name, "PID", pcb.PID)
			kernelModels.ConnectedDeviceManager.MarkAsFreeByPort(device.Port)
			TransitionProcessState(pcb, kernelModels.EstadoExit)
			StartLongTermScheduler()
		}
	}()
}

// TryToDispatchNextIO revisa la cola de espera de un dispositivo y, si hay procesos, despacha el siguiente.
func TryToDispatchNextIO(deviceName string) {
	slog.Debug("Intentando despachar el próximo proceso para I/O.", "dispositivo", deviceName)

	device, exists := kernelModels.ConnectedDeviceManager.GetFreeByName(deviceName)
	if !exists || device == nil {
		return
	}

	pcb, found := kernelModels.WaitingForDeviceManager.Dequeue(deviceName)
	if !found {
		kernelModels.ConnectedDeviceManager.MarkAsFreeByPort(device.Port)
		return
	}

	slog.Info("Despachando proceso en espera a I/O.", "PID", pcb.PID, "dispositivo", deviceName)

	// Recuperamos el tiempo correcto desde el PCB.
	if pcb.PendingIoRequest == nil {
		slog.Error("El PCB en espera no tenía una solicitud de I/O pendiente. Finalizando.", "PID", pcb.PID)
		TransitionProcessState(pcb, kernelModels.EstadoExit)
		StartLongTermScheduler()
		return
	}

	time, _ := strconv.Atoi(pcb.PendingIoRequest.Values[1])
	pcb.PendingIoRequest = nil // Limpiamos la solicitud pendiente.

	dispatchToDevice(pcb, device, time)
}
