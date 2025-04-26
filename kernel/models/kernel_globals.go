package models

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/helpers"
	"time"
)

type Config struct {
	IpMemory           string  `json:"ip_memory"`
	PortMemory         int     `json:"port_memory"`
	PortKernel         int     `json:"port_kernel"`
	SchedulerAlgorithm string  `json:"scheduler_algorithm"`
	NewAlgorithm       string  `json:"new_algorithm"`
	Alpha              float64 `json:"alpha"`
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

var ConnectedDevicesMap = helpers.DeviceMap{M: make(map[string]models.Device)}

type Estado string
type PCB struct {
	PID          int                      // Identificador único del proceso
	PC           int                      // Program Counter
	ME           map[Estado]int           // Métricas de Estado: cuántas veces pasó por cada estado
	MT           map[Estado]time.Duration // Métricas de Tiempo por Estado
	EstadoActual Estado                   // Para saber en qué estado está actualmente
	UltimoCambio time.Time                // Para medir el tiempo que pasa en cada estado
}
