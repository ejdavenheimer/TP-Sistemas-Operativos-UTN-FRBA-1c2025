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
	if err == nil {
		slog.Error(fmt.Sprintf("Error al sacar pcb de QueueExit, ya que está vacía: %v", err))
		return
	}

	slog.Info("Iniciando finalización del proceso", slog.Int("PID", pcb.PID))
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
		slog.Info("Memoria respondió OK al liberar PCB")
	} else {
		slog.Warn("Memoria respondió con error al liberar PCB", slog.Int("status", resp.StatusCode))
	}

	//Logear métricas
	slog.Info("Métricas de estado",
		slog.Int("PID", pcb.PID),
		slog.Int("NEW_COUNT", int(pcb.ME[models.EstadoNew])),
		slog.Int("NEW_TIME", int(pcb.MT[models.EstadoNew])),
		slog.Int("READY_COUNT", pcb.ME[models.EstadoReady]),
		slog.Int("READY_TIME", int(pcb.MT[models.EstadoReady])),
	)

	//Liberar PCB asociado
	slog.Info("Liberado PCB de la cola de EXIT")

	//Intentar inicializar un proceso de SUSP READY sino los de NEW
	//Ya lo hace el plani de mediano plazo

}
