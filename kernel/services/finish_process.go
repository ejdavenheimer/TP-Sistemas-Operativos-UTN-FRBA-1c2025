package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// func FinishProcess(pcb models.PCB) {
func FinishProcess() {
	pcb, err := models.QueueExit.Dequeue()
	if err != nil {
		slog.Error(fmt.Sprintf("Error al sacar pcb de QueueExit, ya que está vacía: %v", err))
		return
	}

	slog.Debug("Iniciando finalización del proceso", "PID", pcb.PID)
	//Conectarse con memoria y enviar PCB
	bodyRequest, err := json.Marshal(pcb.PID)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		panic(err)
	}
	url := fmt.Sprintf("http://%s:%d/memoria/liberarpcb", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)
	slog.Debug("Enviando PCB a memoria", slog.String("url", url))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PCB a ip:%s puerto:%d", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)
	}
	defer resp.Body.Close()

	//Recibir StatusOK por parte de memoria
	if resp.StatusCode == http.StatusOK {
		slog.Info(fmt.Sprintf("## (<%d>) - Finaliza el proceso", pcb.PID))
		slog.Debug("Memoria respondió OK al liberar PCB")
	} else {
		slog.Warn("Memoria respondió con error al liberar PCB", slog.Int("status", resp.StatusCode))
	}

	//Logear métricas
	// TODO: no coincide con log obligatorio, creo que el obligatorio esta dando vuelta
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

	//Liberar PCB asociado
	slog.Debug("Liberado PCB de la cola de EXIT")

	//Intentar inicializar un proceso de SUSP READY sino los de NEW
	//Ya lo hace el plani de mediano plazo

}
