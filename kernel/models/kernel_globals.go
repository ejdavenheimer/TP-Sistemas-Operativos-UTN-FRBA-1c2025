package models

import (
	"sync"
	"time"

	cpuModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
)

// --- Estructura de Configuración ---

type Config struct {
	IpMemory           string  `json:"ip_memory"`
	PortMemory         int     `json:"port_memory"`
	IpKernel           string  `json:"ip_kernel"`
	PortKernel         int     `json:"port_kernel"`
	SchedulerAlgorithm string  `json:"scheduler_algorithm"`
	NewAlgorithm       string  `json:"new_algorithm"`
	Alpha              float32 `json:"alpha"`
	InitialEstimate    int     `json:"initial_estimate"`
	SuspensionTime     int     `json:"suspension_time"`
	LogLevel           string  `json:"log_level"`
}

var KernelConfig *Config

// --- Estados del Sistema ---

type Estado string

const (
	EstadoNew               Estado = "NEW"
	EstadoReady             Estado = "READY"
	EstadoExecuting         Estado = "EXECUTING"
	EstadoBlocked           Estado = "BLOCKED"
	EstadoExit              Estado = "EXIT"
	EstadoSuspendidoReady   Estado = "SUSPREADY"
	EstadoSuspendidoBlocked Estado = "SUSPEND_BLOCKED"
)

type EstadoPlanificador string

const (
	EstadoPlanificadorDetenido EstadoPlanificador = "STOP"
	EstadoPlanificadorActivo   EstadoPlanificador = "START"
)

var SchedulerState EstadoPlanificador = EstadoPlanificadorDetenido

// --- Estructura Principal del Proceso (PCB) ---

type PCB struct {
	PID            uint
	ParentPID      int
	PC             int
	ME             map[Estado]int
	MT             map[Estado]time.Duration
	EstadoActual   Estado
	UltimoCambio   time.Time
	PseudocodePath string
	RafagaReal     float32
	Size           int
	RafagaEstimada float32
	Mutex          sync.Mutex
}

// --- Estructuras de Comunicación y Syscalls ---

type StatusCodePCB int

const (
	NeedFinish         StatusCodePCB = 100
	NeedReplan         StatusCodePCB = 101
	NeedInterrupt      StatusCodePCB = 102
	NeedExecuteSyscall StatusCodePCB = 103
)

type SyscallRequest struct {
	Pid    uint
	Type   string
	Values []string
}

type PCBExecuteRequest struct {
	PID            uint
	PC             int
	StatusCodePCB  StatusCodePCB
	SyscallRequest SyscallRequest
}

type MemoryRequest struct {
	PID  uint   `json:"pid"`
	Size int    `json:"size"`
	Path string `json:"path"`
}

// DeviceRequest es la estructura que el Kernel envía a un módulo de I/O.
type DeviceRequest struct {
	Pid            uint
	SuspensionTime int
}

// --- Gestores de Recursos ---

var ConnectedCpuMap = CpuMap{M: make(map[string]*cpuModels.CpuN)}
var ConnectedDeviceManager = NewDeviceManager()
var WaitingForDeviceManager = NewWaitingProcessManager()

// --- Canales de Notificación para Planificadores ---

var NotifyReady = make(chan int, 1)
var NotifyLongScheduler = make(chan int, 1)
var NotifyMediumScheduler = make(chan int, 1)
