package services

import (
	"fmt"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/helpers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"log/slog"
	"strconv"
)

func ExecuteIO(result models.PCBExecuteRequest) {
	// Primero chequea si existe el dispositivo
	deviceRequested, index, exists := models.ConnectedDeviceList.Find(func(d ioModel.Device) bool {
		return result.SyscallRequest.Values[0] == d.Name
	})

	if !exists || deviceRequested.Name == "" {
		slog.Error(fmt.Sprintf("No se encontro al dispositivo %s", result.SyscallRequest.Values[0]))
		EndProcess(result.SyscallRequest.Pid, fmt.Sprintf("No se encontro al dispositivo %s", result.SyscallRequest.Values[0]))
		//server.SendJsonResponse(writer, map[string]interface{}{
		//	"action": "exit",
		//})
		return
	}

	// Se crea una lista auxiliar con dispositivos del mismo tipo
	connectedDeviceListAux := models.ConnectedDeviceList.FindAll(func(d ioModel.Device) bool {
		return result.SyscallRequest.Values[0] == d.Name
	})

	// Se busca los dispositivos que se encuentren libres
	deviceRequestedAux, indexAux, exists := connectedDeviceListAux.Find(func(d ioModel.Device) bool {
		return d.IsFree
	})

	// En caso de que no encuentre ningún dispositivo libre, se bloquea el proceso
	if index < 0 || !exists {
		slog.Debug("El dispositivo se encuentra ocupado...")
		helpers.AddPidForDevice(deviceRequested.Name, int(result.SyscallRequest.Pid))
		BlockedProcess(result.SyscallRequest.Pid, fmt.Sprintf("El dispositivo %s no se encuentra disponible", result.SyscallRequest.Values[0]))
		//server.SendJsonResponse(writer, map[string]interface{}{
		//	"action": "block",
		//})
		return
	}

	deviceRequestedAux.IsFree = false
	deviceRequestedAux.PID = result.SyscallRequest.Pid
	err := models.ConnectedDeviceList.Set(index+indexAux, deviceRequestedAux)
	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return
	}

	device := ioModel.Device{Ip: deviceRequestedAux.Ip, Port: deviceRequestedAux.Port, Name: result.SyscallRequest.Values[0]}
	sleepTime, _ := strconv.Atoi(result.SyscallRequest.Values[1])
	err = SleepDevice(result.SyscallRequest.Pid, sleepTime, device)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return
	}

	//Acá se bloquea el proceso
	//server.SendJsonResponse(writer, map[string]interface{}{
	//	"action": "block",
	//})
}

func ExecuteDUMP(result models.PCBExecuteRequest) {
	pcb, index, exists := models.QueueExec.Find(func(pcb *models.PCB) bool {
		return pcb.PID == result.SyscallRequest.Pid
	})
	if !exists || index == -1 {
		slog.Warn("TODO: ver que pasa en este caso por ahora hago un exit")
		//server.SendJsonResponse(writer, map[string]string{
		//	"action": "exit",
		//})
		return
	}
	DumpServices(uint(pcb.PID), pcb.Size)
	//server.SendJsonResponse(writer, map[string]string{
	//	"action": "continue",
	//})
}
