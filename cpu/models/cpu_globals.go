package models

import "errors"

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

type TLBEntry struct {
	PID         uint
	PageNumber  int
	FrameNumber int
	LastUsed    int64 //contador para LRU
}

type MemoryConfig struct {
	PageSize       int `json:"page_size"`
	EntriesPerPage int `json:"entries_per_page"`
	NumberOfLevels int `json:"number_of_levels"`
}

var MemConfig *MemoryConfig

type ExecuteInstructionRequest struct {
	Pid    uint
	Values []string
}

// TODO: agregar los registros que consideremos necesarios
type Registers struct {
	PC uint
}

var CpuRegisters Registers

var InterruptControl = InterruptData{
	InterruptPending: false,
	PID:              -1,
}

type InterruptData struct {
	InterruptPending bool
	PID              int
}

type CpuN struct {
	Port         int
	Ip           string
	Id           int
	IsFree       bool
	PIDExecuting uint
	PIDRafaga    float32
}

// Para la instrucción READ
type MemoryReadRequest struct {
	Pid             uint `json:"pid"`
	PhysicalAddress int  `json:"physicalAddress"`
	Size            int  `json:"size"`
}

type WriteRequest struct {
	PID             uint
	PhysicalAddress int
	Data            []byte
}

type ReadRequest struct {
	Pid             uint `json:"pid"`
	PhysicalAddress int  `json:"physicalAddress"`
	Size            int  `json:"size"`
}

// DEFINICION DE ERRORES
var ErrInvalidInstruction = errors.New("invalid instruction")
var ErrInvalidAddress = errors.New("invalid address")
