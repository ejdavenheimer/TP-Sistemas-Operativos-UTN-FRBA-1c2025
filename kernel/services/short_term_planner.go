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
	slog.Debug(fmt.Sprintf("CANTIDAD DE PROCESOS EN LA COLA READY %v", kernelModels.QueueReady.Size()))
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

	// --- LÓGICA ANTI-INANICIÓN (LA TUYA, QUE ES LA CORRECTA) ---
	shortestEstimate := float32(-1)
	var firstNewPcb *kernelModels.PCB
	initialEstimate := float32(kernelModels.KernelConfig.InitialEstimate)

	for _, pcb := range allReadyProcesses {
		if pcb.RafagaEstimada != initialEstimate {
			if shortestEstimate == -1 || pcb.RafagaEstimada < shortestEstimate {
				shortestEstimate = pcb.RafagaEstimada
			}
		} else if firstNewPcb == nil {
			firstNewPcb = pcb
		}
	}

	var pcbToExecute *kernelModels.PCB

	if firstNewPcb != nil && (shortestEstimate == -1 || initialEstimate > shortestEstimate) {
		pcbToExecute = firstNewPcb
		slog.Debug("PCP (SJF): Priorizando proceso nuevo para evitar inanición.", "PID", pcbToExecute.PID)
	} else {
		// Se aplica la lógica SJF estándar sobre toda la lista
		shortestIndex := 0
		finalShortestEstimate := allReadyProcesses[0].RafagaEstimada
		for i := 1; i < len(allReadyProcesses); i++ {
			if allReadyProcesses[i].RafagaEstimada < finalShortestEstimate {
				finalShortestEstimate = allReadyProcesses[i].RafagaEstimada
				shortestIndex = i
			}
		}
		pcbToExecute = allReadyProcesses[shortestIndex]
		slog.Debug("PCP (SJF/SRT): Proceso seleccionado por ráfaga más corta.", "PID", pcbToExecute.PID, "Estimación", pcbToExecute.RafagaEstimada)
	}

	// --- MEJORA DE SEGURIDAD ---
	// Se reemplaza la eliminación por índice por una eliminación segura por PID.
	kernelModels.QueueReady.RemoveWhere(func(p *kernelModels.PCB) bool {
		return p.PID == pcbToExecute.PID
	})
	// -------------------------

	return pcbToExecute, nil
}

func handleCpuExecution(pcb *kernelModels.PCB, cpu *models.CpuN) {
	cpu.PIDExecuting = pcb.PID
	kernelModels.ConnectedCpuMap.Set(strconv.Itoa(cpu.Id), cpu)

	pcb.BurstStartTime = time.Now()

	TransitionProcessState(pcb, kernelModels.EstadoExecuting)
	result := sendProcessToExecute(pcb, cpu)

	// La CPU se marca como libre inmediatamente después de recibir la respuesta,
	// permitiendo que el planificador la asigne a otro proceso mientras
	// el Kernel gestiona el resultado del proceso actual.
	cpu.PIDExecuting = 0
	kernelModels.ConnectedCpuMap.MarkAsFree(cpu.Id)
	slog.Debug("PCP: CPU liberada.", "cpu_id", cpu.Id)
	// --------------------------

	// La lógica para actualizar la ráfaga estimada ahora depende de si el proceso fue interrumpido o no.
	if result.StatusCodePCB == kernelModels.NeedInterrupt {
		// El proceso fue desalojado. No se usa la fórmula de estimación.
		// Se actualiza la estimación restando el tiempo que ya se ejecutó.
		elapsedTime := result.ExecutionTime
		pcb.RafagaEstimada = pcb.RafagaEstimada - elapsedTime
		if pcb.RafagaEstimada < 0 { // Medida de seguridad para evitar estimaciones negativas.
			pcb.RafagaEstimada = 0
		}
		// La ráfaga "real" no se actualiza porque no se completó.
		pcb.RafagaReal = 0
		slog.Debug("SRT: Proceso desalojado. Nueva estimación restante calculada.", "PID", pcb.PID, "Nueva Estimación", pcb.RafagaEstimada)

	} else {
		// El proceso terminó su ráfaga de forma natural (por I/O, EXIT, etc.).
		// Aquí sí se aplica la fórmula de estimación estándar.
		pcb.RafagaReal = result.ExecutionTime
		alpha := kernelModels.KernelConfig.Alpha
		pcb.RafagaEstimada = (alpha * pcb.RafagaReal) + ((1 - alpha) * pcb.RafagaEstimada)
	}

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
