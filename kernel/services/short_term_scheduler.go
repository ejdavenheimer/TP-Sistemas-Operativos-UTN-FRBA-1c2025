package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	cpuModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// INICIA PLANIFICADOR DE CORTO PLAZO
func StartShortTermScheduler() {
	go func() {
		for {
			if models.QueueReady.Size() > 0 {
				ok := SelectToExecute()
				if !ok {
					slog.Debug("No se pudo seleccionar ningún proceso para ejecutar.")
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
}

// Elige el siguiente proceso de READY y lo envía a ejecución
func SelectToExecute() bool {
	slog.Debug("Buscando CPU libre para ejecutar proceso...")

	//VER CPU CONECTADA
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if !ok {
		slog.Debug("No hay CPU disponible.")
		return false // No hay CPU libre, no se puede ejecutar un proceso
	}

	//Si la cola READY está vacía, no hay procesos listos para ejecutar.
	if models.QueueReady.Size() == 0 {
		slog.Debug("No hay procesos en READY.")
		return false // No hay procesos en READY, no se puede ejecutar nada
	}

	//FIFO: obtengo el proceso a ejecutar, el primero de la cola READY
	pcb, err := models.QueueReady.Get(0)
	if err != nil {
		slog.Warn("Error obteniendo proceso de la cola READY:", "error", err)
		return false //Error al obtener el proceso
	}

	//Lo saca de la cola READY. Ya está listo para ejecutarse.
	models.QueueReady.Remove(0)
	TransitionState(&pcb, models.EstadoReady, models.EstadoExecuting)

	// Asigna el proceso a la CPU
	cpu.PIDExecuting = pcb.PID
	key := strconv.Itoa(cpu.Id)
	models.ConnectedCpuMap.Set(key, cpu)

	slog.Debug("CPU marcada como ocupada", "cpu_id", cpu.Id, "pid", cpu.PIDExecuting)
	slog.Debug(fmt.Sprintf("Asignando proceso PID=%d a CPU ID=%d", pcb.PID, cpu.Id))

	go func(pcb *models.PCB, cpu cpuModels.CpuN, key string) {
		result := ExecuteProcess(*pcb, cpu)

		switch result.StatusCodePCB {
		case models.NeedFinish:
			TransitionState(pcb, models.EstadoExecuting, models.EstadoExit)

		case models.NeedReplan:
			TransitionState(pcb, models.EstadoExecuting, models.EstadoReady)
			pcb.PC = result.PC
			models.QueueReady.Add(*pcb)
		}

		// Liberar CPU
		models.ConnectedCpuMap.MarkAsFree(key)
	}(pcb, cpu, key)

	return true
}

func ExecuteProcess(pcb models.PCB, cpu cpuModels.CpuN) models.PCBExecuteRequest {
	var pcbExecute models.PCBExecuteRequest
	pcbExecute.PID = pcb.PID
	pcbExecute.PC = pcb.PC

	//Prepara el Request a Json para envíar a cpu
	bodyRequest, err := json.Marshal(pcbExecute)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		panic(err)
	}

	//Envía a CPU
	url := fmt.Sprintf("http://%s:%d/cpu/exec", cpu.Ip, cpu.Port)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PCB", "ip", cpu.Ip, "port", cpu.Port, "err", err)
		models.ConnectedCpuMap.MarkAsFree(fmt.Sprint(cpu.Id))
		return models.PCBExecuteRequest{StatusCodePCB: models.NeedReplan}
	}
	defer resp.Body.Close()

	//Recibe PID y motivo de terminación de ejecución. (Verifica status HTTP)
	if resp.StatusCode != http.StatusOK {
		slog.Error("Respuesta inesperada de CPU", "status", resp.StatusCode)
		return models.PCBExecuteRequest{StatusCodePCB: models.NeedReplan}
	}
	//Decodificar respuesta JSON
	err = json.NewDecoder(resp.Body).Decode(&pcbExecute)
	if err != nil {
		slog.Error("Error al decodificar el cuerpo de la respuesta de CPU:", "error", err)
		return models.PCBExecuteRequest{StatusCodePCB: models.NeedReplan}
	}
	return pcbExecute
}

// Actualiza el estado de un proceso, su LOG y sus metricas.
func TransitionState(pcb *models.PCB, oldState models.Estado, newState models.Estado) {
	if oldState == newState {
		return
	}

	if pcb.ME == nil {
		pcb.ME = make(map[models.Estado]int)
	}
	if pcb.MT == nil {
		pcb.MT = make(map[models.Estado]time.Duration)
	}

	//Calculas el tiempo que estuvo en el estado anterior
	duration := time.Since(pcb.UltimoCambio)
	pcb.MT[oldState] += duration
	// Incrementa en 1 la cantidad de veces que el proceso estuvo en el estado anterior
	pcb.ME[oldState]++

	//Log de cambio de estado
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado %s", pcb.PID, oldState, newState))

	//Registramos el cambio en las metricas
	pcb.EstadoActual = newState
	pcb.UltimoCambio = time.Now()
}
