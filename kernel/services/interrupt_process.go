package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

func InterruptExec(pcb models.PCB) {
	//Busca el proceso (PCB) que se esta ejecutando con la mayor rafaga restante estimada
	processToInterrupt := models.GetPCBConMayorRafagaRestante()

	// Validar que exista proceso para interrumpir
	if processToInterrupt == nil {
		slog.Info("No hay procesos ejecutándose para interrumpir")
		return
	}

	if pcb.RafagaEstimada < processToInterrupt.RafagaEstimada {
		//GetCPUByPid recorre las CPUs conectadas y retorna la qe esta ejecutando el PID solicitado
		cpu := models.ConnectedCpuMap.GetCPUByPid(processToInterrupt.PID)
		//SI ES POSITIVO, SE CONECTA AL ENDPOINT DE CPU PARA PEDIRLE QUE DESALOJE AL PROCESO TAL
		SendInterruption(processToInterrupt.PID, cpu.Port, cpu.Ip)
	}
}

func SendInterruption(pid int, portCpu int, ipCpu string) {
	slog.Info("Iniciando pedido de interrupción del proceso", slog.Int("PID", pid))
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
