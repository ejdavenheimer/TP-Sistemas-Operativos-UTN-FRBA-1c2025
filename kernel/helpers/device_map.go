package helpers

import (
	"fmt"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"log/slog"
	"sync"
)

type DeviceMap struct {
	mx sync.Mutex
	M  map[string]models.Device
}

func (sMap *DeviceMap) Set(key string, value models.Device) {
	sMap.mx.Lock()
	sMap.M[key] = value
	sMap.mx.Unlock()
}

func (sMap *DeviceMap) Delete(key string) models.Device {
	sMap.mx.Lock()
	var pcb = sMap.M[key]
	delete(sMap.M, key)
	sMap.mx.Unlock()

	return pcb
}

func (sMap *DeviceMap) Get(key string) (models.Device, bool) {
	sMap.mx.Lock()
	var device, find = sMap.M[key]
	sMap.mx.Unlock()

	return device, find
}

func (sMap *DeviceMap) GetAll() {
	sMap.mx.Lock()
	defer sMap.mx.Unlock()

	if len(sMap.M) == 0 {
		slog.Debug("No hay dispositivos conectados.")
		return
	}

	slog.Debug("Dispositivos conectados:")
	for key, device := range sMap.M {
		slog.Debug(fmt.Sprintf("- Key: %s, Device: %+v\n", key, device))
	}
}
