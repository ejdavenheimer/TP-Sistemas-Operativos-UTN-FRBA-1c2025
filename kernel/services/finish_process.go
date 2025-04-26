package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
)

/*
Al momento de finalizar un proceso, el Kernel deberá informar a la Memoria la
finalización del mismo y luego de recibir la confirmación por parte de la Memoria
deberá liberar su PCB asociado e intentar inicializar uno de los que estén esperando,
para ello primero se deberá verificar los procesos que estén en estado SUSP. READY
y luego en caso de que no se pueda iniciar ninguno de esos, se pasará a ver los que se
encuentren en el estado NEW de acuerdo al algoritmo definido, si los hubiere.
Luego de la finalización, se debe loguear las métricas de estado con el formato adecuado.
*/

func FinishProcess(pcb models.PCB) {
	//Conectarse con memoria y enviar PCB
	bodyRequest, err := json.Marshal(pcb)
	if err != nil {
		slog.Error(fmt.Sprintf("Error al pasar a formato json el pcb: %v", err))
		panic(err)
	}
	url := fmt.Sprintf("http://%s:%d/memoria/liberarpcb", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(bodyRequest))
	if err != nil {
		slog.Error("Error enviando el PCB a ip:%s puerto:%d", models.KernelConfig.IpMemory, models.KernelConfig.PortMemory)
	}
	defer resp.Body.Close()

	//Recibir StatusOK por parte de memoria
	if resp.StatusCode == http.StatusOK {
		slog.Info("Respuesta OK de Memoria")
	} else {
		slog.Warn(fmt.Sprintf("Respuesta NO OK de Memoria. Código: %d.", resp.StatusCode))
	}

	//Liberar PCB asociado
	models.QueueExit.Dequeue()

	//Intentar inicializar un proceso de SUSP READY sino los de NEW
	//Logear métricas
}
