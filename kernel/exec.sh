#!/bin/bash

# Colores
RED='\033[0;31m'
NC='\033[0m' # No Color

# Archivo de configuración por defecto
DEFAULT_CONFIG="./configs/kernel.json"

# Usar el archivo pasado por parámetro, o el default si no se especifica
CONFIG_FILE="${1:-$DEFAULT_CONFIG}"

# Verificar si el archivo existe
if [ ! -f "$CONFIG_FILE" ]; then
    echo -e "${RED}Error: El archivo de configuración '$CONFIG_FILE' no existe.${NC}"
    exit 1
fi

# Compilar el archivo Go y capturar errores
go build ./kernel.go 2> build_errors.log

# Verificar si la compilación fue exitosa
if [ $? -eq 0 ]; then
    echo "Compilación exitosa. Ejecutando el programa con config: $CONFIG_FILE"
    ./kernel "$CONFIG_FILE"
else
    echo -e "${RED}Error en la compilación. Revisa los detalles a continuación:${NC}"
    cat build_errors.log
fi