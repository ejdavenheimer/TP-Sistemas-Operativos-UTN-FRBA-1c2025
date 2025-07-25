package models

import (
	"strconv"
	"sync"

	cpuModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	ioModels "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

// --- Gestor de CPUs ---

type CpuMap struct {
	mx sync.Mutex
	M  map[string]*cpuModels.CpuN
}

func (sMap *CpuMap) Set(key string, value *cpuModels.CpuN) {
	sMap.mx.Lock()
	defer sMap.mx.Unlock()
	sMap.M[key] = value
}

func (sMap *CpuMap) GetFirstFree() (*cpuModels.CpuN, bool) {
	sMap.mx.Lock()
	defer sMap.mx.Unlock()
	for _, cpu := range sMap.M {
		if cpu.IsFree {
			cpu.IsFree = false
			return cpu, true
		}
	}
	return nil, false
}

func (sMap *CpuMap) MarkAsFree(id int) {
	sMap.mx.Lock()
	defer sMap.mx.Unlock()
	key := strconv.Itoa(id)
	if cpu, ok := sMap.M[key]; ok {
		cpu.IsFree = true
	}
}

func (sMap *CpuMap) GetCPUByPid(pid uint) *cpuModels.CpuN {
	sMap.mx.Lock()
	defer sMap.mx.Unlock()
	for _, cpu := range sMap.M {
		if cpu.PIDExecuting == pid {
			return cpu
		}
	}
	return nil
}

// --- Gestor de Dispositivos de I/O ---

type DeviceManager struct {
	mx      sync.Mutex
	devices map[string][]*ioModels.Device
}

func NewDeviceManager() *DeviceManager {
	return &DeviceManager{
		devices: make(map[string][]*ioModels.Device),
	}
}

func (dm *DeviceManager) Add(device *ioModels.Device) {
	dm.mx.Lock()
	defer dm.mx.Unlock()
	dm.devices[device.Name] = append(dm.devices[device.Name], device)
}

func (dm *DeviceManager) GetFreeByName(name string) (*ioModels.Device, bool) {
	dm.mx.Lock()
	defer dm.mx.Unlock()
	deviceList, exists := dm.devices[name]
	if !exists {
		return nil, false
	}
	for _, device := range deviceList {
		if device.IsFree {
			device.IsFree = false
			return device, true
		}
	}
	return nil, true
}

func (dm *DeviceManager) MarkAsFreeByPort(port int) (*ioModels.Device, bool) {
	dm.mx.Lock()
	defer dm.mx.Unlock()
	for _, deviceList := range dm.devices {
		for _, device := range deviceList {
			if device.Port == port {
				device.IsFree = true
				device.PID = 0
				return device, true
			}
		}
	}
	return nil, false
}

// **NUEVA FUNCIÓN**
// GetPidByPort devuelve el PID del proceso que se está ejecutando en un dispositivo específico.
func (dm *DeviceManager) GetPidByPort(port int) uint {
	dm.mx.Lock()
	defer dm.mx.Unlock()
	for _, deviceList := range dm.devices {
		for _, device := range deviceList {
			if device.Port == port {
				return device.PID
			}
		}
	}
	return 0 // Retorna 0 si no se encuentra o no hay proceso asignado
}

// **NUEVA FUNCIÓN**
// RemoveByPort elimina un dispositivo de la lista de conectados, identificado por su puerto.
func (dm *DeviceManager) RemoveByPort(port int) {
	dm.mx.Lock()
	defer dm.mx.Unlock()
	for name, deviceList := range dm.devices {
		newList := []*ioModels.Device{}
		for _, device := range deviceList {
			if device.Port != port {
				newList = append(newList, device)
			}
		}
		if len(newList) == 0 {
			delete(dm.devices, name)
		} else {
			dm.devices[name] = newList
		}
	}
}

// --- Gestor de Procesos en Espera de I/O ---

type WaitingProcessManager struct {
	mx     sync.Mutex
	queues map[string]*list.ArrayList[*PCB]
}

func NewWaitingProcessManager() *WaitingProcessManager {
	return &WaitingProcessManager{
		queues: make(map[string]*list.ArrayList[*PCB]),
	}
}

func (wm *WaitingProcessManager) Enqueue(deviceName string, pcb *PCB) {
	wm.mx.Lock()
	defer wm.mx.Unlock()
	if _, exists := wm.queues[deviceName]; !exists {
		wm.queues[deviceName] = &list.ArrayList[*PCB]{}
	}
	wm.queues[deviceName].Add(pcb)
}

func (wm *WaitingProcessManager) Dequeue(deviceName string) (*PCB, bool) {
	wm.mx.Lock()
	defer wm.mx.Unlock()
	queue, exists := wm.queues[deviceName]
	if !exists || queue.Size() == 0 {
		return nil, false
	}
	pcb, err := queue.Dequeue()
	if err != nil {
		return nil, false
	}
	return pcb, true
}
