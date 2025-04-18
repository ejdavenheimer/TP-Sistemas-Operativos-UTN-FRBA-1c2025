package models

type Config struct {
	IpKernel   string `json:"ip_kernel"`
	PortKernel int    `json:"port_kernel"`
	IpIo       string `json:"ip_io"`
	PortIo     int    `json:"port_io"`
	LogLevel   string `json:"log_level"`
}

var IoConfig *Config

type Device struct {
	Name string
	Ip   string
	Port int
}
