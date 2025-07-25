package services

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	kernelModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

// --- Funciones del Planificador ---

// StartShortTermScheduler notifica al planificador que hay trabajo en la cola READY.
func StartShortTermScheduler() {
	select {
	case kernelModels.NotifyReady <- 1:
	default:
	}
}

// ShortTermScheduler es el ciclo principal del planificador de corto plazo.
func ShortTermScheduler() {
	for {
		// 1. Espera una notificación para activarse.
		<-kernelModels.NotifyReady
		slog.Debug("PCP: Planificador activado, hay procesos en READY.")

		// 2. Intenta despachar procesos mientras haya en READY y CPUs libres.
		dispatchAvailableProcesses()
	}
}

// dispatchAvailableProcesses busca una CPU libre y, si la hay, despacha un proceso.
func dispatchAvailableProcesses() {
	for kernelModels.QueueReady.Size() > 0 {
		// 3. Busca una CPU libre ANTES de seleccionar un proceso.
		cpu, found := kernelModels.ConnectedCpuMap.GetFirstFree()
		if !found {
			slog.Debug("PCP: No hay CPUs libres en este momento. Esperando notificación.")
			return // No hay CPUs, salimos y esperamos un nuevo aviso.
		}

		// 4. Si hay CPU, seleccionamos un proceso según el algoritmo.
		pcb := selectProcessToExecute()
		if pcb == nil {
			slog.Warn("PCP: Se esperaba un proceso en READY pero no se obtuvo. Liberando CPU.")
			kernelModels.ConnectedCpuMap.MarkAsFree(cpu.Id)
			continue
		}

		// 5. Despachamos el proceso a la CPU en una goroutine para no bloquear el planificador.
		go handleCpuExecution(pcb, cpu)
	}
}

// --- Lógica de Selección y Despacho ---

// selectProcessToExecute contiene el SWITCH para los algoritmos de planificación.
func selectProcessToExecute() *kernelModels.PCB {
	var pcb *kernelModels.PCB
	var err error

	switch kernelModels.KernelConfig.SchedulerAlgorithm {
	case "FIFO":
		pcb, err = scheduleFIFO()
	case "SJF", "SRT":
		pcb, err = scheduleShortestJobFirst()
	default:
		slog.Error("PCP: Algoritmo no reconocido.", "algoritmo", kernelModels.KernelConfig.SchedulerAlgorithm)
		return nil
	}

	if err != nil {
		slog.Error("PCP: Error al obtener proceso de la cola READY.", "error", err)
		return nil
	}
	return pcb
}

func scheduleFIFO() (*kernelModels.PCB, error) {
	slog.Debug("PCP (FIFO): Seleccionando primer proceso de la cola READY.")
	return kernelModels.QueueReady.Dequeue()
}

func scheduleShortestJobFirst() (*kernelModels.PCB, error) {
	slog.Debug("PCP (SJF/SRT): Buscando proceso con la ráfaga más corta en READY.")
	if kernelModels.QueueReady.Size() == 0 {
		return nil, fmt.Errorf("la cola READY está vacía")
	}

	allReadyProcesses := kernelModels.QueueReady.GetAll()
	if len(allReadyProcesses) == 0 {
		return nil, fmt.Errorf("error al obtener procesos de la cola READY")
	}

	// Encontramos el índice del proceso con la menor estimación.
	shortestIndex := 0
	for i := 1; i < len(allReadyProcesses); i++ {
		if allReadyProcesses[i].RafagaEstimada < allReadyProcesses[shortestIndex].RafagaEstimada {
			shortestIndex = i
		}
	}

	shortestPcb := allReadyProcesses[shortestIndex]
	kernelModels.QueueReady.Remove(shortestIndex) // Lo sacamos de la cola.

	slog.Debug("PCP (SJF/SRT): Proceso seleccionado.", "PID", shortestPcb.PID, "Estimación", shortestPcb.RafagaEstimada)
	return shortestPcb, nil
}

func handleCpuExecution(pcb *kernelModels.PCB, cpu *models.CpuN) {
	cpu.PIDExecuting = pcb.PID
	kernelModels.ConnectedCpuMap.Set(strconv.Itoa(cpu.Id), cpu)

	rafagaStartTime := time.Now()
	TransitionProcessState(pcb, kernelModels.EstadoExecuting)
	result := sendProcessToExecute(pcb, cpu)

	rafagaRealDuration := time.Since(rafagaStartTime)
	pcb.RafagaReal = float32(rafagaRealDuration.Milliseconds())

	alpha := kernelModels.KernelConfig.Alpha
	pcb.RafagaEstimada = (alpha * pcb.RafagaReal) + ((1 - alpha) * pcb.RafagaEstimada)

	pcb.PC = result.PC

	switch result.StatusCodePCB {
	case kernelModels.NeedFinish:
		slog.Debug("PCP: CPU informó que el proceso debe finalizar.", "PID", pcb.PID)
		TransitionProcessState(pcb, kernelModels.EstadoExit)
		StartLongTermScheduler()

	case kernelModels.NeedReplan, kernelModels.NeedInterrupt:
		slog.Debug("PCP: CPU devolvió el proceso para replanificación/interrupción.", "PID", pcb.PID)
		TransitionProcessState(pcb, kernelModels.EstadoReady)
		StartShortTermScheduler()

	case kernelModels.NeedExecuteSyscall:
		slog.Debug("PCP: CPU devolvió el proceso por syscall. Derivando...", "PID", pcb.PID)
		handleBlockingSyscall(result, pcb)

	default:
		slog.Warn("PCP: La CPU devolvió un código desconocido. Se moverá a READY por seguridad.", "PID", pcb.PID)
		TransitionProcessState(pcb, kernelModels.EstadoReady)
		StartShortTermScheduler()
	}

	cpu.PIDExecuting = 0
	kernelModels.ConnectedCpuMap.MarkAsFree(cpu.Id)
	slog.Debug("PCP: CPU liberada.", "cpu_id", cpu.Id)

	StartShortTermScheduler()
}

func sendProcessToExecute(pcb *kernelModels.PCB, cpu *models.CpuN) kernelModels.PCBExecuteRequest {
	slog.Info(fmt.Sprintf("## (%d) - Enviando a ejecutar a CPU %d", pcb.PID, cpu.Id))

	request := kernelModels.PCBExecuteRequest{
		PID: pcb.PID,
		PC:  pcb.PC,
	}

	body, err := json.Marshal(request)
	if err != nil {
		slog.Error("PCP: Error al serializar PCB para enviar a CPU.", "PID", pcb.PID, "error", err)
		return kernelModels.PCBExecuteRequest{StatusCodePCB: kernelModels.NeedReplan, PC: pcb.PC}
	}

	resp, err := client.DoRequest(cpu.Port, cpu.Ip, "POST", "cpu/exec", body)
	if err != nil {
		slog.Error("PCP: Error de comunicación con la CPU.", "cpu_id", cpu.Id, "error", err)
		return kernelModels.PCBExecuteRequest{StatusCodePCB: kernelModels.NeedReplan, PC: pcb.PC}
	}
	defer resp.Body.Close()

	var pcbResult kernelModels.PCBExecuteRequest
	if err := json.NewDecoder(resp.Body).Decode(&pcbResult); err != nil {
		slog.Error("PCP: Error al decodificar respuesta de la CPU.", "cpu_id", cpu.Id, "error", err)
		return kernelModels.PCBExecuteRequest{StatusCodePCB: kernelModels.NeedReplan, PC: pcb.PC}
	}

	return pcbResult
}
