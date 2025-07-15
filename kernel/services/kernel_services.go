package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
)

// este servicio le solicita al dispositivo que duerme por el tiempo que le pasemos.
func SleepDevice(pid uint, timeSleep int, device ioModel.Device) error {
	//Crea y codifica la request de conexion a Kernel
	var request = models.DeviceRequest{Pid: pid, SuspensionTime: timeSleep}
	body, err := json.Marshal(request)

	if err != nil {
		slog.Error("error", slog.String("message", err.Error()))
		return err
	}

	//Envia la request de conexion a Kernel
	slog.Debug(fmt.Sprintf("Enviando syscall a dispositivo %s (%s:%d) - PID: %d - Tiempo: %dms",
		device.Name, device.Ip, device.Port, pid, timeSleep))

	response, err := client.DoRequest(device.Port, device.Ip, "POST", "io", body)
	var deviceResponse ioModel.DeviceResponse

	if response.StatusCode != 200 {
		slog.Error(fmt.Sprintf("status code: %d", response.StatusCode))
		panic(response.Status)
		return errors.New("respuesta inválida")
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	err = json.Unmarshal(responseBody, &deviceResponse)
	if err != nil {
		slog.Error(fmt.Sprintf("error parseando el JSON: %v", err))
		return err
	}

	slog.Debug(fmt.Sprintf("Response: %s", deviceResponse.Reason))

	// se bloquea proceso
	BlockedProcess(pid, deviceResponse.Reason)
	return nil
}

func ExecuteSyscall(syscallRequest models.SyscallRequest, writer http.ResponseWriter) {
	syscallName := syscallRequest.Type
	slog.Info(fmt.Sprintf("## %d - Solicitó syscall: %s", syscallRequest.Pid, syscallName))
	switch syscallName {
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
		additionalArgs := []string{fmt.Sprintf("%d", parentPID)}
		pcb, err := InitProcess(pseudocodeFile, processSize, additionalArgs)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		slog.Debug("Proceso inicializado correctamente", "PID", pcb.PID)

		server.SendJsonResponse(writer, map[string]interface{}{
			"action": "continue",
		})
	case "EXIT":
		EndProcess(syscallRequest.Pid, fmt.Sprintf("Se ejecuta syscall %s", syscallRequest.Type))
		server.SendJsonResponse(writer, map[string]interface{}{
			"action": "exit",
		})
	default:
		slog.Error("Invalid syscall type", slog.String("type", syscallName))
		http.Error(writer, fmt.Sprintf("Tipo de syscall inválido: %s", syscallName), http.StatusBadRequest)
		//panic(fmt.Sprintf("Invalid syscall type: %s", syscallName))
	}
}

func EndProcess(pid uint, reason string) {
	slog.Debug(fmt.Sprintf("[%d] Finaliza el proceso - Motivo: %s", pid, reason))

	pcb, _, exists := models.QueueExec.Find(func(pcb *models.PCB) bool {
		return pcb.PID == pid
	})

	if !exists {
		slog.Error("No se encontró al proceso")
		return
	}

	pcb, isSuccess, err := MoveProcessToState(pcb.PID, models.EstadoExit, false)

	if !isSuccess || err != nil {
		slog.Error(fmt.Sprintf("No se encontro  el proceso <%d>", pid))
		return
	}
	StartLongTermScheduler()

	slog.Debug(fmt.Sprintf("El proceso <%d> se encuentra en estado %s", pcb.PID, pcb.EstadoActual))

	//pcb.EstadoActual = models.EstadoExit
	//models.QueueExit.Add(pcb)
}

func BlockedProcess(pid uint, reason string) {
	slog.Debug(fmt.Sprintf("[%d] Se bloquea el proceso el proceso - Motivo: %s", pid, reason))
	// Para que un proceso se bloquee tiene que estar en ejecución.
	pcb, index, exists := models.QueueExec.Find(func(pcb *models.PCB) bool {
		return pcb.PID == pid
	})

	if index == -1 || !exists {
		slog.Error(fmt.Sprintf("No se encontro  el proceso <%d>", index))
		return
	}

	if strings.Contains(reason, "no se encuentra disponible") {
		pcb.PC--
		err := models.QueueExec.Set(index, pcb)
		if err != nil {
			slog.Error(fmt.Sprintf("error al modificar el proceso %d", pcb.PID))
		}
	}
	//models.QueueExec.Remove(index)

	pcb, isSuccess, err := MoveProcessToState(pcb.PID, models.EstadoBlocked, false)
	//pcb.EstadoActual = models.EstadoBlocked
	if !isSuccess || err != nil {
		slog.Error(fmt.Sprintf("No se encontro el proceso <%d>", pid))
		return
	}

	// LOG OBLIGATORIO DE BLOQUEO POR IO
	slog.Info(fmt.Sprintf("## (%d) - Bloqueado por IO: %s", pid, reason))

	cpu := models.ConnectedCpuMap.GetCPUByPid(pcb.PID)

	if cpu != nil {
		//	slog.Error(fmt.Sprintf("No se encontro la CPU para el proceso <%d>", pid))
		models.ConnectedCpuMap.MarkAsFree(strconv.Itoa(cpu.Id))
	}

	StartSuspensionTimer(pcb)
}

func DumpServices(pid uint, size int) {
	var request = memoriaModel.DumpMemoryRequest{Pid: pid, Size: size}
	body, err := json.Marshal(request)

	if err != nil {
		slog.Error("error", slog.String("message", err.Error()))
		return
	}

	BlockedProcess(pid, "dumping services")

	response, err := client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/dump-memory", body)
	var dumpMemoryResponse memoriaModel.DumpMemoryResponse

	if err != nil || response.StatusCode != 200 {
		slog.Error(fmt.Sprintf("error: %v", err))
		EndProcess(pid, "DUMP MEMORY")
		return
	}

	responseBody, _ := io.ReadAll(response.Body)
	slog.Debug(fmt.Sprintf("Response: %s", string(responseBody)))

	err = json.Unmarshal(responseBody, &dumpMemoryResponse)
	if err != nil {
		slog.Error(fmt.Sprintf("error parseando el JSON: %v", err))
		return
	}

	_, isSuccess, err := MoveProcessToState(pid, models.EstadoExit, false)
	if !isSuccess || err != nil {
		slog.Error("Qué rompimos? :(")
		return
	}

	//pcb, _, exists := models.QueueBlocked.Find(func(pcb models.PCB) bool {
	//	return pcb.PID == int(pid)
	//})
	//
	//if !exists {
	//	slog.Error("No se encontró al proceso")
	//	return
	//}
	//
	//pcb.EstadoActual = models.EstadoReady
	//models.QueueReady.Add(pcb)

	slog.Debug(fmt.Sprintf("Response: %s", dumpMemoryResponse.Result))
}

func FinishDevice(port int) (bool, int) {
	deviceRequested, index, _ := models.ConnectedDeviceList.Find(func(d ioModel.Device) bool {
		return port == d.Port && !d.IsFree
	})

	if index == -1 {
		slog.Debug("No se encontró el dispositivo")
		return false, -1
	}

	deviceRequested.IsFree = true

	err := models.ConnectedDeviceList.Set(index, deviceRequested)
	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		return false, -1
	}

	return true, int(deviceRequested.PID)
}
