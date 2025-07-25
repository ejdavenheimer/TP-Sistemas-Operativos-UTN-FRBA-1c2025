package client

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// DoRequest es una función genérica para realizar peticiones HTTP (GET, POST, PUT, DELETE, etc.) desde un cliente.
// Retorna la respuesta del servidor. En caso que se produzca un error va a retornar el error que se produjo.
//
// Parámetros:
//   - port: el puerto al que se hará la petición
//   - ip: la IP o dominio del servidor
//   - metodo: metodo HTTP
//   - query: parte final de la URL
//   - bodies ...[]byte: (opcional) body del request (usado por ejemplo en un POST/PUT), puede pasarse vacío.
//
// Ejemplo:
//
//	func main() {
//		query := fmt.Sprintf("example?name=%s", message)
//		response, err := client.DoRequest(8080, "127.0.0.1", "GET", query, nil)
//
//		if err != nil {
//			slog.Error(fmt.Sprintf("Ocurrió un error: %v", err))
//			return
//		}
//
//		responseBody, _ := io.ReadAll(response.Body)
//		fmt.Printf("Response: %s", string(responseBody))
//	}
func DoRequest(port int, ip string, metodo string, query string, bodies ...[]byte) (*http.Response, error) {
	// Se declara un nuevo cliente
	cliente := &http.Client{}

	// Se declara la url a utilizar (depende de una ip y un puerto).
	url := fmt.Sprintf("http://%s:%d/%s", ip, port, query)

	body := ifBody(bodies...)

	// Se crea una request donde se "efectúa" el metodo (PUT / DELETE / GET / POST) hacia url, enviando el Body si lo hay
	req, err := http.NewRequest(metodo, url, body)

	// Error Handler de la construcción de la request
	if err != nil {
		slog.Error(fmt.Sprintf("error creando request a ip: %s puerto: %d", ip, port))
		return nil, err
	}

	// Se establecen los headers
	req.Header.Set("Content-Type", "application/json")

	// Se envía el request al servidor
	respuesta, err := cliente.Do(req)

	// Error handler de la request
	if err != nil {
		errorMsg := fmt.Errorf("error enviando request a ip: %s puerto: %d - %v", ip, port, err)
		slog.Error(errorMsg.Error())
		return nil, err
	}

	// **INICIO DE LA CORRECCIÓN**
	// Verificar el código de estado de la respuesta del servidor a nuestra request (de no ser OK)
	if respuesta.StatusCode != http.StatusOK {
		// Creamos un error explícito para que el código que llama a esta función sepa que algo falló.
		errorMsg := fmt.Errorf("Status Error: %d %s", respuesta.StatusCode, http.StatusText(respuesta.StatusCode))
		slog.Error(errorMsg.Error())
		// Devolvemos la respuesta (puede contener un cuerpo con más detalles del error) y el nuevo error.
		return respuesta, errorMsg
	}
	// **FIN DE LA CORRECCIÓN**

	return respuesta, nil // Devolvemos nil como error solo si todo fue exitoso.
}

func ifBody(bodies ...[]byte) io.Reader {
	if len(bodies) == 0 {
		return nil
	}
	return bytes.NewBuffer(bodies[0])
}
