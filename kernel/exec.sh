#!/bin/bash

# Colores
RED='\033[0;31m'
NC='\033[0m' # No Color

# Validar que se pasaron al menos 2 parámetros: nombre y puerto
if [ $# -lt 2 ]; then
    echo -e "${RED}Error: Debes proporcionar un nombre de dispositivo y un puerto como parámetros.${NC}"
    echo "Uso: ./exec.sh <nombre> <puerto> [config_file]"
    exit 1
fi

# Parámetros obligatorios
PATH="$1"
SIZE="$2"

# Config por defecto u opcional
DEFAULT_CONFIG="./configs/kernel.json"
CONFIG_FILE="${3:-$DEFAULT_CONFIG}"

# Verificar si el archivo de configuración existe
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}Error: El archivo de configuración '$CONFIG_FILE' no existe.${NC}"
    exit 1
fi

# Compilar el archivo Go y capturar errores
go build -o kernel ./kernel.go 2> build_errors.log

# Verificar si la compilación fue exitosa
if [ $? -eq 0 ]; then
    echo "Compilación exitosa."
    echo "Ejecutando con:"
    echo "Path: $PATH"
    echo "Tamaño de memoria: $SIZE"
    echo "Config: $CONFIG_FILE"

    ./kernel "$PATH" "$SIZE" "$CONFIG_FILE" 
else
    echo -e "${RED}Error en la compilación. Revisa los detalles a continuación:${NC}"
    cat build_errors.log
fi
