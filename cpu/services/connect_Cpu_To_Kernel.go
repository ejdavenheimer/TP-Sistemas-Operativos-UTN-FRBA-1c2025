package services

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/cpu/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/client"
)

func ConnectToKernel(idCpu int, cpuConfig *models.Config) {
	//Crea y codifica la request de conexion a Kernel
	var request = models.CpuN{Id: idCpu, Ip: cpuConfig.IpCpu, Port: cpuConfig.PortCpu}
	body, err := json.Marshal(request)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}

	//Envia la request de conexion a Kernel
	_, err = client.DoRequest(cpuConfig.PortKernel, cpuConfig.IpKernel, "POST", "kernel/cpus", body)

	if err != nil {
		slog.Error(fmt.Sprintf("error: %v", err))
		panic(err)
	}
}
