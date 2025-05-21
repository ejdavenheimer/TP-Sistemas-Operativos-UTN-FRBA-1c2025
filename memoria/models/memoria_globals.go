package models

type Config struct {
	PortMemory     int    `json:"port_memory"`
	IpMemory       string  `json:"ip_memory"`
	MemorySize     int    `json:"memory_size"`
	PageSize       int    `json:"page_size"`
	EntriesPerPage int    `json:"entries_per_page"`
	NumberOfLevels int    `json:"number_of_levels"`
	MemoryDelay    int    `json:"memory_delay"`
	SwapFilePath   string `json:"swap_file_path"`
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

type MemoryInstructionRequest struct {
	Pid              int
	PhysicalAddress  int
	Size             int
}

type Process struct {
	Pid         int
	BaseAddress int
	Size        int
}

type Metrics struct {
	PageTableAccesses int
	InstructionFetches int
	SwapsOut int
	SwapsIn int
	Reads int
	Writes int
}

var ProcessMetrics = make(map[uint]*Metrics)
var ProcessTable = make(map[int]Process)
var MemoryConfig *Config
var InstructionsMap map[uint][]string
var NextFreeAddress int = 0
var UserMemory []byte //Espacio contiguo de memoria de usuario
var PageTables = make(map[uint]map[int]interface{})

type WriteRequest struct {
	Address int    `json:"address"`
	Data    string `json:"data"`
}