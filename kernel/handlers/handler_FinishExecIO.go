package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	ioModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
)

func FinishExecIOHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var req ioModels.DeviceResponse
		err := json.NewDecoder(request.Body).Decode(&req)
		if err != nil {
			slog.Warn("Error al decodificar PID desde IO", "error", err)
			http.Error(writer, "PID inválido", http.StatusBadRequest)
			return
		}

		isSuccess, pid := services.FinishDevice(req.Port)

		if !isSuccess {
			slog.Error("Qué rompimos? :(")
			http.Error(writer, "Qué rompimos? :(", http.StatusBadRequest)
			return
		}

		slog.Debug("Solicitud de finalización de IO recibida", "pid", pid)
		slog.Debug(fmt.Sprintf("Motivo de desalojo: %s", req.Reason))

		// Validar si la cola de bloqueados está vacía
		if models.QueueBlocked.Size() == 0 {
			slog.Warn("Cola de bloqueados vacía, no se puede procesar PID", "pid", pid)
			//http.Error(writer, "No hay procesos bloqueados", http.StatusConflict)
			//return
		}

		// Buscar el proceso en la cola de bloqueados
		pcb, index, found := models.QueueBlocked.Find(func(pcb *models.PCB) bool {
			return pcb.PID == uint(pid)
		})

		if found {
			//http.Error(writer, fmt.Sprintf("PID %d no está bloqueado", pid), http.StatusNotFound)
			//return
			models.QueueBlocked.Remove(index)
			slog.Debug("Proceso eliminado de la cola de bloqueados", "pid", pid)
			services.TransitionState(pcb, models.EstadoReady)
			services.AddProcessToReady(pcb)
			writer.WriteHeader(http.StatusOK)
			return
		}

		slog.Warn("Proceso no encontrado en la cola de bloqueados", "pid", pid)

		// Buscar el proceso en la cola de bloqueados
		pcb, index, found = models.QueueSuspBlocked.Find(func(pcb *models.PCB) bool {
			return pcb.PID == uint(pid)
		})

		if !found {
			slog.Warn("Proceso no encontrado en la cola de bloqueado-suspendido", "pid", pid)
			http.Error(writer, fmt.Sprintf("PID %d no está bloqueado-suspendido", pid), http.StatusNotFound)
			return
		}

		slog.Debug("Proceso encontrado en bloqueado-suspendido", "pcb", pcb)

		// Eliminar de la cola de bloqueados
		models.QueueSuspBlocked.Remove(index)
		slog.Debug("Proceso eliminado de la cola de bloqueado-suspendido", "pid", pid)

		// Cambiar estado y pasar a SUSPENDED_READY
		services.TransitionState(pcb, models.EstadoSuspendidoReady)
		models.QueueSuspReady.Add(pcb)
		slog.Debug("Proceso agregado a la cola SUSPENDED_READY", "pid", pid)

		services.NotifyToMediumScheduler()
		writer.WriteHeader(http.StatusOK)
	}
}
