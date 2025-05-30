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

	//Verificar CPUs disponibles
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if !ok {
		slog.Debug("No hay CPU disponible.")
		return false // No hay CPU libre, no se puede ejecutar un proceso
	}

	var pcb models.PCB
	var index int = -1 // inicializo en -1, valor default si da error
	var err error

	switch models.KernelConfig.SchedulerAlgorithm {
	case "FIFO":
		pcb, err = models.QueueReady.Get(0) //obtengo el proceso a ejecutar, el primero de la cola READY
		index = 0
	case "SJF":
		pcb, index, err = getShortestJob()
	case "SRT":
		pcb, index, err = getShortestJob()
	}

	if err != nil {
		slog.Warn("Error obteniendo proceso de la cola READY:", "error", err)
		return false //Error al obtener el proceso
	}
	//Lo saca de la cola READY. Ya está listo para ejecutarse.
	models.QueueReady.Remove(index)
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
			AddProcessToReady(*pcb)

		case models.NeedInterrupt:
			TransitionState(pcb, models.EstadoExecuting, models.EstadoReady)
			pcb.PC = result.PC
			AddProcessToReady(*pcb)
			slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", pcb.PID))
		}

		// Liberar CPU
		models.ConnectedCpuMap.MarkAsFree(key)
	}(&pcb, cpu, key)

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

func getShortestJob() (models.PCB, int, error) {
	if models.QueueReady.Size() == 0 {
		return models.PCB{}, -1, fmt.Errorf("Cola READY vacía")
	}
	shortestIndex := 0
	shortestJob, _ := models.QueueReady.Get(shortestIndex)

	for i := 1; i < models.QueueReady.Size(); i++ {
		job, _ := models.QueueReady.Get(i)
		if job.Rafaga < shortestJob.Rafaga {
			shortestJob = job
			shortestIndex = i
		}
	}
	return shortestJob, shortestIndex, nil
}

func AddProcessToReady(pcb models.PCB) {
	switch models.KernelConfig.SchedulerAlgorithm {
	case "FIFO":
		// En FIFO agrego al final sin ordenar
		models.QueueReady.Add(pcb)

	case "SJF", "SRT": // Inserto ordenadamente en la cola READY, ordenada por Rafaga ascendente
		inserted := false
		for i := 0; i < models.QueueReady.Size(); i++ {
			proc, _ := models.QueueReady.Get(i)
			if pcb.Rafaga < proc.Rafaga {
				models.QueueReady.Insert(i, pcb) // Inserto el proceso en la posición i
				inserted = true
				break
			}
		}
		if !inserted { // Si no se insertó antes, lo agrego al final
			models.QueueReady.Add(pcb)
		}

		if models.KernelConfig.SchedulerAlgorithm == "SRT" {
			InterruptExec(pcb) // Pido interrupción si corresponde
		}
	}
}
