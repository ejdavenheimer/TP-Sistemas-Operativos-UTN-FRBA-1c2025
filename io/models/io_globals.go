package models

import "sync"

type Config struct {
	IpKernel   string `json:"ip_kernel"`
	PortKernel int    `json:"port_kernel"`
	IpIo       string `json:"ip_io"`
	PortIo     int    `json:"port_io"`
	LogLevel   string `json:"log_level"`
}

var IoConfig *Config
var DeviceMutex sync.Mutex

type Device struct {
	Name   string
	Ip     string
	Port   int
	IsFree bool
}

type DeviceResponse struct {
	Pid    int
	Reason string
	Port   int
}

type IORequest struct {
	PID    int `json:"pid"`
	TimeIO int `json:"tiempo"`
}
