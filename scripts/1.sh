#!/bin/bash

# Códigos de color
ROJO='\033[0;31m'
VERDE='\033[0;32m'
AMARILLO='\033[0;33m'
AZUL='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

test_obtener_cpus_conectadas() {
    echo -e "${VERDE}Obtener CPUs conectadas${NC}"
    curl --location --request GET "http://localhost:8001/kernel/cpus-conectadas" \
        --header 'Content-Type: application/json'
}

test_obtener_dispositivos_conectado() {
    echo -e "${VERDE}Obtener dispositivos conectados${NC}"
    curl --location --request GET "http://localhost:8001/kernel/dispositivos-conectados" \
        --header 'Content-Type: application/json'
}

test_obtener_instruccion_io() {
    echo -e "${VERDE}Obtener instrucción IO${NC}"
    curl --location --request GET "http://localhost:8002/memoria/instrucciones?pid=0&pathName=example1" \
        --header 'Content-Type: application/json'
}

test_ejecutar_cpu_exec() {
    echo -e "${VERDE}Ejecutando instrucción IO desde CPU${NC}"
    read -p "$(echo -e ${AMARILLO}Pid:${NC} )" pid
    read -p "$(echo -e ${AMARILLO}PC:${NC} )" pc
    echo -e "${VERDE}El Pid ingresado es:${NC} $pid"
    echo -e "${VERDE}El PC ingresado es:${NC} $pc"
    curl --location --request POST http://localhost:8004/cpu/exec \
        --header 'Content-Type: application/json' \
        --data "{\"pid\": $pid, \"pc\": $pc}"
}
# {
#     "pid": 1,
#     "pc": 0,
# }

test_ejecutar_cpu_process() {
    echo -e "${VERDE}Ejecutando instrucción IO desde CPU${NC}"
    read -p "$(echo -e ${AMARILLO}Pid:${NC} )" pid
    read -p "$(echo -e ${AMARILLO}PC:${NC} )" pc
    read -p "$(echo -e ${AMARILLO}Path:${NC} )" pathName
    echo -e "${VERDE}El Pid ingresado es:${NC} $pid"
    echo -e "${VERDE}El PC ingresado es:${NC} $pc"
    echo -e "${VERDE}El Path ingresado es:${NC} $pathName"
    curl --location --request POST http://localhost:8004/cpu/process \
        --header 'Content-Type: application/json' \
        --data "{\"pid\": $pid, \"pc\": $pc, \"pathName\": \"$pathName\"}"
}
# {
#     "pid": 1,
#     "pc": 0,
#     "pathName": "example1"
# }

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
# {
#     "type": "impresora10",
#     "values": ["IO", "impresora1", "25000"]
# }
test_ejecutar_syscall_init_proc() {
    echo -e "${VERDE}Ejecutando syscall INIT_PROC${NC}"

    # Solicitar datos al usuario
    read -p "$(echo -e ${AMARILLO}Pseudocode File:${NC} )" pseudocodeFile
    read -p "$(echo -e "${AMARILLO}Process Size (en bytes):${NC} ")" processSize
    read -p "$(echo -e ${AMARILLO}Parent PID:${NC} )" parentPID

    # Mostrar los datos ingresados
    echo -e "${VERDE}El archivo pseudocódigo ingresado es:${NC} $pseudocodeFile"
    echo -e "${VERDE}El tamaño del proceso ingresado es:${NC} $processSize"
    echo -e "${VERDE}El Parent PID ingresado es:${NC} $parentPID"
    
    # Realizar la llamada al servidor para ejecutar la syscall INIT_PROC
    response=$(curl --silent --location --request POST http://localhost:8001/kernel/syscall \
        --header 'Content-Type: application/json' \
        --data "{\"type\": \"INIT_PROC\", \"values\": [\"$pseudocodeFile\", \"$processSize\"], \"pid\": $parentPID}")

    # Mostrar la respuesta del servidor
    echo -e "${VERDE}Respuesta del servidor:${NC} $response"
}

test_ejecutar_dump_memory() {
    echo -e "${VERDE}Ejecutando syscall DUMP_MEMORY${NC}"
    read -p "$(echo -e ${AMARILLO}Pid:${NC} )" pid
    read -p "$(echo -e ${AMARILLO}Size:${NC} )" size
    echo -e "${VERDE}El Pid ingresado es:${NC} $pid"
    echo -e "${VERDE}El PC ingresado es:${NC} $size"
    curl --location --request POST http://localhost:8002/memoria/dump-memory \
        --header 'Content-Type: application/json' \
        --data "{\"pid\": $pid, \"size\": $size}"
}

test_ejecutar_proceso_desde_kernel() {
    echo -e "${VERDE}Ejecutando instrucción desde Kernel${NC}"
    read -p "$(echo -e ${AMARILLO}Pid:${NC} )" pid
    read -p "$(echo -e ${AMARILLO}PC:${NC} )" pc
    read -p "$(echo -e ${AMARILLO}Path:${NC} )" pathName
    echo -e "${VERDE}El Pid ingresado es:${NC} $pid"
    echo -e "${VERDE}El PC ingresado es:${NC} $pc"
    echo -e "${VERDE}El Path ingresado es:${NC} $pathName"
    curl --location --request POST http://localhost:8001/kernel/ejecutarProceso \
        --header 'Content-Type: application/json' \
        --data "{\"pid\": $pid, \"pc\": $pc, \"pathName\": \"$pathName\"}"
}

test_finalizar_proceso_kernel() {
    echo -e "${VERDE}Finalizando un proceso desde Kernel${NC}"

    read -p "$(echo -e ${AMARILLO}Pid:${NC} )" pid
    read -p "$(echo -e ${AMARILLO}PC:${NC} )" pc
    read -p "$(echo -e ${AMARILLO}Path:${NC} )" pathName

    # Simulamos valores de métricas (esto depende del modelo que tengas en memoria)
    echo -e "${VERDE}Simulando métricas en el PCB...${NC}"
    metrics='{
        "NEW": 2,
        "READY": 3
    }'
    times='{
        "NEW": 150,
        "READY": 230
    }'

    curl --location --request POST http://localhost:8001/kernel/finalizarProceso \
        --header 'Content-Type: application/json' \
        --data "{
            \"pid\": $pid,
            \"pc\": $pc,
            \"pathName\": \"$pathName\",
            \"ME\": {\"NEW\": 2, \"READY\": 3},
            \"MT\": {\"NEW\": 150, \"READY\": 230}
        }"
}

while true; do
    echo -e "${AMARILLO}1.${NC} Obtener intrucción IO"
    echo -e "${AMARILLO}2.${NC} Ejecutando instrucción IO desde CPU (EXEC)"
    echo -e "${AMARILLO}3.${NC} Ejecutando syscall IO"
    echo -e "${AMARILLO}4.${NC} Obtener dispositivos conectados"
    echo -e "${AMARILLO}5.${NC} Obtener CPUs conectadas"
    echo -e "${AMARILLO}6.${NC} Ejecutando instrucción IO desde CPU (PROCESS)"
    echo -e "${AMARILLO}7.${NC} Ejecutando instrucción INIT_PROC desde CPU"
    echo -e "${AMARILLO}8.${NC} Ejecutar proceso desde Kernel"
    echo -e "${AMARILLO}9.${NC} Finalizar proceso desde Kernel"
    echo -e "${AMARULLI}10.${NC} Ejecutando instrucción DUMP_MEMORY"
    echo -e "${ROJO}s.${NC} Salir"
    echo
    read -p "$(echo -e ${AMARILLO}Opción:${NC} )" opcion

    case $opcion in
        1) test_obtener_instruccion_io ;;
        2) test_ejecutar_cpu_exec ;;
        3) test_ejercutar_syscall_io ;;
        4) test_obtener_dispositivos_conectado ;;
        5) test_obtener_cpus_conectadas ;;
        6) test_ejecutar_cpu_process ;;
        7) test_ejecutar_syscall_init_proc ;;
        8) test_ejecutar_proceso_desde_kernel ;;
        9) test_finalizar_proceso_kernel ;;
        10) test_ejecutar_dump_memory ;;
        s) echo -e "${ROJO}Saliendo...${NC}"; break ;;
        *) echo -e "${ROJO}Opción no válida${NC}" ;;
    esac
    echo
done