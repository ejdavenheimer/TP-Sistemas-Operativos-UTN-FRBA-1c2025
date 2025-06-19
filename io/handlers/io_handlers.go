package handlers

import (
	"encoding/json"
	ioModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/io/services"
	kernelModel "github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/kernel/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/web/server"
	"net/http"
	"bufio"
	"strings"
	"log/slog"
	"fmt"
	"net"
)

func SleepHandler() func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		ioModel.DeviceMutex.Lock()

		//--------- RECIBE ---------

		var deviceRequest kernelModel.DeviceRequest

		// Decodifica el request (codificado en formato json).
		err := json.NewDecoder(request.Body).Decode(&deviceRequest)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		//--------- EJECUTA ---------
		services.Sleep(deviceRequest.Pid, deviceRequest.SuspensionTime)

		//--------- RESPUESTA ---------

		response := ioModel.DeviceResponse{
			Pid:    deviceRequest.Pid,
			Reason: "Fin de IO",
		}

		server.SendJsonResponse(writer, response)
		ioModel.DeviceMutex.Unlock()
	}
}

//CONEXION CON EL KERNEL 
func ConectToKernel(nombre, ip string, puerto int) { 
	direccion := net.JoinHostPort(ip, fmt.Sprintf("%d", puerto))
	conn, err := net.Dial("tcp", direccion) //Establece conexion TCP con el Kernel
	if err != nil {
		slog.Error("No se pudo conectar al Kernel", "error", err)
		return
	}
	defer conn.Close()

	// comunicacion inicial
	fmt.Fprintf(conn, "%s\n", nombre)
	slog.Info("Handshake enviado al Kernel", "nombre", nombre)

	// Espera la petición
	reader := bufio.NewReader(conn)
	for {
		linea, err := reader.ReadString('\n')
		if err != nil {
			slog.Error("Error leyendo petición del Kernel", "error", err)
			break
		}
		linea= strings.TrimSpace(linea)
		slog.Info("Petición recibida", "mensaje", strings.TrimSpace(linea))

		// Analiza el tiempo de la peticion 
		var (
			pid uint
		    tiempo int
		)
			
		_, err = fmt.Sscanf(linea, "PID: %*d|TIEMPO_IO: %d", &pid, &tiempo)
		if err != nil {
			slog.Warn("Petición inválida", "detalle", linea)
			continue
		}
		services.Sleep(pid, tiempo)
	}
}
