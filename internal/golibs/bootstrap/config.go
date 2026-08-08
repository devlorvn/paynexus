package bootstrap

import "time"

type ServerConfig struct {
	ServiceName     string        `json:"service_name"`
	Port            string        `json:"port"`
	MetricsPort     string        `json:"metrics_port"`
	IsDebug         bool          `json:"is_debug"`
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

func DefaultConfig(serviceName string, port string) ServerConfig {
	return ServerConfig{
		ServiceName:     serviceName,
		Port:            port,
		MetricsPort:     "9090",
		IsDebug:         false,
		ShutdownTimeout: 10 * time.Second,
	}
}
