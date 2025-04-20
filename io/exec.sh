#!/bin/bash

# Colores
RED='\033[0;31m'
NC='\033[0m' # No Color

# Validar que se pasaron al menos 2 parámetros
if [ $# -lt 1 ]; then
    echo -e "${RED}Error: Debes proporcionar un nombre de dispositivo como parámetros.${NC}"
    echo "Uso: ./exec.sh <nombre> [config_file]"
    exit 1
fi

# Parámetros obligatorios
NOMBRE_DISPOSITIVO="$1"

# Config por defecto u opcional
DEFAULT_CONFIG="./configs/io.json"
CONFIG_FILE="${3:-$DEFAULT_CONFIG}"

# Verificar si el archivo de configuración existe
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}Error: El archivo de configuración '$CONFIG_FILE' no existe.${NC}"
    exit 1
fi

# Compilar el archivo Go y capturar errores
go build ./io.go 2> build_errors.log

# Verificar si la compilación fue exitosa
if [ $? -eq 0 ]; then
    echo "Compilación exitosa."
    echo "Ejecutando con:"
    echo "Nombre: $NOMBRE_DISPOSITIVO"
    echo "Config: $CONFIG_FILE"

    ./io "$CONFIG_FILE" "$NOMBRE_DISPOSITIVO"
else
    echo -e "${RED}Error en la compilación. Revisa los detalles a continuación:${NC}"
    cat build_errors.log
fi