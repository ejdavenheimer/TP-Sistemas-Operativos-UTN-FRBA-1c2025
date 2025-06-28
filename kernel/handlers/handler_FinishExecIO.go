package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

func FinishExecIOHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var pid uint
		err := json.NewDecoder(request.Body).Decode(&pid)
		if err != nil {
			slog.Warn("Error al decodificar PID desde IO", "error", err)
			http.Error(writer, "PID inválido", http.StatusBadRequest)
			return
		}

		slog.Info("Solicitud de finalización de IO recibida", "pid", pid)

		// Validar si la cola de bloqueados está vacía
		if models.QueueBlocked.Size() == 0 {
			slog.Warn("Cola de bloqueados vacía, no se puede procesar PID", "pid", pid)
			http.Error(writer, "No hay procesos bloqueados", http.StatusConflict)
			return
		}

		// Buscar el proceso en la cola de bloqueados
		pcb, index, found := models.QueueBlocked.Find(func(pcb *models.PCB) bool {
			return pcb.PID == pid
		})

		if !found {
			slog.Warn("Proceso no encontrado en la cola de bloqueados", "pid", pid)
			http.Error(writer, fmt.Sprintf("PID %d no está bloqueado", pid), http.StatusNotFound)
			return
		}

		slog.Debug("Proceso encontrado en bloqueados", "pcb", pcb)

		// Eliminar de la cola de bloqueados
		models.QueueBlocked.Remove(index)
		slog.Info("Proceso eliminado de la cola de bloqueados", "pid", pid)

		// Cambiar estado y pasar a SUSPENDED_READY
		pcb.EstadoActual = models.EstadoSuspendidoReady
		models.QueueSuspReady.Add(pcb)
		slog.Info("Proceso agregado a la cola SUSPENDED_READY", "pid", pid)

		writer.WriteHeader(http.StatusOK)
	}
}
