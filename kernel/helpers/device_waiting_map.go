package helpers

import "sync"

var (
	// PidWaitingForDevice mapea el nombre de un dispositivo a un slice de PIDs que esperan por él.
	PidWaitingForDevice = make(map[string][]int)
	pidMutex            sync.RWMutex // Mutex para proteger el mapa de accesos concurrentes
)

// AddPidForDevice agrega un PID al slice de PIDs de un dispositivo.
// Si el dispositivo no existe en el mapa, se crea una nueva entrada.
func AddPidForDevice(deviceName string, pid int) {
	pidMutex.Lock() // Bloquear para escritura
	defer pidMutex.Unlock()

	PidWaitingForDevice[deviceName] = append(PidWaitingForDevice[deviceName], pid)
}

// GetPidsForDevice obtiene todos los PIDs asociados a un dispositivo.
// Devuelve el slice de PIDs y un booleano indicando si el dispositivo fue encontrado.
func GetPidsForDevice(deviceName string) ([]int, bool) {
	pidMutex.RLock() // Bloquear para lectura
	defer pidMutex.RUnlock()

	pids, found := PidWaitingForDevice[deviceName]
	return pids, found
}

// RemoveDeviceFromWaiting elimina completamente la entrada de un dispositivo del mapa.
func RemoveDeviceFromWaiting(deviceName string) {
	pidMutex.Lock() // Bloquear para escritura
	defer pidMutex.Unlock()

	delete(PidWaitingForDevice, deviceName)
}

// RemoveSpecificPidFromDevice elimina un PID específico del slice de un dispositivo.
func RemoveSpecificPidFromDevice(deviceName string, pid int) {
	pidMutex.Lock()
	defer pidMutex.Unlock()

	pids, found := PidWaitingForDevice[deviceName]
	if !found {
		return // El dispositivo no está en el mapa
	}

	// Crear un nuevo slice para almacenar los PIDs que queremos mantener
	newPids := []int{}
	for _, existingPid := range pids {
		if existingPid != pid {
			newPids = append(newPids, existingPid)
		}
	}

	if len(newPids) == 0 {
		// Si no quedan PIDs después de eliminar, borrar la entrada del dispositivo
		delete(PidWaitingForDevice, deviceName)
	} else {
		// Actualizar el slice con los PIDs restantes
		PidWaitingForDevice[deviceName] = newPids
	}
}

// IsPidWaitingForDevice verifica si un PID específico está en la lista de espera de un dispositivo.
func IsPidWaitingForDevice(deviceName string, pid int) bool {
	pidMutex.RLock()
	defer pidMutex.RUnlock()

	pids, found := PidWaitingForDevice[deviceName]
	if !found {
		return false
	}

	for _, existingPid := range pids {
		if existingPid == pid {
			return true
		}
	}
	return false
}

// GetAndRemoveOnePidForDevice obtiene el primer PID de un dispositivo y lo elimina del slice.
// Devuelve el PID, y un booleano indicando si un PID fue encontrado y devuelto.
func GetAndRemoveOnePidForDevice(deviceName string) (int, bool) {
	pidMutex.Lock() // Bloquear para escritura, ya que vamos a modificar el slice
	defer pidMutex.Unlock()

	pids, found := PidWaitingForDevice[deviceName]
	if !found || len(pids) == 0 {
		// El dispositivo no está en el mapa o no tiene PIDs esperando
		return 0, false
	}

	// Obtener el primer PID del slice
	pid := pids[0]

	// Eliminar el primer PID del slice
	if len(pids) == 1 {
		// Si solo queda un PID, eliminar la entrada completa del dispositivo del mapa
		delete(PidWaitingForDevice, deviceName)
	} else {
		// Si hay más de un PID, eliminar el primero recreando el slice sin él
		PidWaitingForDevice[deviceName] = pids[1:]
	}

	return pid, true
}

// HasMoreThanOnePidWaiting verifica si la cantidad total de PIDs esperando en todos los dispositivos
// es mayor que uno.
func HasMoreThanOnePidWaiting() bool {
	pidMutex.RLock() // Bloquear para lectura concurrente
	defer pidMutex.RUnlock()

	totalPids := 0
	for _, pids := range PidWaitingForDevice {
		totalPids += len(pids)
		// Optimización: si ya encontramos más de un PID, podemos salir temprano
		if totalPids >= 1 {
			return true
		}
	}

	return false
}

// GetAndRemoveAnyWaitingPid obtiene el primer PID de CUALQUIER dispositivo que tenga PIDs esperando
// y lo elimina de su respectiva lista. Devuelve el PID encontrado, el nombre del dispositivo al que pertenecía, y un booleano indicando si un PID fue encontrado y devuelto.
func GetAndRemoveAnyWaitingPid() (int, string, bool) {
	pidMutex.Lock() // Bloqueamos el mapa completo para esta operación de escritura/lectura
	defer pidMutex.Unlock()

	// Iteramos sobre el mapa de dispositivos esperando PIDs
	for deviceName, pids := range PidWaitingForDevice {
		if len(pids) > 0 {
			// Hemos encontrado un dispositivo que tiene PIDs esperando.
			// Procedemos a tomar el primer PID de este slice.
			pid := pids[0]

			// Ahora, eliminamos este PID del slice
			if len(pids) == 1 {
				// Si este era el último PID para este dispositivo, eliminamos la entrada del mapa
				delete(PidWaitingForDevice, deviceName)
			} else {
				// Si quedan más PIDs, actualizamos el slice para excluir el primero
				PidWaitingForDevice[deviceName] = pids[1:]
			}
			// Devolvemos el PID, el nombre del dispositivo y true
			return pid, deviceName, true
		}
	}
	// Si el bucle termina, significa que ningún dispositivo tiene PIDs esperando
	return 0, "", false
}
