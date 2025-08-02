package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// FinishProcess procesa la cola de EXIT. Se encarga de la comunicación con Memoria
// y de loguear las métricas finales del proceso.
func FinishProcess() {
	// 1. Saca el próximo proceso de la cola de finalizados.
	pcb, err := models.QueueExit.Dequeue()
	if err != nil {
		// Es normal que la cola esté vacía, no es un error crítico.
		slog.Debug("Se intentó finalizar un proceso, pero la cola EXIT está vacía.")
		return
	}

	slog.Debug("Iniciando finalización del proceso", "PID", pcb.PID)

	// 2. Informa a Memoria que libere los recursos del proceso.
	bodyRequest, err := json.Marshal(pcb.PID)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al serializar el PID para Memoria: %v", err))
		return
	}
	url := fmt.Sprintf("http://%s:%d/memoria/liberarpcb", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando solicitud de liberación a Memoria", "PID", pcb.PID, "error", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			slog.Debug("Memoria respondió OK al liberar PCB", "PID", pcb.PID)
		} else {
			slog.Warn("Memoria respondió con error al liberar PCB", "PID", pcb.PID, "status", resp.StatusCode)
		}
	}

	// 3. Loguea las métricas finales, como pide el enunciado.
	slog.Info(fmt.Sprintf("## (<%d>) - Finaliza el proceso", pcb.PID))
	slog.Info(fmt.Sprintf("## PID: (<%d>) - Métricas de estado - NEW_COUNT: %d; NEW_TIME_MS: %d; READY_COUNT: %d; READY_TIME_MS: %d; BLOCKED_COUNT: %d; BLOCKED_TIME_MS: %d; EXEC_COUNT: %d; EXEC_TIME_MS: %d; SUSP_BLOCKED_COUNT: %d; SUSP_BLOCKED_TIME_MS: %d; SUSP_READY_COUNT: %d; SUSP_READY_TIME_MS: %d",
		pcb.PID,
		pcb.ME[models.EstadoNew],
		pcb.MT[models.EstadoNew].Milliseconds(),
		pcb.ME[models.EstadoReady],
		pcb.MT[models.EstadoReady].Milliseconds(),
		pcb.ME[models.EstadoBlocked],
		pcb.MT[models.EstadoBlocked].Milliseconds(),
		pcb.ME[models.EstadoExecuting],
		pcb.MT[models.EstadoExecuting].Milliseconds(),
		pcb.ME[models.EstadoSuspendidoBlocked],
		pcb.MT[models.EstadoSuspendidoBlocked].Milliseconds(),
		pcb.ME[models.EstadoSuspendidoReady],
		pcb.MT[models.EstadoSuspendidoReady].Milliseconds(),
	))

	slog.Debug("Recursos del PCB liberados. Finalización completa.", "PID", pcb.PID)
}
