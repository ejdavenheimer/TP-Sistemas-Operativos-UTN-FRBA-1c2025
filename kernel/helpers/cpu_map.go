package helpers

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
)

type CpuMap struct {
	mx sync.Mutex
	M  map[string]*models.CpuN
}

func (sMap *CpuMap) Set(key string, value *models.CpuN) {
	sMap.mx.Lock()
	sMap.M[key] = value
	sMap.mx.Unlock()
}

func (sMap *CpuMap) Delete(key string) *models.CpuN {
	sMap.mx.Lock()
	cpu := sMap.M[key]
	delete(sMap.M, key)
	sMap.mx.Unlock()

	return cpu
}

func (sMap *CpuMap) Get(key string) (*models.CpuN, bool) {
	sMap.mx.Lock()
	cpu, find := sMap.M[key]
	sMap.mx.Unlock()

	return cpu, find
}

func (sMap *CpuMap) GetAll() {
	sMap.mx.Lock()
	defer sMap.mx.Unlock()

	if len(sMap.M) == 0 {
		slog.Debug("No hay CPUs conectados.")
		return
	}

	slog.Debug("CPUs conectados:")
	for key, cpus := range sMap.M {
		slog.Debug(fmt.Sprintf("- Key: %s, Cpu: %+v", key, cpus))
	}
}

func (sMap *CpuMap) GetFirstFree() (*models.CpuN, bool) {
	sMap.mx.Lock()
	defer sMap.mx.Unlock()

	for _, cpu := range sMap.M {
		if cpu.IsFree {
			cpu.IsFree = false // Marcar como ocupada
			return cpu, true
		}
	}
	return nil, false
}

func (sMap *CpuMap) MarkAsFree(id string) {
	sMap.mx.Lock()
	defer sMap.mx.Unlock()
	if cpu, ok := sMap.M[id]; ok {
		cpu.IsFree = true
	}
}

func (cpuMap *CpuMap) GetCPUByPid(pid int) *models.CpuN {
	cpuMap.mx.Lock()
	defer cpuMap.mx.Unlock()

	for _, cpu := range cpuMap.M {
		if cpu.PIDExecuting == pid {
			return cpu
		}
	}
	return nil
}

// func (sMap *CpuMap) GetMaxRafagaPCBExecuting() models.CpuN {
// 	sMap.mx.Lock()
// 	defer sMap.mx.Unlock()

// 	var max models.CpuN
// 	max.PIDRafaga = -1 // Valor imposible para inicializar

// 	for _, cpu := range sMap.M {
// 		if !cpu.IsFree && cpu.PIDRafaga > max.PIDRafaga {
// 			max = cpu
// 		}
// 	}

// 	return max
// }
