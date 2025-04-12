package handlers

import (
	"github.com/sisoputnfrba/tp-golang/utils/web/server"
	"net/http"
)

// HandshakeHandler se usa para chequear la conexión al servidor
//
// Parámetros:
//   - message: el mensaje que querés devolver en la respuesta
//
// Ejemplo:
//
//	func main() {
//		http.HandleFunc("/e", handlers.HandshakeHandler("Mensaje de ejemplo"))
//
//		err := server.InitServer(8001)
//		if err != nil {
//			slog.Error("init server error: ", err)
//		}
//	}
func HandshakeHandler(message string) func(http.ResponseWriter, *http.Request) {
	return func(writer http.ResponseWriter, request *http.Request) {
		server.SendJsonResponse(writer, message)
	}
}
