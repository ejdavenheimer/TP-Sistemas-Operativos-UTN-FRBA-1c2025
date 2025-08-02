package services

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// suspendProcessLogic contiene la lógica para mover un proceso a SUSPEND_BLOCKED.
// Se asume que el Mutex del PCB ya fue adquirido antes de llamar a esta función.
func suspendProcessLogic(pcb *models.PCB) {
	slog.Debug(fmt.Sprintf("## (%d) - Proceso supera tiempo máximo en BLOCKED. Pasa a SUSPEND_BLOCKED.", pcb.PID))

	// 1. Calculamos el tiempo que estuvo en BLOCKED y actualizamos la métrica.
	oldState := pcb.EstadoActual
	if !pcb.UltimoCambio.IsZero() {
		duration := time.Since(pcb.UltimoCambio)
		pcb.MT[oldState] += duration
	}

	// 2. Cambiamos los atributos del PCB al nuevo estado.
	pcb.EstadoActual = models.EstadoSuspendidoBlocked
	pcb.UltimoCambio = time.Now()
	pcb.ME[models.EstadoSuspendidoBlocked]++

	// 3. Lo sacamos de la cola BLOCKED y lo agregamos a la nueva cola.
	models.QueueBlocked.RemoveWhere(func(p *models.PCB) bool { return p.PID == pcb.PID })
	models.QueueSuspBlocked.Add(pcb)

	// Notificamos al PMP que tiene un proceso para mover a SWAP.
	StartMediumTermScheduler()
}

// StartSuspensionTimer inicia un temporizador cancelable cuando un proceso entra a BLOCKED.
func StartSuspensionTimer(pcb *models.PCB) {
	suspensionTime := time.Duration(models.KernelConfig.SuspensionTime) * time.Millisecond
	slog.Debug("Timer de suspensión iniciado para proceso en BLOCKED.", "PID", pcb.PID, "Tiempo(ms)", suspensionTime)

	// Creamos un timer que ejecutará la lógica de suspensión después del tiempo especificado.
	timer := time.AfterFunc(suspensionTime, func() {
		pcb.Mutex.Lock()
		defer pcb.Mutex.Unlock()

		// Verificamos si el proceso AÚN está en BLOCKED cuando el timer se dispara.
		if pcb.EstadoActual == models.EstadoBlocked {
			suspendProcessLogic(pcb)
		}
		// Si ya no está en BLOCKED, no hacemos nada, el timer ya fue detenido.
	})

	// Guardamos la referencia al timer en el PCB para poder cancelarlo si sale de BLOCKED antes de tiempo.
	pcb.SuspensionTimer = timer
}
