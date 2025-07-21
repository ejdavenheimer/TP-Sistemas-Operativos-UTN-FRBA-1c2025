# Enunciado

[The Rise of Gopher](https://docs.google.com/document/d/1zoFRoBn9QAfYSr0tITsL3PD6DtPzO2sq9AtvE8NGrkc/edit?tab=t.0)

## Pruebas

[Documento de prueba](https://docs.google.com/document/d/13XPliZvUBtYjaRfuVUGHWbYX8LBs8s3TDdaDa9MFr_I/edit?tab=t.0) |
[Repositorio de prueba](https://github.com/sisoputnfrba/revenge-of-the-cth-pruebas)

## Deploy

### Guía Rápida

```
Levantar la VM, copiar la IP en Putty. En caso que no muestra la ip ejecutar
> ifconfig

Clonar el repo, se debe usar el Personal Access Token
> git clone https://github.com/sisoputnfrba/tp-2025-1c-Los-magiOS.git

Una vez que tenemos el tp clonado, hacer un ls y chequear si existe la carpeta de scripts. 
Si no existen los scripts de pruebas, se ejecutar el siguiente comando: 
> ./create_scripts.sh

Luego se debe modificar las ips de los archivos de configs.

Para modificar los archivos configuración, posicionarse en el tp

> cd tp-2025-1c-Los-magiOS
> go build update_config.go
> ./update_config.exe ip_memory 127.0.0.1 ip_cpu 127.0.0.1 ip_kernel 127.0.0.1 ip_io 127.0.0.1

Para chequear si se modificaron los archivos (posicionarse en el módulo que corresponda)
> cd kernel && cat ./configs/kernel.json

Levantar los módulos y rezar a superman o en el que crean! 

> make build 

Si falta alguno de los módulos hacer:
> make clean
> make build

> ./bin/memoria ./memoria/configs/memoria.json
> ./bin/kernel PLANI_LYM_IO 256 ./kernel/configs/kernel.json
> ./bin/cpu 1 8004 ./cpu/configs/cpu.json
> ./bin/io disco 8005
```

### Guía Rápida de Vim
1. vim ejemplo.txt
2. Tocar la tecla i para poder editar el archivo.
3. Editar el archivo, moverse con las teclas.
4. Una vez que terminas de editar, tocar Esc
5. Para guardar :wq
   6. Si no funca usar :wq!

## Compilación del proyecto
Para compilar los binarios de los módulos, utilizamos un Makefile con los siguientes comandos:

### Construir todos los módulos
```
make buils
```

### Construir un módulo individual
```
make memoria
make kernel
make io
make cpu
```

### Limpiar binarios compilados
```
make clean
```
Esto eliminará todos los ejecutables dentro del directorio bin/.

## Ejecución de los módulos
Los binarios compilados se encuentran en el directorio bin/. Para ejecutar un módulo:
```
./bin/memoria
./bin/kernel
./bin/io [nombre_interface]
./bin/cpu [identificador_cpu]
```

## Checkpoint

Para cada checkpoint de control obligatorio, se debe crear un tag en el
repositorio con el siguiente formato:

```
checkpoint-{número}
```

Donde `{número}` es el número del checkpoint.

Para crear un tag y subirlo al repositorio, podemos utilizar los siguientes
comandos:

```bash
git tag -a checkpoint-{número} -m "Checkpoint {número}"
git push origin checkpoint-{número}
```

Asegúrense de que el código compila y cumple con los requisitos del checkpoint
antes de subir el tag.
