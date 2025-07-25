package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// StartSuspensionTimer se inicia como una goroutine cuando un proceso entra a BLOCKED.
func StartSuspensionTimer(pcb *models.PCB) {
	suspensionTime := time.Duration(models.KernelConfig.SuspensionTime) * time.Millisecond
	slog.Debug("Timer de suspensión iniciado para proceso en BLOCKED.", "PID", pcb.PID, "Tiempo(ms)", suspensionTime)
	time.Sleep(suspensionTime)

	// Es crucial volver a bloquear el PCB para leer su estado de forma segura.
	pcb.Mutex.Lock()
	defer pcb.Mutex.Unlock()

	// Si el proceso ya no está en BLOCKED (porque terminó su I/O), cancelamos la suspensión.
	if pcb.EstadoActual != models.EstadoBlocked {
		slog.Debug("Proceso ya no está bloqueado. Suspensión cancelada.", "PID", pcb.PID, "EstadoActual", pcb.EstadoActual)
		return
	}

	// Si sigue bloqueado, lo movemos a SUSPENDED_BLOCKED.
	slog.Info(fmt.Sprintf("## (%d) - Proceso supera tiempo máximo en BLOCKED. Pasa a SUSPEND_BLOCKED.", pcb.PID))

	// Como estamos dentro de un Lock del PCB, hacemos la transición manualmente para evitar deadlocks.
	// 1. Cambiamos los atributos del PCB.
	pcb.EstadoActual = models.EstadoSuspendidoBlocked
	pcb.UltimoCambio = time.Now()
	pcb.ME[models.EstadoSuspendidoBlocked]++

	// 2. Lo sacamos de la cola BLOCKED y lo agregamos a la nueva cola.
	models.QueueBlocked.RemoveWhere(func(p *models.PCB) bool { return p.PID == pcb.PID })
	models.QueueSuspBlocked.Add(pcb)

	// Notificamos al PMP que tiene un proceso para mover a SWAP.
	StartMediumTermScheduler()
}
