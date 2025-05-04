package models

type Config struct {
	PortMemory     int    `json:"port_memory"`
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

var MemoryConfig *Config
var InstructionsMap map[uint][]string
