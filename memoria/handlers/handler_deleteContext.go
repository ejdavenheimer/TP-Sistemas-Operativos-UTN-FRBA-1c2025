package handlers

import "net/http"

func DeleteContextHandler(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(http.StatusOK)

	//TODO: Más adelante, debe enviar el PCB a una función que realmente se encargue de borrar el contexto
}
