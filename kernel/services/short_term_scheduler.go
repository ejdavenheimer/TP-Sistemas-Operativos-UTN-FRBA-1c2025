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
			<-models.NotifyReady // Espera bloqueado hasta recibir señal
			for models.QueueReady.Size() > 0 {
				ok := SelectToExecute()
				if !ok {
					slog.Debug("No se pudo seleccionar ningún proceso para ejecutar.")
					break // Si no hay CPU libre, salgo del for interno y espero nueva señal
				}
			}
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

	pcb, err = models.QueueReady.Get(0)
	index = 0

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
	slog.Debug("Asignando proceso a CPU", "pid", pcb.PID, "cpu_id", cpu.Id)

	go func(pcb *models.PCB, cpu cpuModels.CpuN, key string) {
		inicioEjecucion := time.Now()
		result := ExecuteProcess(pcb, cpu)
		tiempoEjecutado := time.Since(inicioEjecucion)

		if models.KernelConfig.SchedulerAlgorithm == "SJF" || models.KernelConfig.SchedulerAlgorithm == "SRT" {
			pcb.RafagaReal = float32(tiempoEjecutado.Milliseconds())
			// Est(n+1)        =                α          * R(n)           + (1 - α)                      * Est(n)
			pcb.RafagaEstimada = models.KernelConfig.Alpha*pcb.RafagaReal + (1-models.KernelConfig.Alpha)*pcb.RafagaEstimada
		}

		switch result.StatusCodePCB {
		case models.NeedFinish:
			TransitionState(pcb, models.EstadoExecuting, models.EstadoExit)

		case models.NeedReplan:
			TransitionState(pcb, models.EstadoExecuting, models.EstadoReady)
			pcb.PC = result.PC
			AddProcessToReady(pcb)

		case models.NeedInterrupt:
			TransitionState(pcb, models.EstadoExecuting, models.EstadoReady)
			pcb.PC = result.PC
			AddProcessToReady(pcb)
			slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT", pcb.PID))
		}

		// Liberar CPU
		models.ConnectedCpuMap.MarkAsFree(key)
	}(&pcb, *cpu, key)

	return true
}

func ExecuteProcess(pcb *models.PCB, cpu cpuModels.CpuN) models.PCBExecuteRequest {
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

	pcb.Mutex.Lock()
	defer pcb.Mutex.Unlock()

	if newState == models.EstadoBlocked {
		go StartSuspensionTimer(pcb)
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

func StartSuspensionTimer(pcb *models.PCB) {
	slog.Debug("Timer iniciado para posible suspensión de proceso", "PID", pcb.PID)

	time.Sleep(time.Duration(models.KernelConfig.SuspensionTime) * time.Millisecond)

	pcb.Mutex.Lock()
	defer pcb.Mutex.Unlock()

	// Si todavía está en estado BLOCKED, debe pasar a SUSP.BLOCKED
	if pcb.EstadoActual == models.EstadoBlocked {
		// Pasamos a SUSP.BLOCKED
		slog.Info(fmt.Sprintf("## (%d) - Proceso pasa a SUSP.BLOCKED", pcb.PID))
		TransitionState(pcb, models.EstadoBlocked, models.EstadoSuspendidoBlocked)

		models.QueueBlocked.Remove(pcb.PID)
		models.QueueSuspBlocked.Add(*pcb)

		// Informar al módulo MEMORIA para que lo pase a SWAP
		go movePrincipalMemoryToSwap()

	}
}

func AddProcessToReady(pcb *models.PCB) {
	switch models.KernelConfig.SchedulerAlgorithm {
	case "FIFO":
		// En FIFO agrego al final sin ordenar
		models.QueueReady.Add(*pcb)

	case "SJF", "SRT": // Inserto ordenadamente en la cola READY, ordenada por Rafaga ascendente
		procesoInsertado := false
		for i := 0; i < models.QueueReady.Size(); i++ {
			procesoIterado, _ := models.QueueReady.Get(i)
			if pcb.RafagaEstimada < procesoIterado.RafagaEstimada {
				models.QueueReady.Insert(i, *pcb) // Inserto el proceso en la posición i
				procesoInsertado = true
				break
			}
		}
		if !procesoInsertado { // Si no se insertó antes, lo agrego al final
			models.QueueReady.Add(*pcb)
		}

		if models.KernelConfig.SchedulerAlgorithm == "SRT" {
			InterruptExec(*pcb) // Pido interrupción si corresponde
		}
	}
	select {
	case models.NotifyReady <- 1:
		// Se pudo enviar la notificación
	default:
		// Ya hay una notificación pendiente, no hacemos nada
	}

}
