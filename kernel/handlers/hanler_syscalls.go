package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/services"
)

// InitProcSyscallHandler maneja la syscall síncrona INIT_PROC.
func InitProcSyscallHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		var syscallRequest models.SyscallRequest
		err := json.NewDecoder(request.Body).Decode(&syscallRequest)
		if err != nil {
			http.Error(writer, "Error en la solicitud de syscall", http.StatusBadRequest)
			return
		}

		if syscallRequest.Type == "INIT_PROC" {
			path := syscallRequest.Values[0]
			size, _ := strconv.Atoi(syscallRequest.Values[1])
			parentPIDStr := fmt.Sprintf("%d", syscallRequest.Pid)

			services.InitProcess(path, size, []string{parentPIDStr})

			// Respondemos OK para que la CPU sepa que puede continuar.
			writer.WriteHeader(http.StatusOK)
		} else {
			http.Error(writer, "Syscall no reconocida en este endpoint", http.StatusBadRequest)
		}
	}
}
