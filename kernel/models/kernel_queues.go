package models

import (
	"github.com/sisoputnfrba/tp-2025-1c-Los-magiOS/utils/list"
)

/* ---------- Colas de estados ----------> */

// Cola del estado NEW
var QueueNew = &list.ArrayList[PCB]{}

// Cola del estado Ready
var QueueReady = &list.ArrayList[PCB]{}

//A futuro... var QueueNewPMCP = orderList(&list.ArrayList[PCB]{})

var QueueExit = &list.ArrayList[PCB]{}
