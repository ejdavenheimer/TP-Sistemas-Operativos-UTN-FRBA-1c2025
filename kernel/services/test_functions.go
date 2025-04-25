package services

import (
	"fmt"
	"time"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

func TestQueueNew() {
	// Crear un PCB
	pcb1 := models.PCB{
		PID:          1,
		PC:           0,
		EstadoActual: "Nuevo",
		UltimoCambio: time.Now(),
	}

	pcb2 := models.PCB{
		PID:          2,
		PC:           0,
		EstadoActual: "Ready",
		UltimoCambio: time.Now(),
	}

	// Agregar el PCB a la cola
	models.QueueNew.Add(pcb1)
	models.QueueNew.Add(pcb2)

	// Obtener el primer elemento
	primero, err := models.QueueNew.Get(0)
	if err != nil {
		fmt.Println("Error al obtener el primer PCB:", err)
		return
	}

	fmt.Printf("Primer PCB: PID=%d, Estado=%s\n", primero.PID, primero.EstadoActual)

	models.QueueNew.Dequeue()
	primero, err = models.QueueNew.Get(0)
	if err != nil {
		fmt.Println("Error al obtener el primer PCB:", err)
		return
	}

	fmt.Printf("Primer PCB, después de eliminar el anterior: PID=%d, Estado=%s\n", primero.PID, primero.EstadoActual)

}
