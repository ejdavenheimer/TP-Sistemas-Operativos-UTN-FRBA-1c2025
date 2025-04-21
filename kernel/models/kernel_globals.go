package models

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"sync"
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
	Type   string
	Values []string
}

var ConnectedDevicesMap = DeviceMap{m: make(map[string]models.Device)}

type DeviceMap struct {
	mx sync.Mutex
	m  map[string]models.Device
}

// TODO: remover las funciones a un helper
func (sMap *DeviceMap) Set(key string, value models.Device) {
	sMap.mx.Lock()
	sMap.m[key] = value
	sMap.mx.Unlock()
}

func (sMap *DeviceMap) Delete(key string) models.Device {
	sMap.mx.Lock()
	var pcb = sMap.m[key]
	delete(sMap.m, key)
	sMap.mx.Unlock()

	return pcb
}

func (sMap *DeviceMap) Get(key string) (models.Device, bool) {
	sMap.mx.Lock()
	var device, find = sMap.m[key]
	sMap.mx.Unlock()

	return device, find
}
