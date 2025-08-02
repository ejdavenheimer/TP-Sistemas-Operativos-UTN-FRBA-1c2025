package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

// ConnectIoHandler maneja la conexión de un nuevo dispositivo de I/O.
func ConnectIoHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var device ioModel.Device
		if err := json.NewDecoder(r.Body).Decode(&device); err != nil {
			http.Error(w, "Datos de dispositivo inválidos", http.StatusBadRequest)
			return
		}
		device.IsFree = true
		models.ConnectedDeviceManager.Add(&device)
		slog.Info("Dispositivo de I/O conectado", "nombre", device.Name, "puerto", device.Port)

		// CORRECCIÓN: Damos una pequeña pausa para que el servidor del I/O termine de levantarse.
		// Esto previene la condición de carrera al conectar un nuevo dispositivo.
		time.Sleep(100 * time.Millisecond)

		// Ahora sí, intentamos despachar un proceso que pudiera estar esperando.
		services.TryToDispatchNextIO(device.Name)

		w.WriteHeader(http.StatusOK)
	}
}

// FinishIoHandler maneja la notificación de fin de I/O de un dispositivo.
func FinishIoHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var response ioModel.DeviceResponse
		if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
			http.Error(w, "Respuesta de dispositivo inválida", http.StatusBadRequest)
			return
		}

		slog.Info(fmt.Sprintf("## PID: (<%d>) finalizó IO y pasa a READY", response.Pid))

		device, found := models.ConnectedDeviceManager.MarkAsFreeByPort(response.Port)
		if !found {
			slog.Warn("Se recibió fin de I/O de un dispositivo no registrado.", "puerto", response.Port)
		}

		// CORRECCIÓN: Usamos la nueva función de desbloqueo inteligente UnblockProcessAfterIO.
		services.UnblockProcessAfterIO(response.Pid)

		// Intentamos despachar al siguiente proceso en la cola de espera.
		if found {
			services.TryToDispatchNextIO(device.Name)
		}

		w.WriteHeader(http.StatusOK)
	}
}

// DisconnectIoHandler maneja la desconexión de un dispositivo de I/O.
func DisconnectIoHandler() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var response ioModel.DeviceResponse
		if err := json.NewDecoder(r.Body).Decode(&response); err != nil {
			http.Error(w, "Respuesta de dispositivo inválida", http.StatusBadRequest)
			return
		}

		slog.Info("## Dispositivo de I/O desconectado", "nombre", response.Name, "puerto", response.Port)

		// 1. Identificar el proceso que estaba en ejecución en este dispositivo y finalizarlo
		executingPID := models.ConnectedDeviceManager.GetPidByPort(response.Port)
		if executingPID != 0 {
			pcb, found := services.FindPCBInAnyQueue(executingPID)
			if found {
				slog.Warn("Finalizando proceso en ejecución por desconexión de IO", "PID", pcb.PID, "dispositivo", response.Name)
				services.TransitionProcessState(pcb, models.EstadoExit)
				services.StartLongTermScheduler()
			}
		}

		// 2. Remover el dispositivo de la lista de conectados
		models.ConnectedDeviceManager.RemoveByPort(response.Port)

		// 3. Verificar si quedan más instancias de este tipo de dispositivo
		_, exists := models.ConnectedDeviceManager.GetFreeByName(response.Name)

		// 4. Si no quedan más instancias, finalizar todos los procesos en espera para este dispositivo
		if !exists {
			slog.Warn("No quedan más instancias del dispositivo. Finalizando todos los procesos en espera.", "dispositivo", response.Name)

			// Limpiar cola de espera específica del dispositivo
			for {
				pcb, found := models.WaitingForDeviceManager.Dequeue(response.Name)
				if !found {
					break // La cola está vacía
				}
				slog.Warn("Finalizando proceso en cola de espera por desconexión de IO", "PID", pcb.PID, "dispositivo", response.Name)
				services.TransitionProcessState(pcb, models.EstadoExit)
				services.StartLongTermScheduler()
			}

			// También es necesario revisar las colas BLOCKED y SUSPEND_BLOCKED por si algún proceso
			// quedó allí esperando por este tipo de dispositivo genérico.
			queuesToClean := []*list.ArrayList[*models.PCB]{
				models.QueueBlocked,
				models.QueueSuspBlocked,
			}

			for _, queue := range queuesToClean {
				// Es importante obtener una copia para no modificar la lista mientras se itera
				allProcesses := queue.GetAll()
				for _, pcb := range allProcesses {
					if pcb.PendingIoRequest != nil && pcb.PendingIoRequest.Values[0] == response.Name {
						slog.Warn("Finalizando proceso en cola de bloqueo general por desconexión de IO", "PID", pcb.PID, "dispositivo", response.Name)
						services.TransitionProcessState(pcb, models.EstadoExit)
						services.StartLongTermScheduler()
					}
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}
