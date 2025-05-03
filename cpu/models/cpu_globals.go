package models

type Config struct {
	PortCpu          int    `json:"port_cpu"`
	IpCpu            string `json:"ip_cpu"`
	IpMemory         string `json:"ip_memory"`
	PortMemory       int    `json:"port_memory"`
	IpKernel         string `json:"ip_kernel"`
	PortKernel       int    `json:"port_kernel"`
	TlbEntries       int    `json:"tlb_entries"`
	TlbReplacement   string `json:"tlb_replacement"`
	CacheEntries     int    `json:"cache_entries"`
	CacheReplacement string `json:"cache_replacement"`
	CacheDelay       int    `json:"cache_delay"`
	LogLevel         string `json:"log_level"`
}

var CpuConfig *Config

type ExecuteInstructionRequest struct {
	Pid    int
	Values []string
}

// TODO: agregar los registros que consideremos necesarios
type Registers struct {
	PC uint
}

var CpuRegisters Registers
var InterruptPending bool = false

type CpuN struct {
	Port   int
	Ip     string
	Id     int
	IsFree bool
}
