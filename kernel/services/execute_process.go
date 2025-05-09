package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	cpuModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

func ExecuteProcess(pcb models.PCB, cpu cpuModels.CpuN) {

	var pcbExecute models.PCBExecuteRequest
	pcbExecute.PID = pcb.PID
	pcbExecute.PC = pcb.PC

	//Prepara el Request a Json para envíar a memoria
	bodyRequest, err := json.Marshal(pcbExecute)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		panic(err)
	}

	//Envía los datos
	url := fmt.Sprintf("http://%s:%d/cpu/exec", cpu.Ip, cpu.Port)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PCB", "ip", cpu.Ip, "port", cpu.Port, "err", err)
		models.ConnectedCpuMap.MarkAsFree(fmt.Sprint(cpu.Id))
		return
	}
	defer resp.Body.Close()

	//Recibe PID y motivo de terminación de ejecución
	if resp.StatusCode != http.StatusOK {
		slog.Error("Respuesta inesperada de CPU", "status", resp.StatusCode)
		return
	}

	err = json.NewDecoder(resp.Body).Decode(&pcbExecute)

	if err != nil {
		slog.Error("Error al decodificar el cuerpo de la respuesta de CPU:", "error", err)
		return
	}

	switch pcbExecute.StatusCodePCB {
	case models.NeedFinish:
		//Se ejecutó bien el proceso, así que hay que finalizarlo
		pcb.EstadoActual = models.EstadoExit

	case models.NeedReplan:
		//Se ejecutó bien el proceso, pero aún quedan instrucciones a ejecutar porque fue desalojado
		pcb.PC = pcbExecute.PC
		pcb.EstadoActual = models.EstadoReady
	}

	// Volver a marcar CPU como libre para se pueda re-utilizar
	models.ConnectedCpuMap.MarkAsFree(fmt.Sprint(cpu.Id))

	//Manda a ejecutar al próximo proceso del estado READY
}
