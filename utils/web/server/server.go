package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// InitServer inicializa el servidor, en caso de no poder levantarlo retorna un error
//
// Parámetros:
//   - port: puerto donde se iniciará el servidor
//
// Ejemplo:
//
//	func main() {
//		err := server.InitServer(globals.ConfigMemoria.Port)
//		if err != nil {
//			fmt.Errorf("error initializing server: %v", err)
//			panic(err)
//		}
//	}
func InitServer(port int) error {
	addr := ":" + strconv.Itoa(port)

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println("Error al escuchar en el puerto " + addr)
		fmt.Println(err)
	}
	return err
}

// SendJsonResponse retorna la respues del servidor en formato JSON
//
// Parámetros:
//   - writer: el http.ResponseWriter con el que se escribe la respuesta HTTP
//   - data: cualquier estructura de datos que querés enviar al cliente, se convierte automáticamente a JSON.
//
// Ejemplo:
//
//	func HandshakeHandler(message string) func(http.ResponseWriter, *http.Request) {
//		return func(writer http.ResponseWriter, request *http.Request) {
//			server.SendJsonResponse(writer, message)
//		}
//	}
func SendJsonResponse(writer http.ResponseWriter, data interface{}) {
	response, err := json.Marshal(data)
	if err != nil {
		http.Error(writer, "Error al convertir datos a JSON", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	writer.Write(response)
}
