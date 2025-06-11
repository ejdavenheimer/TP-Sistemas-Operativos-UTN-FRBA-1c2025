package models

type Config struct {
	PortMemory     int    `json:"port_memory"`
	IpMemory       string `json:"ip_memory"`
	MemorySize     int    `json:"memory_size"`
	PageSize       int    `json:"page_size"`
	EntriesPerPage int    `json:"entries_per_page"`
	NumberOfLevels int    `json:"number_of_levels"`
	MemoryDelay    int    `json:"memory_delay"`
	SwapFilePath   string `json:"swapfile_path"`
	SwapDelay      int    `json:"swap_delay"`
	LogLevel       string `json:"log_level"`
	DumpPath       string `json:"dump_path"`
	ScriptsPath    string `json:"scripts_path"`
}

type InstructionsResponse struct {
	Instruction map[uint][]string
}

type InstructionResponse struct {
	Instruction string
	IsLast      bool
}

type MemoryRequest struct {
	PID  uint   `json:"pid"`
	Size int    `json:"size"`
	Path string `json:"path"`
}

type InstructionRequest struct {
	Pid      int
	PC       int
	PathName string
}

//Para READ
type ReadRequest struct {
	Pid             uint   `json:"pid"`
    PhysicalAddress int    `json:"physicalAddress"`
    Size            int    `json:"size"`
}

type Metrics struct {
	PageTableAccesses  int
	InstructionFetches int
	SwapsOut           int
	SwapsIn            int
	Reads              int
	Writes             int
}

type Process struct {
	Pid         uint
	Size        int
	Pages       int // cantidad de páginas que va a usar
	Metrics     *Metrics // metricas del proceso
}
// Maps para procesos y métricas
var ProcessMetrics = make(map[uint]*Metrics)
var ProcessTable = make(map[uint]*Process)

type DumpMemoryRequest struct {
	Pid  uint
	Size int
}

type DumpMemoryResponse struct {
	Result string
}

var MemoryConfig *Config
var InstructionsMap map[uint][]string

type MemoryFrame struct {
	StartAddr int  // Dirección inicial en UserMemory
	IsFree    bool // Si el frame está disponible
}

var FrameTable []MemoryFrame

type ProcessFrames struct {
	PID    int
	Frames []int
}

var ProcessFramesTable = make(map[int]ProcessFrames)

type SwapEntry struct {
	Offset int64 // posición inicial en archivo
	Size   int64 // tamaño en bytes del bloque de frames
}

var ProcessSwapTable = make(map[int]SwapEntry)

type PIDRequest struct {
	PID int `json:"pid"`
}

//Para buscar frames ocupados
type FrameUsage struct {
	Frame int  `json:"frame"`
	Pid   uint `json:"pid"`
}

type FramesInUseResponse struct {
	Frames []FrameUsage `json:"frames"`
}

//Espacio contiguo en memoria principal (usuario)
var UserMemory []byte

//Tabla jerarquica multinivel
type PageEntry struct {
	Frame    int  // Número de marco físico en UserMemory
	Presence bool // Presente en memoria física
	Use      bool // Bit de uso para reemplazo
	Modified bool // Bit de modificación
}

// PageTableLevel representa un nodo de la tabla de páginas multinivel.
// Puede ser un nodo interno con referecia a niveles inferiores o el nodo 
// que contiene una entrada (Entry) con la información del marco físico.
type PageTableLevel struct {
	IsLeaf   bool // cuando es true es el nodo que contiene la entrada
	SubTables map[int]*PageTableLevel // Si no es el último nodo va a apuntar a un nodo inferior
	Entry    *PageEntry             // Si es el último nodo, contiene la entrada que apunta al marco físico
}

var PageTables = make(map[uint]*PageTableLevel)
var FreeFrames []bool // true si el frame está libre, false si está ocupado
