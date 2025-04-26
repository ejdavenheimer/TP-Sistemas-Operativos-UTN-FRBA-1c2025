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
NOMBRE_DISPOSITIVO="$1"
PORT="$2"

# Config por defecto u opcional
DEFAULT_CONFIG="./configs/io.json"
CONFIG_FILE="${3:-$DEFAULT_CONFIG}"

# Verificar si el archivo de configuración existe
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}Error: El archivo de configuración '$CONFIG_FILE' no existe.${NC}"
    exit 1
fi

# Compilar el archivo Go y capturar errores
go build -o io ./io.go 2> build_errors.log

# Verificar si la compilación fue exitosa
if [ $? -eq 0 ]; then
    echo "Compilación exitosa."
    echo "Ejecutando con:"
    echo "Nombre: $NOMBRE_DISPOSITIVO"
    echo "Puerto: $PORT"
    echo "Config: $CONFIG_FILE"

    ./io "$NOMBRE_DISPOSITIVO" "$PORT" "$CONFIG_FILE" 
else
    echo -e "${RED}Error en la compilación. Revisa los detalles a continuación:${NC}"
    cat build_errors.log
fi
