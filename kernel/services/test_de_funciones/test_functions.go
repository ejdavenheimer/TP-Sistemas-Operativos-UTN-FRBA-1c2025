package services

import (
	"fmt"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/config"
)

func TestQueueNew() {
	// Crear un PCB
	pcb1 := models.PCB{
		PID:          1,
		PC:           0,
		EstadoActual: "Nuevo",
	}

	pcb2 := models.PCB{
		PID:          2,
		PC:           0,
		EstadoActual: "Ready",
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

const (
	ConfigPath = "configs/kernel.json"
)

func TestFinalizarProceso() {
	config.InitConfig(ConfigPath, &models.KernelConfig)

	// Crear un PCB
	pcb1 := models.PCB{
		PID:          1,
		PC:           0,
		EstadoActual: "Nuevo",
	}

	pcb2 := models.PCB{
		PID:          2,
		PC:           0,
		EstadoActual: "Ready",
	}

	// Agregar el PCB a la cola
	models.QueueExit.Add(pcb1)
	models.QueueExit.Add(pcb2)

	services.FinishProcess(pcb1)

	primero, err := models.QueueExit.Get(0)
	if err != nil {
		fmt.Println("Error al obtener el primer PCB:", err)
		return
	}
	fmt.Printf("Primer PCB: PID=%d, Estado=%s\n", primero.PID, primero.EstadoActual)

}
