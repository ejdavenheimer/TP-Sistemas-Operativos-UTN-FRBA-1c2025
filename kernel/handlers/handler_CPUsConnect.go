package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	cpuModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

// ConnectCpuHandler maneja las solicitudes de conexión de nuevas CPUs.
func ConnectCpuHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var cpuConnected cpuModel.CpuN
		err := json.NewDecoder(request.Body).Decode(&cpuConnected)

		if err != nil {
			http.Error(writer, "Error decodificando los datos de la CPU", http.StatusBadRequest)
			return
		}

		cpuConnected.IsFree = true

		slog.Info(fmt.Sprintf("CPU conectada: ID=%d en %s:%d", cpuConnected.Id, cpuConnected.Ip, cpuConnected.Port))

		// Usamos nuestro helper seguro para guardar la nueva CPU.
		key := strconv.Itoa(cpuConnected.Id)
		models.ConnectedCpuMap.Set(key, &cpuConnected)

		writer.WriteHeader(http.StatusOK)
	}
}
