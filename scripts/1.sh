#!/bin/bash

# Códigos de color
ROJO='\033[0;31m'
VERDE='\033[0;32m'
AMARILLO='\033[0;33m'
AZUL='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

test_obtener_instruccion_io() {
    echo -e "${VERDE}Obtener instrucción IO${NC}"
    curl --location --request GET http://localhost:8002/memoria/instrucciones \
        --header 'Content-Type: application/json'
}

test_ejecutar_cpu() {
    echo -e "${VERDE}Ejecutando instrucción IO desde CPU${NC}"
    curl --location --request POST http://localhost:8004/cpu/exec \
        --header 'Content-Type: application/json'
}

test_ejercutar_syscall_io() {
    echo -e "${VERDE}Ejecutando syscall IO${NC}"
    read -p "$(echo -e ${AMARILLO}Type:${NC} )" type
    read -p "$(echo -e "${AMARILLO}Values (separados por coma):${NC} ")" values
    formatted_values=$(echo $values | sed 's/[^,][^,]*/"&"/g')
    echo -e "${VERDE}El Type ingresado es:${NC} $type"
    echo -e "${VERDE}El Values ingresado es:${NC} $formatted_values"
    curl --location --request POST http://localhost:8001/kernel/syscall \
        --header 'Content-Type: application/json' \
        --data "{\"type\": \"$type\", \"values\": [$formatted_values]}"
}

while true; do
    echo -e "${AMARILLO}1.${NC} Obtener intrucción IO"
    echo -e "${AMARILLO}2.${NC} Ejecutando instrucción IO desde CPU"
    echo -e "${AMARILLO}3.${NC} Ejecutando syscall IO"
    echo -e "${ROJO}s.${NC} Salir"
    echo
    read -p "$(echo -e ${AMARILLO}Opción:${NC} )" opcion

    case $opcion in
        1) test_obtener_instruccion_io ;;
        2) test_ejecutar_cpu ;;
        3) test_ejercutar_syscall_io ;;
        s) echo -e "${ROJO}Saliendo...${NC}"; break ;;
        *) echo -e "${ROJO}Opción no válida${NC}" ;;
    esac
    echo
done