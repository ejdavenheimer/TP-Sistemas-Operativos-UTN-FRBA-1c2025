#!/bin/bash

# Códigos de color
ROJO='\033[0;31m'
VERDE='\033[0;32m'
AMARILLO='\033[0;33m'
AZUL='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

test_iniciar_proceso() {
    echo -e "${VERDE}Iniciando proceso${NC}"
    read -p "$(echo -e ${AMARILLO}PID:${NC} )" pid
    echo -e "${VERDE}El PID ingresado es:${NC} $pid"
    curl --location --request POST http://localhost:8005/kernel/proceso \
            --header 'Content-Type: application/json' \
            --data "{\"pid\": $pid}"
}

test_de_ejecutar_cpu() {
    echo -e "${VERDE}Ejecutando CPU${NC}"
    curl --location --request POST http://localhost:8004/cpu/exec \
            --header 'Content-Type: application/json' 
}

iniciar_proceso_memoria() {
    echo -e "${VERDE}Ejecutando CPU${NC}"
    read -p "$(echo -e ${AMARILLO}PID:${NC} )" pid
    read -p "$(echo -e ${AMARILLO}Path:${NC} )" path
    echo -e "${VERDE}El PID ingresado es:${NC} $pid"
    echo -e "${VERDE}El PATH ingresado es:${NC} $path"
    curl --location --request PUT http://localhost:8002/memoria/proceso \
            --header 'Content-Type: application/json' \
            --data "{\"pid\": $pid, \"path\": \"$path\"}"
}

while true; do
    echo -e "${AMARILLO}1.${NC} Prueba para iniciar proceso"
    echo -e "${AMARILLO}2.${NC} Prueba para ejecutar CPU"
    echo -e "${AMARILLO}3.${NC} Iniciar proceso de memoria"
    echo -e "${ROJO}s.${NC} Salir"
    echo 
    read -p "$(echo -e ${AMARILLO}Opción:${NC} )" opcion

    case $opcion in
        1) test_iniciar_proceso;;
        2) test_de_ejecutar_cpu;;
        3) iniciar_proceso_memoria;;
        s) echo -e "${ROJO}Saliendo...${NC}"; break ;;
        *) echo -e "${ROJO}Opción no válida${NC}" ;;
    esac
    echo
done