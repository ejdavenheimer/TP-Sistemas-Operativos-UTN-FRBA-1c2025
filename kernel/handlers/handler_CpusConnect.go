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

func ConnectCpuHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var cpuConnected cpuModel.CpuN
		err := json.NewDecoder(request.Body).Decode(&cpuConnected)

		if err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		cpuConnected.IsFree = true

		slog.Debug(fmt.Sprintf("CPU conectada: %v", cpuConnected))

		//Guarda el dispositivo en el map de dispositivos conectados
		key := strconv.Itoa(cpuConnected.Id)
		models.ConnectedCpuMap.Set(key, cpuConnected)
		writer.WriteHeader(http.StatusOK)
	}
}
