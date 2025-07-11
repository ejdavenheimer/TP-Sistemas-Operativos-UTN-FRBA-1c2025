package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	memoriaModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

func NotifyToMediumScheduler() {
	select {
	case models.NotifyMediumScheduler <- 1:
	default:
	}
}

func MediumTermScheduler() {
	for {
		//Si ambas colas están vacías, vuelve a mirar en otro momento
		<-models.NotifyMediumScheduler
		slog.Debug("Planificador de mediano plazo iniciado.")

		if models.QueueSuspReady.Size() == 0 && models.QueueSuspBlocked.Size() == 0 {
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
	}
}

// SWAP OUT. En este momento se debe informar al módulo memoria que debe ser
// movido de memoria principal a disco. Cabe aclarar que en este momento vamos
// a tener más memoria libre en el sistema por lo que se debe verificar si uno o
// más nuevos procesos pueden entrar (tanto de la cola NEW como de SUSP. READY).
func movePrincipalMemoryToSwap() {
	//Armar estructura a enviar
	var pcb, _ = models.QueueSuspBlocked.Get(0)

	slog.Debug("Iniciando solicitud para mover el proceso de memoria principal a Disco", "PID", pcb.PID)

	//Conectarse con memoria y enviar PCB
	var request = memoriaModel.PIDRequest{PID: pcb.PID}
	bodyRequest, err := json.Marshal(request)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		return
	}

	slog.Debug("Enviando PCB a memoria para swap out", "PID", pcb.PID)

	resp, err := client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/swapOut", bodyRequest)
	if err != nil {
		slog.Error("Error enviando el PCB a memoria para swap out", "PID", pcb.PID, "error", err)
		return
	}
	defer resp.Body.Close()

	//Memoria libera espacio, así que nos envía un OK
	if resp.StatusCode == http.StatusOK {
		slog.Debug("Memoria swapeó el proceso y liberó espacio para que entre otro a ready", "PID", pcb.PID)
	} else {
		slog.Warn("Memoria respondió con error al swapear PCB", "PID", pcb.PID, "status", resp.StatusCode)
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

	moveSwapToPrincipalMemory(process)
	StartLongTermScheduler()
}

func mediumScheduleShortestFirst() {
	slog.Debug("Ya entré al algoritmo PMCP en mediano plazo")
	if models.QueueSuspReady.Size() == 0 {
		return
	}

	var slice []*models.PCB
	for i := 0; i < models.QueueSuspReady.Size(); i++ {
		value, _ := models.QueueSuspReady.Get(i)
		slice = append(slice, value)
	}

	// Ordenar los procesos por tamaño (ascendente)
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Size < slice[j].Size
	})

	process := slice[0]

	moveSwapToPrincipalMemory(process)
	StartLongTermScheduler()
}

func moveSwapToPrincipalMemory(pcb *models.PCB) {
	slog.Debug("Iniciando solicitud para mover el proceso de swap a memoria principal", "PID", pcb.PID)

	err := CheckUserMemoryCapacity(pcb.PID, pcb.Size)
	if err != nil {
		slog.Warn("Memoria insuficiente para proceso", "PID", pcb.PID, "error", err)
		return
	}
	slog.Debug("Ahora voy a remover de SuspReady el proceso", "PID", pcb.PID)

	index := findProcessIndexByPID(models.QueueSuspReady, pcb.PID)
	if index != -1 {
		models.QueueSuspReady.Remove(index)
	}

	// Conectarse con memoria y enviar PCB
	request := memoriaModel.PIDRequest{PID: pcb.PID}
	bodyRequest, err := json.Marshal(request)
	if err != nil {
		slog.Error("Error al convertir el PCB a JSON", "PID", pcb.PID, "error", err)
		return
	}

	slog.Debug("Enviando PCB a memoria para swap in", "PID", pcb.PID)

	resp, err := client.DoRequest(models.KernelConfig.PortMemory, models.KernelConfig.IpMemory, "POST", "memoria/swapIn", bodyRequest)
	if err != nil {
		slog.Error("Error enviando el PCB a memoria para swap in", "PID", pcb.PID, "error", err)
		return
	}
	defer resp.Body.Close()

	// Swap libera espacio, así que nos envía un OK
	if resp.StatusCode == http.StatusOK {
		slog.Debug("Memoria des-swapeó el proceso, entra a READY", "PID", pcb.PID)
		TransitionState(pcb, models.EstadoReady)
		slog.Info(fmt.Sprintf("## PID: %d - Finalizó IO y pasa a READY", pcb.PID))
		AddProcessToReady(pcb)
	} else {
		slog.Warn("Memoria respondió con error al des-swapear PCB", "PID", pcb.PID, "status", resp.StatusCode)
		TransitionState(pcb, models.EstadoExit)
		models.QueueExit.Add(pcb)
		slog.Error(fmt.Sprintf("## PID: %d - No encontrado en swap, se manda a finalizar.", pcb.PID))
	}
}
