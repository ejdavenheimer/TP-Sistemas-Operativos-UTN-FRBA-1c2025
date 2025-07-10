package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/helpers"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	cpuModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// INICIA PLANIFICADOR DE CORTO PLAZO
func StartShortTermScheduler() {
	for {
		<-models.NotifyReady // Espera bloqueado hasta recibir señal
		for models.QueueReady.Size() > 0 {
			ok := SelectToExecute()
			if !ok {
				slog.Debug("No se pudo seleccionar ningún proceso para ejecutar.")
				break // Si no hay CPU libre, salgo del for interno y espero nueva señal
			}
		}
	}
}

// Se activa cuando hay un proceso en la cola de READY.
// 1.Busca una CPU libre -> 2. aca un proceso de la cola de READY -> 3.Trnasiciona el estado a EXEC -> 4.Marca CPU como ocupada
// 5. Ejecuta el proceso en otra Goroutine -> 6. Al finalizar el proceso (o ser interrumpido), libera la CPU
func SelectToExecute() bool {
	slog.Debug("Buscando CPU libre para ejecutar proceso...")

	// Buscamos una CPU disponible para ejecutar nuestro proceso
	cpu, ok := models.ConnectedCpuMap.GetFirstFree()
	if !ok {
		slog.Debug("No hay CPU disponible.")
		return false // No hay CPU libre, no se puede ejecutar un proceso
	}

	pcb, err := models.QueueReady.Get(0)

	if err != nil {
		slog.Warn("Error obteniendo proceso de la cola READY:", "error", err)
		return false //Error al obtener el proceso
	}
	//Lo saca de la cola READY. Ya está listo para ejecutarse.
	models.QueueReady.Remove(0)
	TransitionState(pcb, models.EstadoExecuting)
	//Lo pasamos a la cola de EXECUTING
	models.QueueExec.Add(pcb)

	CPUID := assignProcessToCPU(pcb, cpu)

	runProcessInCPU(pcb, *cpu, CPUID)

	return true
}

// Asigna el proceso a la CPU
func assignProcessToCPU(pcb *models.PCB, cpu *cpuModels.CpuN) string {
	cpu.PIDExecuting = pcb.PID
	CPUID := strconv.Itoa(cpu.Id)
	models.ConnectedCpuMap.Set(CPUID, cpu)

	slog.Debug("CPU marcada como ocupada", "cpu_id", cpu.Id, "pid", cpu.PIDExecuting)
	slog.Debug("Asignando proceso a CPU", "pid", pcb.PID, "cpu_id", cpu.Id)
	return CPUID
}

func runProcessInCPU(pcb *models.PCB, cpu cpuModels.CpuN, CPUID string) {
	inicioEjecucion := time.Now()
	result := ExecuteProcess(pcb, cpu)

	tiempoEjecutado := time.Since(inicioEjecucion)

	pcb.Mutex.Lock()
	pcb.PC = result.PC
	// Si el algoritmo escogido es
	if models.KernelConfig.SchedulerAlgorithm == "SJF" || models.KernelConfig.SchedulerAlgorithm == "SRT" {
		pcb.RafagaReal = float32(tiempoEjecutado.Milliseconds())
		// Est(n+1)        =                α          * R(n)           + (1 - α)                      * Est(n)
		pcb.RafagaEstimada = models.KernelConfig.Alpha*pcb.RafagaReal + (1-models.KernelConfig.Alpha)*pcb.RafagaEstimada
	}
	// Remover el proceso de la cola EXEC antes de cambiar de estado
	index := findProcessIndexByPID(models.QueueExec, pcb.PID)
	if index != -1 && result.StatusCodePCB != models.NeedExecuteSyscall {
		models.QueueExec.Remove(index)
	}

	pcb.Mutex.Unlock()

	switch result.StatusCodePCB {
	case models.NeedFinish:
		slog.Info(fmt.Sprintf("## (%d) - Terminando ejecución, pasando a EXIT", pcb.PID))
		TransitionState(pcb, models.EstadoExit)
		models.QueueExit.Add(pcb)

	case models.NeedReplan:
		slog.Info(fmt.Sprintf("## (%d) - Replanificando, pasando a READY", pcb.PID))
		TransitionState(pcb, models.EstadoReady)
		AddProcessToReady(pcb)

	case models.NeedInterrupt:
		slog.Info(fmt.Sprintf("## (%d) - Desalojado por algoritmo SJF/SRT, pasando a READY", pcb.PID))
		TransitionState(pcb, models.EstadoReady)
		AddProcessToReady(pcb)
	case models.NeedExecuteSyscall:
		syscallName := result.SyscallRequest.Type
		slog.Info(fmt.Sprintf("## %d - Solicitó syscall: %s", result.SyscallRequest.Pid, syscallName))
		if syscallName == "IO" {
			// Primero chequea si existe el dispositivo
			deviceRequested, index, exists := models.ConnectedDeviceList.Find(func(d ioModel.Device) bool {
				return result.SyscallRequest.Values[0] == d.Name
			})

			if !exists || deviceRequested.Name == "" {
				slog.Error(fmt.Sprintf("No se encontro al dispositivo %s", result.SyscallRequest.Values[0]))
				EndProcess(result.SyscallRequest.Pid, fmt.Sprintf("No se encontro al dispositivo %s", result.SyscallRequest.Values[0]))
				//server.SendJsonResponse(writer, map[string]interface{}{
				//	"action": "exit",
				//})
				return
			}

			// Se crea una lista auxiliar con dispositivos del mismo tipo
			connectedDeviceListAux := models.ConnectedDeviceList.FindAll(func(d ioModel.Device) bool {
				return result.SyscallRequest.Values[0] == d.Name
			})

			// Se busca los dispositivos que se encuentren libres
			deviceRequestedAux, indexAux, exists := connectedDeviceListAux.Find(func(d ioModel.Device) bool {
				return d.IsFree
			})

			// En caso de que no encuentre ningún dispositivo libre, se bloquea el proceso
			if index < 0 || !exists {
				slog.Debug("El dispositivo se encuentra ocupado...")
				helpers.AddPidForDevice(deviceRequested.Name, int(result.SyscallRequest.Pid))
				BlockedProcess(result.SyscallRequest.Pid, fmt.Sprintf("El dispositivo %s no se encuentra disponible", result.SyscallRequest.Type))
				//server.SendJsonResponse(writer, map[string]interface{}{
				//	"action": "block",
				//})
				return
			}

			deviceRequestedAux.IsFree = false
			deviceRequestedAux.PID = result.SyscallRequest.Pid
			err := models.ConnectedDeviceList.Set(index+indexAux, deviceRequestedAux)
			if err != nil {
				slog.Error(fmt.Sprintf("error: %v", err))
				return
			}

			device := ioModel.Device{Ip: deviceRequestedAux.Ip, Port: deviceRequestedAux.Port, Name: result.SyscallRequest.Values[0]}
			sleepTime, _ := strconv.Atoi(result.SyscallRequest.Values[1])
			err = SleepDevice(result.SyscallRequest.Pid, sleepTime, device)

			if err != nil {
				slog.Error(fmt.Sprintf("error: %v", err))
				return
			}

			//Acá se bloquea el proceso
			//server.SendJsonResponse(writer, map[string]interface{}{
			//	"action": "block",
			//})
		}

		if syscallName == "DUMP_MEMORY" {
			pcb, index, exists := models.QueueExec.Find(func(pcb *models.PCB) bool {
				return pcb.PID == result.SyscallRequest.Pid
			})
			if !exists || index == -1 {
				slog.Warn("TODO: ver que pasa en este caso por ahora hago un exit")
				//server.SendJsonResponse(writer, map[string]string{
				//	"action": "exit",
				//})
				return
			}
			DumpServices(uint(pcb.PID), pcb.Size)
			//server.SendJsonResponse(writer, map[string]string{
			//	"action": "continue",
			//})
		}
	}
	// Liberar CPU
	models.ConnectedCpuMap.MarkAsFree(CPUID)

	select {
	case models.NotifyReady <- 1:
		// Se pudo enviar la notificación
	default:
		// Ya hay una notificación pendiente, no hacemos nada
	}

}

func ExecuteProcess(pcb *models.PCB, cpu cpuModels.CpuN) models.PCBExecuteRequest {
	var pcbExecute models.PCBExecuteRequest
	pcbExecute.PID = pcb.PID
	pcbExecute.PC = pcb.PC

	//Prepara el Request a Json para envíar a cpu
	bodyRequest, err := json.Marshal(pcbExecute)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		return models.PCBExecuteRequest{}
	}

	//Envía a CPU
	url := fmt.Sprintf("http://%s:%d/cpu/exec", cpu.Ip, cpu.Port)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PCB", "ip", cpu.Ip, "port", cpu.Port, "err", err)
		models.ConnectedCpuMap.MarkAsFree(fmt.Sprint(cpu.Id))
		return models.PCBExecuteRequest{StatusCodePCB: models.NeedReplan}
	}
	defer resp.Body.Close()

	//Recibe PID y motivo de terminación de ejecución. (Verifica status HTTP)
	if resp.StatusCode != http.StatusOK {
		slog.Error("Respuesta inesperada de CPU", "status", resp.StatusCode)
		return models.PCBExecuteRequest{StatusCodePCB: models.NeedReplan}
	}
	//Decodificar respuesta JSON
	err = json.NewDecoder(resp.Body).Decode(&pcbExecute)
	if err != nil {
		slog.Error("Error al decodificar el cuerpo de la respuesta de CPU:", "error", err)
		return models.PCBExecuteRequest{StatusCodePCB: models.NeedReplan}
	}
	return pcbExecute
}

// Actualiza el estado de un proceso, su LOG y sus metricas.
func TransitionState(pcb *models.PCB, newState models.Estado) {
	oldState := pcb.EstadoActual
	if oldState == newState {
		slog.Warn(fmt.Sprintf("## (%d) El proceso ya está en el estado %s, no se realiza la transición.", pcb.PID, newState))
		return
	}

	pcb.Mutex.Lock()
	defer pcb.Mutex.Unlock()

	if pcb.ME == nil {
		pcb.ME = make(map[models.Estado]int)
	}
	if pcb.MT == nil {
		pcb.MT = make(map[models.Estado]time.Duration)
	}

	//Calculas el tiempo que estuvo en el estado anterior
	duration := time.Since(pcb.UltimoCambio)
	pcb.MT[oldState] += duration
	// Incrementa en 1 la cantidad de veces que el proceso estuvo en el estado anterior
	pcb.ME[oldState]++

	//Log de cambio de estado
	slog.Info(fmt.Sprintf("## (%d) Pasa del estado %s al estado %s", pcb.PID, oldState, newState))

	//Registramos el cambio en las metricas
	pcb.EstadoActual = newState
	pcb.UltimoCambio = time.Now()
}

func StartSuspensionTimer(pcb *models.PCB) {
	slog.Debug("Timer iniciado para posible suspensión de proceso", "PID", pcb.PID)

	time.Sleep(time.Duration(models.KernelConfig.SuspensionTime) * time.Millisecond)

	//pcb.Mutex.Lock()
	//defer pcb.Mutex.Unlock()

	// Si todavía está en estado BLOCKED, debe pasar a SUSP.BLOCKED
	if pcb.EstadoActual == models.EstadoBlocked {
		// Pasamos a SUSP.BLOCKED
		slog.Info(fmt.Sprintf("## (%d) - Proceso pasa a SUSP.BLOCKED", pcb.PID))
		TransitionState(pcb, models.EstadoSuspendidoBlocked)

		index := findProcessIndexByPID(models.QueueBlocked, pcb.PID)
		if index != -1 {
			models.QueueBlocked.Remove(index)
		}

		models.QueueSuspBlocked.Add(pcb)
		NotifyToMediumScheduler()
	}
}

func AddProcessToReady(pcb *models.PCB) {
	pcb.Mutex.Lock()
	defer pcb.Mutex.Unlock() // Desbloquea el mutex al final de la función

	switch models.KernelConfig.SchedulerAlgorithm {
	case "FIFO":
		// En FIFO agrego al final sin ordenar
		models.QueueReady.Add(pcb)

	case "SJF", "SRT": // Inserto ordenadamente en la cola READY, ordenada por Rafaga ascendente
		procesoInsertado := false
		for i := 0; i < models.QueueReady.Size(); i++ {
			procesoIterado, _ := models.QueueReady.Get(i)
			if pcb.RafagaEstimada < procesoIterado.RafagaEstimada {
				models.QueueReady.Insert(i, pcb) // Inserto el proceso en la posición i
				procesoInsertado = true
				break
			}
		}
		if !procesoInsertado { // Si no se insertó antes, lo agrego al final
			models.QueueReady.Add(pcb)
		}

		if models.KernelConfig.SchedulerAlgorithm == "SRT" {
			//Busca el proceso (PCB) que se esta ejecutando con la mayor rafaga restante estimada
			processToInterrupt := GetPCBConMayorRafagaRestante()
			if processToInterrupt == nil {
				slog.Debug("No hay procesos ejecutándose para interrumpir")
				return
			}
			if pcb.RafagaEstimada < processToInterrupt.RafagaEstimada {
				//GetCPUByPid recorre las CPUs conectadas y retorna la qe esta ejecutando el PID solicitado
				cpu := models.ConnectedCpuMap.GetCPUByPid(processToInterrupt.PID)
				//SI ES POSITIVO, SE CONECTA AL ENDPOINT DE CPU PARA PEDIRLE QUE DESALOJE AL PROCESO TAL
				SendInterruption(processToInterrupt.PID, cpu.Port, cpu.Ip)
			}

		}
	}
	select {
	case models.NotifyReady <- 1:
		// Se pudo enviar la notificación
	default:
		// Ya hay una notificación pendiente, no hacemos nada
	}

}
