#!/bin/bash

REPO_URL="https://github.com/sisoputnfrba/revenge-of-the-cth-pruebas.git"
DESTINO="/home/utnso/scripts"
TEMP_DIR="repo_tmp"

# Clonar el repositorio en una carpeta temporal
git clone "$REPO_URL" "$TEMP_DIR"

# Crear el directorio de destino si no existe
mkdir -p "$DESTINO"

# Mover todo excepto archivos *.md y el directorio .git
shopt -s dotglob
for file in "$TEMP_DIR"/*; do
    if [[ $(basename "$file") != *.md && $(basename "$file") != ".git" ]]; then
        mv "$file" "$DESTINO"
    fi
done

# Eliminar la carpeta temporal
rm -rf "$TEMP_DIR"