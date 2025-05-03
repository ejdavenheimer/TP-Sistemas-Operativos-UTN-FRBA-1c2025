package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

func ExecuteProcess(pcb models.PCB) {

	//Cambiar a estado EXEC
	pcb.EstadoActual = models.EstadoExecuting

	var pcbExecute models.ExecuteRequest
	pcbExecute.PID = pcb.PID
	pcbExecute.PC = pcb.PC

	//Envíar a módulo CPU conectado y libre el PID y PC a través de endpoint dispatch
	bodyRequest, err := json.Marshal(pcbExecute)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		panic(err)
	}

	//VER CPU CONECTADA
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if !ok {
		slog.Info("No hay CPU libre.")
		return
	}

	//Envía los datos
	url := fmt.Sprintf("http://%s:%d/cpu/exec", cpu.Ip, cpu.Port)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PCB a ip:%s puerto:%d", cpu.Ip, cpu.Port)

		// Volver a marcar CPU como libre por si surgió algún error
		models.ConnectedCpuMap.MarkAsFree(fmt.Sprint(cpu.Id))
	}
	defer resp.Body.Close()

	//Recibe PID y motivo de terminación de ejecución
	if resp.StatusCode == http.StatusOK {
		//Chequear que el PC no sea el último, sino actualizar PC
		pcb.EstadoActual = models.EstadoReady
	} else {
		//INTERRUMPIR EL PROCESO
	}

	//Mandar a ejecutar al próximo proceso del estado READY
}
