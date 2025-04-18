package services

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/memoria/models"
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

// TODO: esto no es necesario, esta a modo de prueba
func SetupInstructions() {
	models.Instructions.Add("IO 25000")
	models.Instructions.Add("NOOP")
	models.Instructions.Add("WRITE 0 EJEMPLO_DE_ENUNCIADO")
	models.Instructions.Add("READ 0 20")
	models.Instructions.Add("GOTO 0")
	models.Instructions.Add("INIT_PROC proceso1 256")
	models.Instructions.Add("DUMP_MEMORY")
	models.Instructions.Add("EXIT")
}

func GetInstructions() list.ArrayList[string] {
	SetupInstructions() //TODO: las instrucciones van a estar en un archivo
	return models.Instructions
}

func GetIoInstruction() string {
	models.Instructions = GetInstructions()
	value, _ := models.Instructions.Get(0)
	return value
}
