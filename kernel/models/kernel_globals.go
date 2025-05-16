package models

import (
	"time"

	cpuModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/helpers"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

type Config struct {
	IpMemory           string  `json:"ip_memory"`
	PortMemory         int     `json:"port_memory"`
	IpKernel           int     `josn:"ip_kernel"`
	PortKernel         int     `json:"port_kernel"`
	SchedulerAlgorithm string  `json:"scheduler_algorithm"`
	NewAlgorithm       string  `json:"new_algorithm"`
	Alpha              float64 `json:"alpha"`
	InitialEstimate    int     `json:"initial_estimate"`
	SuspensionTime     int     `json:"suspension_time"`
	LogLevel           string  `json:"log_level"`
}

var KernelConfig *Config

type DeviceRequest struct {
	Pid            int
	SuspensionTime int
}

type SyscallRequest struct {
	Pid    int
	Type   string
	Values []string
}

var ConnectedDevicesMap = helpers.DeviceMap{M: make(map[string]models.Device)} //TODO: borrar despues
var ConnectedDeviceList list.ArrayList[models.Device]

var ConnectedCpuMap = helpers.CpuMap{M: make(map[string]cpuModels.CpuN)}

type Estado string

const (
	EstadoNew             Estado = "NEW"
	EstadoReady           Estado = "READY"
	EstadoExecuting       Estado = "EXECUTING"
	EstadoBlocked         Estado = "BLOCKED"
	EstadoExit            Estado = "EXIT"
	EstadoSuspendidoReady Estado = "SUSPREADY"
)

type PCB struct {
	PID            int                      // Identificador único del proceso
	ParentPID      int                      // Identificador del proceso padre
	PC             int                      // Program Counter
	ME             map[Estado]int           // Métricas de Estado: cuántas veces pasó por cada estado
	MT             map[Estado]time.Duration // Métricas de Tiempo por Estado
	EstadoActual   Estado                   // Para saber en qué estado está actualmente
	UltimoCambio   time.Time                // Para medir el tiempo que pasa en cada estado
	PseudocodePath string
	Rafaga         float32
	Size           int
}

type MemoryRequest struct {
	PID  int    `json:"pid"`
	Size int    `json:"size"`
	Path string `json:"path"`
}

type EstadoPlanificador string

const (
	EstadoPlanificadorDetenido EstadoPlanificador = "STOP"
	EstadoPlanificadorActivo   EstadoPlanificador = "START"
)

type PCBExecuteRequest struct {
	PID           int
	PC            int
	StatusCodePCB StatusCodePCB
}

type StatusCodePCB int

const (
	NeedFinish    StatusCodePCB = 100
	NeedReplan    StatusCodePCB = 101
	NeedInterrupt StatusCodePCB = 102
)
