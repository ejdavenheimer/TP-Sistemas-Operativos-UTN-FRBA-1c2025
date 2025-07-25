package services

import (
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

// UnblockProcessAfterIO es la nueva función para desbloquear un proceso después de una I/O.
// Revisa si el proceso estaba en BLOCKED o en SUSPENDED_BLOCKED y actúa en consecuencia.
func UnblockProcessAfterIO(pid uint) {
	// Intentamos encontrarlo en la cola de bloqueados (en memoria).
	pcb, _, foundInBlocked := models.QueueBlocked.Find(func(p *models.PCB) bool { return p.PID == pid })
	if foundInBlocked {
		slog.Debug("Desbloqueando proceso de BLOCKED a READY.", "PID", pid)
		TransitionProcessState(pcb, models.EstadoReady)
		StartShortTermScheduler()
		return
	}

	// Si no estaba ahí, lo buscamos en la cola de suspendidos-bloqueados (en SWAP).
	pcb, _, foundInSuspBlocked := models.QueueSuspBlocked.Find(func(p *models.PCB) bool { return p.PID == pid })
	if foundInSuspBlocked {
		slog.Debug("Proceso fin de I/O en SWAP. Moviendo de SUSPENDED_BLOCKED a SUSPENDED_READY.", "PID", pid)

		// CORRECCIÓN: Reiniciamos el flag para que pueda ser swapeado de nuevo si es necesario en el futuro.
		pcb.Mutex.Lock()
		pcb.SwapRequested = false
		pcb.Mutex.Unlock()

		TransitionProcessState(pcb, models.EstadoSuspendidoReady)
		StartMediumTermScheduler() // Notificamos al PMP que tiene un proceso para evaluar SWAP-IN.
		return
	}

	slog.Warn("Se intentó desbloquear un PID por fin de I/O, pero no fue encontrado en ninguna cola de bloqueo.", "PID", pid)
}

// FindPCBInAnyQueue busca un PCB por su PID en cualquiera de las colas de planificación.
func FindPCBInAnyQueue(pid uint) (*models.PCB, bool) {
	// CORREGIDO: La variable 'queues' ahora es un slice del tipo correcto.
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
		pcb, _, found := queue.Find(func(p *models.PCB) bool {
			return p.PID == pid
		})
		if found {
			return pcb, true
		}
	}
	return nil, false
}
