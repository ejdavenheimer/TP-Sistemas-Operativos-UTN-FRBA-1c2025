package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

type KernelSwapController struct {
	mutex            sync.RWMutex
	activeOperations map[uint]string // PID -> tipo de operación ("swap_in" o "swap_out")
	pmpMutex         sync.Mutex      // Mutex para el PMP
}

var kernelSwapController = &KernelSwapController{
	activeOperations: make(map[uint]string),
}

func canProcessSwap(pid uint, operation string) bool {
	kernelSwapController.mutex.RLock()
	defer kernelSwapController.mutex.RUnlock()

	currentOp, exists := kernelSwapController.activeOperations[pid]
	if exists {
		slog.Debug("Operación ya activa para proceso", "PID", pid, "current_operation", currentOp, "requested_operation", operation)
		return false
	}
	return true
}

func registerKernelSwapOperation(pid uint, operation string) bool {
	kernelSwapController.mutex.Lock()
	defer kernelSwapController.mutex.Unlock()

	if _, exists := kernelSwapController.activeOperations[pid]; exists {
		return false
	}

	kernelSwapController.activeOperations[pid] = operation
	slog.Debug("Operación de kernel registrada", "PID", pid, "operation", operation)
	return true
}

func unregisterKernelSwapOperation(pid uint) {
	kernelSwapController.mutex.Lock()
	defer kernelSwapController.mutex.Unlock()

	operation, exists := kernelSwapController.activeOperations[pid]
	if exists {
		delete(kernelSwapController.activeOperations, pid)
		slog.Debug("Operación de kernel completada", "PID", pid, "operation", operation)
	}
}

// --- Funciones del Planificador ---

func StartMediumTermScheduler() {
	select {
	case models.NotifyMediumScheduler <- 1:
	default:
	}
}

// MediumTermScheduler es el ciclo principal del PMP. Reacciona a notificaciones.
func MediumTermScheduler() {
	for {
		<-models.NotifyMediumScheduler
		slog.Debug("PMP: Planificador de Mediano Plazo activado.")

		kernelSwapController.pmpMutex.Lock()
		if models.QueueSuspReady.Size() > 0 {
			handleSuspendedReady()
		}

		if models.QueueSuspBlocked.Size() > 0 {
			handleSuspendedBlocked()
		}
		kernelSwapController.pmpMutex.Unlock()
	}
}

// --- Lógica de Suspensión (SWAP-IN) ---

func handleSuspendedBlocked() {
	var pcbToSwap *models.PCB = nil

	// Buscamos un proceso que necesite ser swapeado sin quitarlo de la cola.
	allSuspended := models.QueueSuspBlocked.GetAll()

	for _, pcb := range allSuspended {
		pcb.Mutex.Lock()
		if !pcb.SwapRequested && canProcessSwap(pcb.PID, "swap_in") {
			_, _, stillInQueue := models.QueueSuspBlocked.Find(func(p *models.PCB) bool {
				return p.PID == pcb.PID
			})
			if stillInQueue {
				pcb.SwapRequested = true // Lo marcamos para que no se vuelva a procesar
				pcbToSwap = pcb
				pcb.Mutex.Unlock()
				break
			}
		}
		pcb.Mutex.Unlock()
	}

	if pcbToSwap == nil {
		slog.Debug("PMP: No hay nuevos procesos en SUSPEND_BLOCKED para solicitar swap.")
		return
	}

	if err := requestSwapIn(pcbToSwap); err != nil {
		slog.Error("PMP: Error en swap in", "PID", pcbToSwap.PID, "error", err)
		// Limpiar el flag en caso de error
		pcbToSwap.Mutex.Lock()
		pcbToSwap.SwapRequested = false
		pcbToSwap.Mutex.Unlock()
	}
}

func requestSwapIn(pcb *models.PCB) error {
	// Registrar operación para evitar concurrencia
	if !registerKernelSwapOperation(pcb.PID, "swap_in") {
		return fmt.Errorf("operación ya activa para PID %d", pcb.PID)
	}
	defer unregisterKernelSwapOperation(pcb.PID)

	slog.Debug("PMP: Solicitando a Memoria mover proceso a SWAP.", "PID", pcb.PID)

	req := struct {
		PID uint `json:"pid"`
	}{PID: pcb.PID}
	body, _ := json.Marshal(req)

	// Hacer la petición HTTP de forma síncrona
	response, err := client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/putSwap", body)

	if err != nil {
		slog.Error("PMP: Error al solicitar SWAP IN a Memoria.", "PID", pcb.PID, "error", err)
		return err
	}

	// Verificar el código de respuesta
	if response == nil {
		slog.Error("PMP: Respuesta nula de Memoria para SWAP IN.", "PID", pcb.PID)
		return fmt.Errorf("respuesta nula de memoria")
	}

	slog.Debug("PMP: Memoria confirmó SWAP IN exitosamente.", "PID", pcb.PID)
	StartLongTermScheduler()
	return nil
}

// --- Lógica de Desuspensión (SWAP-OUT) ---

func handleSuspendedReady() {
	switch models.KernelConfig.NewAlgorithm {
	case "FIFO":
		if models.QueueSuspReady.Size() > 0 {
			pcb, err := models.QueueSuspReady.Get(0)
			if err != nil {
				slog.Error("Error obteniendo proceso de cola SUSP_READY", "error", err)
				return
			}
			desuspendProcess(pcb)
		}
	case "PMCP":
		//allSuspended := models.QueueSuspReady.GetAll()
		//if len(allSuspended) == 0 {
		//	return
		//}
		//sort.Slice(allSuspended, func(i, j int) bool {
		//	return allSuspended[i].Size < allSuspended[j].Size
		//})
		//pcb := allSuspended[0]
		//for _, pcb := range allSuspended {
		//	desuspendProcess(pcb)
		//}
		//_, _, found := models.QueueSuspReady.Find(func(p *models.PCB) bool { return p.PID == pcb.PID })
		//if found {
		//	models.QueueSuspReady.(pcb)
		//	desuspendProcess(pcb)
		//}
		scheduleReadyPMCP()
	default:
		if models.QueueSuspReady.Size() > 0 {
			pcb, err := models.QueueSuspReady.Get(0)
			if err != nil {
				slog.Error("Error obteniendo proceso de cola SUSP_READY", "error", err)
				return
			}
			desuspendProcess(pcb)
		}
	}
}

func scheduleReadyPMCP() {
	if models.QueueSuspReady.Size() == 0 {
		return
	}

	allSuspended := models.QueueSuspReady.GetAll()
	sort.Slice(allSuspended, func(i, j int) bool {
		return allSuspended[i].Size < allSuspended[j].Size
	})

	pcb := allSuspended[0]
	desuspendProcess(pcb)
}

func desuspendProcess(pcb *models.PCB) {
	if !canProcessSwap(pcb.PID, "swap_out") {
		slog.Debug("PMP: Proceso ya siendo procesado para swap out", "PID", pcb.PID)
	}
	slog.Debug("Desuspendiendo proceso para pasar a ready")
	_, _, found := models.QueueSuspReady.Find(func(p *models.PCB) bool { return p.PID == pcb.PID })
	if !found {
		return
	}

	if !isProcessInSwap(pcb.PID) {
		slog.Debug("PMP: Proceso no está en swap, transicionando directamente a READY", "PID", pcb.PID)

		// Verificar si hay capacidad de memoria (por las dudas)
		if err := CheckUserMemoryCapacity(pcb.PID, pcb.Size); err != nil {
			slog.Debug("PMP: No hay memoria suficiente para proceso no swapeado. Permanece en SUSP_READY.", "PID", pcb.PID)
			return
		}

		// Remover de la cola SUSP_READY
		_, index, found := models.QueueSuspReady.Find(func(p *models.PCB) bool {
			return p.PID == pcb.PID
		})
		if found {
			models.QueueSuspReady.Remove(index)
			slog.Debug("PMP: Proceso removido de SUSP_READY", "PID", pcb.PID, "index", index)
		} else {
			slog.Warn("PMP: Proceso no encontrado en SUSP_READY para remover", "PID", pcb.PID)
			return
		}

		// Limpiar flag de swap
		pcb.Mutex.Lock()
		pcb.SwapRequested = false
		pcb.Mutex.Unlock()

		// Pasar directamente a READY
		slog.Debug(fmt.Sprintf("## (%d) - Pasa de SUSPENDED_READY a READY (no estaba en swap)", pcb.PID))
		TransitionProcessState(pcb, models.EstadoReady)
		StartShortTermScheduler()
		return
	}

	if err := CheckUserMemoryCapacity(pcb.PID, pcb.Size); err != nil {
		slog.Debug("PMP: No hay memoria para desuspender proceso. Permanece en SUSP_READY.", "PID", pcb.PID)
		return
	}

	requestSwapOut(pcb)
}

func requestSwapOut(pcb *models.PCB) {
	if !registerKernelSwapOperation(pcb.PID, "swap_out") {
		return
	}
	defer unregisterKernelSwapOperation(pcb.PID)
	slog.Debug("PMP: Solicitando a Memoria SWAP OUT.", "PID", pcb.PID)
	req := struct {
		PID uint `json:"pid"`
	}{PID: pcb.PID}
	body, _ := json.Marshal(req)

	response, err := client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/removeSwap", body)

	if err != nil {
		slog.Error("PMP: Error al solicitar SWAP OUT a Memoria. Finalizando proceso.", "PID", pcb.PID, "error", err)
		TransitionProcessState(pcb, models.EstadoExit)
		StartLongTermScheduler()
		return
	}

	if response == nil {
		slog.Error("PMP: Respuesta nula de Memoria para SWAP OUT.", "PID", pcb.PID)
		TransitionProcessState(pcb, models.EstadoExit)
		StartLongTermScheduler()
		return
	}

	// Reiniciamos el flag para que pueda volver a ser suspendido en el futuro.
	pcb.Mutex.Lock()
	pcb.SwapRequested = false
	pcb.Mutex.Unlock()

	slog.Debug(fmt.Sprintf("## (%d) - Pasa de SUSPENDED_READY a READY", pcb.PID))
	TransitionProcessState(pcb, models.EstadoReady)
	StartShortTermScheduler()
}

func isProcessInSwap(pid uint) bool {
	req := struct {
		PID uint `json:"pid"`
	}{PID: pid}

	body, err := json.Marshal(req)
	if err != nil {
		slog.Error("Error serializando request para check swap", "error", err)
		return false
	}

	response, err := client.DoRequest(
		models.KernelConfig.PortMemory,
		models.KernelConfig.IpMemory,
		"POST",
		"memoria/checkSwap",
		body,
	)

	if err != nil {
		slog.Error("Error consultando estado de swap", "PID", pid, "error", err)
		return false
	}
	if response == nil {
		slog.Error("Respuesta nula al consultar estado de swap", "PID", pid)
		return false
	}

	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		slog.Error("Error leyendo body de respuesta de check swap", "error", err)
		return false
	}

	var swapStatus struct {
		PID     uint `json:"pid"`
		InSwap  bool `json:"in_swap"`
		Success bool `json:"success"`
	}

	if err := json.Unmarshal(responseBody, &swapStatus); err != nil {
		slog.Error("Error deserializando respuesta de check swap", "error", err)
		return false
	}

	if !swapStatus.Success {
		slog.Error("Memoria reportó error al consultar swap", "PID", pid)
		return false
	}

	slog.Debug("Estado de swap consultado", "PID", pid, "in_swap", swapStatus.InSwap)
	return swapStatus.InSwap
}
