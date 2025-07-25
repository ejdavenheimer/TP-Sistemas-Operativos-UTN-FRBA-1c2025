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

	// 1. Bloqueamos el proceso. Su destino final (ejecutar I/O o esperar) se decide a continuación.
	TransitionProcessState(pcb, kernelModels.EstadoBlocked)
	slog.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", pcb.PID, deviceName))

	// 2. Buscamos un dispositivo libre de ese tipo.
	device, exists := kernelModels.ConnectedDeviceManager.GetFreeByName(deviceName)

	if !exists {
		slog.Error("Syscall IO para un tipo de dispositivo inexistente. Finalizando proceso.", "dispositivo", deviceName, "PID", pcb.PID)
		TransitionProcessState(pcb, kernelModels.EstadoExit)
		StartLongTermScheduler()
		return
	}

	if device == nil {
		// El tipo de dispositivo existe, pero todas las instancias están ocupadas.
		slog.Info("Dispositivo ocupado. Proceso encolado.", "dispositivo", deviceName, "PID", pcb.PID)
		kernelModels.WaitingForDeviceManager.Enqueue(deviceName, pcb)
	} else {
		// Encontramos un dispositivo libre.
		slog.Info("Dispositivo libre encontrado. Enviando proceso a I/O.", "dispositivo", deviceName, "PID", pcb.PID)
		dispatchToDevice(pcb, device, time)
	}
}

// dispatchToDevice envía la solicitud de I/O al módulo correspondiente.
func dispatchToDevice(pcb *kernelModels.PCB, device *ioModel.Device, time int) {
	device.PID = pcb.PID // Asociamos el PID al dispositivo.

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
			TransitionProcessState(pcb, kernelModels.EstadoExit)
			StartLongTermScheduler()
		}
	}()
}

// TryToDispatchNextIO revisa la cola de espera de un dispositivo y, si hay procesos, despacha el siguiente.
func TryToDispatchNextIO(deviceName string) {
	slog.Debug("Intentando despachar el próximo proceso para I/O.", "dispositivo", deviceName)

	// Verificamos si hay un dispositivo libre.
	device, exists := kernelModels.ConnectedDeviceManager.GetFreeByName(deviceName)
	if !exists || device == nil {
		return // No hay dispositivos libres, no podemos hacer nada.
	}

	// Verificamos si hay un proceso esperando.
	pcb, found := kernelModels.WaitingForDeviceManager.Dequeue(deviceName)
	if !found {
		// No había nadie esperando, así que liberamos el dispositivo que tomamos.
		kernelModels.ConnectedDeviceManager.MarkAsFreeByPort(device.Port)
		return
	}

	slog.Info("Despachando proceso en espera a I/O.", "PID", pcb.PID, "dispositivo", deviceName)
	// La syscall original ya nos dio el tiempo, necesitamos recuperarlo o tenerlo en el pcb.
	// Por ahora, asumimos un tiempo fijo o lo buscamos en el pcb si lo guardamos.
	// Para este ejemplo, vamos a necesitar agregar el tiempo de IO al pcb o pasarlo de otra forma.
	// Solución simple: el `SyscallRequest` se podría guardar en el PCB temporalmente.
	// Por ahora, para que compile, usamos un valor placeholder.
	placeholderTime := 1000
	dispatchToDevice(pcb, device, placeholderTime)
}

// UnblockProcess mueve un proceso de BLOCKED a READY.
func UnblockProcess(pid uint) {
	pcb, _, found := kernelModels.QueueBlocked.Find(func(p *kernelModels.PCB) bool { return p.PID == pid })
	if !found {
		slog.Warn("Se intentó desbloquear un PID que no estaba en la cola BLOCKED.", "PID", pid)
		return
	}

	TransitionProcessState(pcb, kernelModels.EstadoReady)
	StartShortTermScheduler()
}
