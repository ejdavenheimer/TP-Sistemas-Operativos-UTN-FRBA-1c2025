package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

// getQueueByState es una función auxiliar interna para obtener la cola de un estado.
func getQueueByState(state models.Estado) *list.ArrayList[*models.PCB] {
	switch state {
	case models.EstadoNew:
		return models.QueueNew
	case models.EstadoReady:
		return models.QueueReady
	case models.EstadoExecuting:
		return models.QueueExec
	case models.EstadoBlocked:
		return models.QueueBlocked
	case models.EstadoSuspendidoReady:
		return models.QueueSuspReady
	case models.EstadoSuspendidoBlocked:
		return models.QueueSuspBlocked
	case models.EstadoExit:
		return models.QueueExit
	default:
		return nil
	}
}

// removeProcessFromCurrentQueue busca y elimina un PCB de cualquier cola.
func removeProcessFromCurrentQueue(pid uint) {
	queues := []*list.ArrayList[*models.PCB]{
		models.QueueNew,
		models.QueueReady,
		models.QueueExec,
		models.QueueBlocked,
		models.QueueSuspReady,
		models.QueueSuspBlocked,
		models.QueueExit,
	}

	for _, queue := range queues {
		queue.RemoveWhere(func(p *models.PCB) bool {
			return p.PID == pid
		})
	}
}

// TransitionProcessState se encarga de cambiar un proceso de estado.
// Mueve el PCB entre colas y actualiza sus atributos y métricas.
func TransitionProcessState(pcb *models.PCB, newState models.Estado) {
	pcb.Mutex.Lock()
	defer pcb.Mutex.Unlock()

	oldState := pcb.EstadoActual

	// --- INICIO DE LA MEJORA ---
	// Si el proceso está saliendo del estado BLOCKED y tiene un timer de suspensión activo,
	// lo detenemos para prevenir que se ejecute innecesariamente.
	if oldState == models.EstadoBlocked && pcb.SuspensionTimer != nil {
		if pcb.SuspensionTimer.Stop() {
			slog.Debug("Timer de suspensión detenido para proceso que sale de BLOCKED.", "PID", pcb.PID)
		}
		pcb.SuspensionTimer = nil // Limpiamos la referencia
	}
	// --- FIN DE LA MEJORA ---

	if oldState != "" {
		removeProcessFromCurrentQueue(pcb.PID)
		if !pcb.UltimoCambio.IsZero() {
			duration := time.Since(pcb.UltimoCambio)
			pcb.MT[oldState] += duration
		}
	}

	pcb.EstadoActual = newState
	pcb.UltimoCambio = time.Now()
	pcb.ME[newState]++

	targetQueue := getQueueByState(newState)
	if targetQueue != nil {
		targetQueue.Add(pcb)
	} else {
		slog.Error(fmt.Sprintf("## (%d) - Intento de mover a un estado con cola no definida: %s", pcb.PID, newState))
		return
	}

	if oldState == "" {
		slog.Info(fmt.Sprintf("## (%d) Se crea el proceso - Estado : NEW", pcb.PID))
	} else {
		slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado %s", pcb.PID, oldState, newState))
	}

	// Si el proceso está entrando al estado BLOCKED, iniciamos el timer de suspensión.
	if newState == models.EstadoBlocked {
		go StartSuspensionTimer(pcb)
	}

	// Lógica de desalojo para SRT.
	if newState == models.EstadoReady && models.KernelConfig.SchedulerAlgorithm == "SRT" {
		go checkForPreemption(pcb)
	}
}
