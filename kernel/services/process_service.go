package services

import (
	"fmt"
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

func GetProcess(pid int) models.ProcessResponse {
	pcb, _, exists := FindPCBInAnyQueue(pid)

	var processResponse models.ProcessResponse

	if !exists {
		slog.Error(fmt.Sprintf("No se encontro el  proceso <%d>", pid))
		return processResponse
	}

	processResponse.Pid = pcb.PID
	processResponse.EstadoActual = pcb.EstadoActual

	return processResponse
}

// FindPCBInAnyQueue busca un PCB con el PID dado en cualquiera de las colas de planificación.
// Retorna el PCB encontrado, un puntero a la cola donde se encontró y un booleano indicando si fue encontrado.
// Si el PCB no se encuentra, retorna un PCB vacío, nil y false.
func FindPCBInAnyQueue(pid int) (models.PCB, *list.ArrayList[models.PCB], bool) {
	queuesToSearch := []*list.ArrayList[models.PCB]{
		models.QueueNew,
		models.QueueReady,
		models.QueueExec,
		models.QueueBlocked,
		models.QueueSuspReady,
		models.QueueSuspBlocked,
		models.QueueExit,
	}

	for _, queue := range queuesToSearch {
		pcb, _, found := queue.Find(func(p models.PCB) bool {
			return p.PID == pid
		})
		if found {
			slog.Debug("Kernel: PCB encontrado en cola", slog.Int("pid", pid), slog.Any("queue", queue))
			return pcb, queue, true
		}
	}

	slog.Debug("Kernel: PCB no encontrado en ninguna cola", slog.Int("pid", pid))
	var zeroPCB models.PCB
	return zeroPCB, nil, false
}

// GetQueuesState recolecta el estado actual de todas las colas de planificación.
// Retorna un mapa donde la clave es el nombre de la cola y el valor es una lista de PCBInfo.
func GetQueuesState() map[string][]models.ProcessResponse {
	queuesState := make(map[string][]models.ProcessResponse)

	// Definimos las colas con sus nombres para facilitar la iteración y el mapeo.
	// El orden aquí puede reflejar el orden lógico de tu sistema.
	queues := []struct {
		Name  string
		Queue *list.ArrayList[models.PCB]
	}{
		{"QueueNew", models.QueueNew},
		{"QueueReady", models.QueueReady},
		{"QueueExec", models.QueueExec},
		{"QueueBlocked", models.QueueBlocked},
		{"QueueSuspReady", models.QueueSuspReady},
		{"QueueSuspBlocked", models.QueueSuspBlocked},
		{"QueueExit", models.QueueExit},
	}

	for _, q := range queues {
		pcbsInQueue := make([]models.ProcessResponse, 0)
		allPCBs := q.Queue.GetAll() // Asumo que GetAll() es un método seguro y retorna una copia del slice interno.
		if allPCBs == nil {
			slog.Warn("Kernel: GetAll() de cola retornó nil. Puede indicar un problema en ArrayList.", slog.String("queue", q.Name))
			continue
		}

		for _, pcb := range allPCBs {
			pcbsInQueue = append(pcbsInQueue, models.ProcessResponse{
				Pid:          pcb.PID,
				EstadoActual: pcb.EstadoActual,
			})
		}
		queuesState[q.Name] = pcbsInQueue
		slog.Debug(fmt.Sprintf("Kernel: Recolectada información de la cola %s. PCBs: %d", q.Name, len(pcbsInQueue)))
	}

	return queuesState
}

// AddProcessToQueue añade un PCB a la cola correspondiente según el estado especificado en ProcessRequest.
// Retorna true si se añadió con éxito, false en caso contrario (ej. estado inválido).
func AddProcessToQueue(pid int, estado models.Estado) (bool, error) {
	// Crea un nuevo PCB a partir de la solicitud.
	// Puedes inicializar otros campos del PCB aquí si son obligatorios.
	newPCB := models.PCB{
		PID:          pid,
		EstadoActual: estado,
	}

	var targetQueue *list.ArrayList[models.PCB]
	var queueName string

	// Determina la cola de destino basándose en el EstadoActual
	switch estado {
	case models.EstadoNew:
		targetQueue = models.QueueNew
		queueName = "QueueNew"
	case models.EstadoReady:
		targetQueue = models.QueueReady
		queueName = "QueueReady"
	case models.EstadoExecuting:
		targetQueue = models.QueueExec
		queueName = "QueueExec"
	case models.EstadoBlocked:
		targetQueue = models.QueueBlocked
		queueName = "QueueBlocked"
	case models.EstadoSuspendidoReady:
		targetQueue = models.QueueSuspReady
		queueName = "QueueSuspReady"
	case models.EstadoSuspendidoBlocked:
		targetQueue = models.QueueSuspBlocked
		queueName = "QueueSuspBlocked"
	case models.EstadoExit:
		targetQueue = models.QueueExit
		queueName = "QueueExit"
	default:
		slog.Error("Kernel: Estado de PCB inválido especificado para añadir a cola",
			slog.Int("pid", pid), slog.String("estado", string(estado)))
		return false, fmt.Errorf("estado de PCB inválido: %s", estado)
	}

	if existingPCB, _, found := FindPCBInAnyQueue(pid); found {
		slog.Warn("Kernel: Intento de añadir PCB que ya existe en el sistema.",
			slog.Int("pid", pid), slog.String("estado_existente", string(existingPCB.EstadoActual)),
			slog.String("estado_solicitado", string(estado)))
		return false, fmt.Errorf("el PCB con PID %d ya existe en el estado %s", pid, existingPCB.EstadoActual)
	}

	// Añadir el PCB a la cola de destino
	targetQueue.Add(newPCB)
	slog.Debug(fmt.Sprintf("Kernel: PCB %d añadido a la cola %s.", pid, queueName),
		slog.Int("pid", pid), slog.String("estado_destino", string(estado)))

	return true, nil
}

// MoveProcessToState busca un PCB por su PID, lo remueve de su cola actual
// y lo añade a la cola correspondiente a su NewEstado.
// Retorna el PCB actualizado, un booleano indicando si la operación fue exitosa,
// y un error si ocurre algún problema.
func MoveProcessToState(pid int, nuevoEstado models.Estado) (models.PCB, bool, error) {
	// 1. Encontrar el PCB en cualquiera de las colas
	pcbToMove, currentQueue, found := FindPCBInAnyQueue(pid)
	if !found {
		slog.Warn("Kernel: Intento de mover PCB no encontrado",
			slog.Int("pid", pid), slog.String("new_estado", string(nuevoEstado)))
		return models.PCB{}, false, fmt.Errorf("proceso con PID %d no encontrado en ninguna cola", pid)
	}

	// 2. Validar que el nuevo estado sea diferente al actual (opcional, pero útil)
	if pcbToMove.EstadoActual == nuevoEstado {
		slog.Warn("Kernel: Intento de mover PCB a su mismo estado actual",
			slog.Int("pid", pid), slog.String("estado", string(nuevoEstado)))
		return pcbToMove, true, fmt.Errorf("el proceso %d ya se encuentra en el estado %s", pid, nuevoEstado)
	}

	slog.Info(fmt.Sprintf("Kernel: Moviendo PCB %d de %s a %s.",
		pid, pcbToMove.EstadoActual, nuevoEstado))

	// 3. Remover el PCB de su cola actual
	// Asumo que tu ArrayList.Remove() también es concurrente-seguro.
	removedPCB, index, _ := currentQueue.Find(func(p models.PCB) bool {
		return p.PID == pid
	})
	if index == -1 {
		// Esto no debería pasar si FindPCBInAnyQueue lo encontró y el mutex está bien
		slog.Error("Kernel: Fallo al remover PCB de su cola actual después de encontrarlo",
			slog.Int("pid", pid)) // Asumiendo que Queue tiene Name
		return models.PCB{}, false, fmt.Errorf("error interno: no se pudo remover PCB %d de la cola", pid)
	}

	currentQueue.Remove(index)

	// 4. Actualizar el estado del PCB
	removedPCB.EstadoActual = nuevoEstado
	// Puedes actualizar otros campos aquí si la transición lo requiere
	// Por ejemplo, si pasa a READY, quizás resetear algún contador de CPU.

	// 5. Determinar la cola de destino y añadir el PCB
	var targetQueue *list.ArrayList[models.PCB]
	var targetQueueName string

	switch nuevoEstado {
	case models.EstadoNew:
		targetQueue = models.QueueNew
		targetQueueName = "QueueNew"
	case models.EstadoReady:
		targetQueue = models.QueueReady
		targetQueueName = "QueueReady"
	case models.EstadoExecuting:
		// ¡Cuidado aquí! Si ya hay un proceso en EXEC, esto puede ser un error.
		// Tu planificador debería manejar la transición a EXEC de forma exclusiva.
		// Por simplicidad, lo añadimos, pero ten en cuenta la lógica de tu simulador.
		if models.QueueExec.Size() > 0 {
			slog.Warn("Kernel: Ya hay un proceso en QueueExec. Añadiendo PCB %d de todas formas.", slog.Int("pid", pid))
			// Podrías añadir lógica aquí para "desalojar" al PCB actual de EXEC.
		}
		targetQueue = models.QueueExec
		targetQueueName = "QueueExec"
	case models.EstadoBlocked:
		targetQueue = models.QueueBlocked
		targetQueueName = "QueueBlocked"
	case models.EstadoSuspendidoReady:
		targetQueue = models.QueueSuspReady
		targetQueueName = "QueueSuspReady"
	case models.EstadoSuspendidoBlocked:
		targetQueue = models.QueueSuspBlocked
		targetQueueName = "QueueSuspBlocked"
	case models.EstadoExit:
		targetQueue = models.QueueExit
		targetQueueName = "QueueExit"
	default:
		// Si el nuevo estado es inválido, deberíamos devolver el PCB a su cola original
		// o manejarlo como un error irrecuperable.
		// Por ahora, lo dejamos fuera y retornamos un error.
		currentQueue.Add(removedPCB) // Intenta devolverlo
		slog.Error("Kernel: Intento de mover PCB a un estado desconocido",
			slog.Int("pid", pid), slog.String("new_estado", string(nuevoEstado)))
		return models.PCB{}, false, fmt.Errorf("estado de destino inválido: %s", nuevoEstado)
	}

	targetQueue.Add(removedPCB)
	slog.Info(fmt.Sprintf("Kernel: PCB %d movido exitosamente a %s.",
		pid, targetQueueName),
		slog.Int("pid", pid),
		slog.String("destino", targetQueueName))

	return removedPCB, true, nil
}
