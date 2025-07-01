package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

func SendInterruption(pid uint, portCpu int, ipCpu string) {
	slog.Info("Iniciando pedido de interrupción del proceso", "PID", pid)
	//Conectarse con cpu y enviar PID
	bodyRequest, err := json.Marshal(pid)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pid: %v", err))
		return
	}
	url := fmt.Sprintf("http://%s:%d/cpu/interrupt", ipCpu, portCpu)
	slog.Debug("Enviando PID a cpu", slog.String("url", url))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PID a ip:%s puerto:%d", ipCpu, portCpu)
		return
	}
	defer resp.Body.Close()

	//Recibir StatusOK por parte de CPU
	if resp.StatusCode == http.StatusOK {
		slog.Info("CPU respondió OK para desalojar el PCB")
	} else {
		slog.Warn("CPU respondió con error al desalojar el PCB", slog.Int("status", resp.StatusCode))
	}
}

func GetPCBConMayorRafagaRestante() *models.PCB { // De los procesos en ejecución
	var max *models.PCB
	size := models.QueueExec.Size() // Guarda el tamaño actual de la lista QueueExec

	for i := 0; i < size; i++ {
		pcb, err := models.QueueExec.Get(i)
		if err != nil {
			continue
		}
		rafagaRestante := pcb.RafagaEstimada - pcb.RafagaReal
		if max == nil || rafagaRestante > (max.RafagaEstimada-max.RafagaReal) {
			max = pcb
		}
	}
	return max
}
