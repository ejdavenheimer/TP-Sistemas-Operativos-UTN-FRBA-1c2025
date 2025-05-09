package models

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

/* ---------- Colas de estados ----------> */

// Colas del Planificador de Largo Plazo
var QueueNew = &list.ArrayList[PCB]{}
var QueueExit = &list.ArrayList[PCB]{}

// Colas del Planificador de Corto Plazo
var QueueReady = &list.ArrayList[PCB]{}
var QueueBlocked = &list.ArrayList[PCB]{}

//A futuro... var QueueNewPMCP = orderList(&list.ArrayList[PCB]{})

// Colas del Planificador de Mediano Plazo
var QueueSuspReady = &list.ArrayList[PCB]{}
var QueueSuspBlocked = &list.ArrayList[PCB]{}
