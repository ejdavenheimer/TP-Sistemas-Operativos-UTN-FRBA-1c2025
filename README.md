# Enunciado

[The Rise of Gopher](https://docs.google.com/document/d/1zoFRoBn9QAfYSr0tITsL3PD6DtPzO2sq9AtvE8NGrkc/edit?tab=t.0)

## Pruebas

[Documento de prueba](https://docs.google.com/document/d/13XPliZvUBtYjaRfuVUGHWbYX8LBs8s3TDdaDa9MFr_I/edit?tab=t.0) |
[Repositorio de prueba](https://github.com/sisoputnfrba/revenge-of-the-cth-pruebas)

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
