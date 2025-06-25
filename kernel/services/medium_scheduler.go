package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

func MediumTermScheduler() {
	slog.Info("Planificador de mediano plazo iniciado.")
	/////////////////////////////////////////////////////
	//BORRAR DESPUÉS ESTE PCB, es solo de prueba
	//var pcb = models.PCB{
	//	PID:            1,
	//	Size:           5,
	//	PseudocodePath: "./scripts/prueba2",
	//}
	/////////////////////////////////////////////////////
	//models.QueueSuspReady.Add(pcb)
	for {
		//Si ambas colas están vacías, vuelve a mirar en otro momento

		if models.QueueSuspReady.Size() == 0 && models.QueueSuspBlocked.Size() == 0 {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		//Maneja el swapeo de procesos en la cola SuspBlocked
		if models.QueueSuspBlocked.Size() != 0 {
			movePrincipalMemoryToSwap()
		}

		if models.QueueSuspReady.Size() != 0 {
			switch models.KernelConfig.NewAlgorithm {
			case "FIFO":
				slog.Debug("Me meto al plani de mediano plazo FIFO")
				mediumScheduleFIFO()
			case "PMCP":
				slog.Debug("Me meto al plani de mediano plazo PMCP")
				mediumScheduleShortestFirst()
			default:
				slog.Warn("Algoritmo no reconocido, utilizando FIFO por defecto")
				mediumScheduleFIFO()
			}
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// SUSP. BLOCKED. En este momento se debe informar al módulo memoria que debe ser
// movido de memoria principal a swap. Cabe aclarar que en este momento vamos
// a tener más memoria libre en el sistema por lo que se debe verificar si uno o
// más nuevos procesos pueden entrar (tanto de la cola NEW como de SUSP. READY).
func movePrincipalMemoryToSwap() {
	//Armar estructura a enviar
	var pcb, _ = models.QueueSuspBlocked.Get(0)

	slog.Info("Iniciando solicitud para mover el proceso de memoria principal a Disco", "PID", pcb.PID)

	//Conectarse con memoria y enviar PCB
	bodyRequest, err := json.Marshal(pcb.PID)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		panic(err)
	}
	url := fmt.Sprintf("http://%s:%d/memoria/swapIn", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)
	slog.Debug("Enviando PCB a memoria", slog.String("url", url))

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PCB a ip:%s puerto:%d", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)
	}
	defer resp.Body.Close()

	//Memoria libera espacio, así que nos envía un OK
	if resp.StatusCode == http.StatusOK {
		slog.Info("Memoria swapeó el proceso y liberó espacio para que entre otro a ready")
	} else {
		slog.Warn("Memoria respondió con error al swapear PCB", slog.Int("status", resp.StatusCode))
	}

	//Se intenta desencolar procesos en SuspReady en la rutina normal
}

// Una vez que el proceso llegue a SUSP. READY tendrá el mismo comportamiento,
//
//	es decir, utilizará el mismo algoritmo que la cola NEW teniendo más
//	 prioridad que esta última. De esta manera, ningún proceso que esté
//	  esperando en la cola de NEW podrá ingresar al sistema si hay al
//	   menos un proceso en SUSP. READY.
func mediumScheduleFIFO() {
	if models.QueueSuspReady.Size() == 0 {
		return
	}
	slog.Debug("Ya entré al algoritmo FIFO en mediano plazo")
	process, _ := models.QueueSuspReady.Get(0)

	err := requestMemorySpace(process.PID, process.Size, process.PseudocodePath)
	if err != nil {
		slog.Warn("Memoria insuficiente para proceso", "PID", process.PID)
		return
	}
	slog.Debug("Ahora voy a remover de SuspReady el proceso")
	models.QueueSuspReady.Remove(0) // Elimina el primer proceso de la cola NEW
	TransitionState(&process, models.EstadoSuspendidoReady, models.EstadoReady)
	models.QueueReady.Add(process)

}

func mediumScheduleShortestFirst() {
	slog.Debug("Ya entré al algoritmo PMCP en mediano plazo")
	if models.QueueSuspReady.Size() == 0 {
		return
	}

	var slice []models.PCB
	for i := 0; i < models.QueueSuspReady.Size(); i++ {
		value, _ := models.QueueSuspReady.Get(i)
		slice = append(slice, value)
	}

	// Ordenar los procesos por tamaño (ascendente)
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Size < slice[j].Size
	})

	// Verificar si hay suficiente memoria para el primer proceso en la cola NEW
	process := slice[0]
	err := requestMemorySpace(process.PID, process.Size, process.PseudocodePath)
	if err != nil {
		slog.Warn("Memoria insuficiente para proceso", "PID", process.PID)
		return
	}

	// Si hay espacio, mover a READY
	// Eliminar solo el primer proceso (más chico) de la cola NEW
	slog.Debug("Ahora voy a remover de SuspReady el proceso")
	models.QueueSuspReady.Remove(0) // Eliminar el primer proceso de la cola NEW
	TransitionState(&process, models.EstadoSuspendidoReady, models.EstadoReady)
	models.QueueReady.Add(process) // Agregarlo a la cola READY
}
