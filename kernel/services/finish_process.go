package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

func FinishProcess(pcb models.PCB) {
	//Conectarse con memoria y enviar PCB
	bodyRequest, err := json.Marshal(pcb)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		panic(err)
	}
	url := fmt.Sprintf("http://%s:%d/memoria/liberarpcb", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PCB a ip:%s puerto:%d", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)
	}
	defer resp.Body.Close()

	//Recibir StatusOK por parte de memoria
	if resp.StatusCode == http.StatusOK {
		slog.Info("Respuesta OK de Memoria")
	} else {
		slog.Warn(fmt.Sprintf("Respuesta NO OK de Memoria. Código: %d.", resp.StatusCode))
	}

	//Logear métricas
	slog.Info(" ## (PID)- Métricas de Estado: NEW NEW_COUNT NEW_TIME READY READY_COUNT READY_TIME")
	slog.Info("Métricas de estado",
		slog.Int("PID", pcb.PID),
		slog.Int("NEW_COUNT", int(pcb.ME[models.EstadoNew])),
		slog.Int("NEW_TIME", int(pcb.MT[models.EstadoNew])),
		slog.Int("READY_COUNT", pcb.ME[models.EstadoReady]),
		slog.Int("READY_TIME", int(pcb.MT[models.EstadoReady])),
	)

	//Liberar PCB asociado
	models.QueueExit.Dequeue()

	//Intentar inicializar un proceso de SUSP READY sino los de NEW
	for models.QueueSuspReady.Size() != 0 {
		pcb, err := models.QueueSuspReady.Dequeue()
		if err != nil {
			slog.Error("Error al hacer Dequeue de SuspReady:", "error", err)
			return
		}
		pcb.EstadoActual = models.EstadoReady
		models.QueueReady.Add(pcb)
	}

	//Ya se hace automaticamente por el planificador de largo plazo en la creación de proceso
	//y allí invoca al planificador de mediano plazo que se encarga de pasar los de susp ready a ready
}
