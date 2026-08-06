package bootstrap

import "time"

type ServerConfig struct {
	ServiceName     string        `json:"service_name"`
	Port            string        `json:"port"`
	MetricsPort     string        `json:"metrics_port"`
	IsDebug         bool          `json:"is_debug"`
	ShutDownTimeout time.Duration `json:"shutdown_timeout"`
}

func DefaultConfig(serviceName string, port string) ServerConfig {
	return ServerConfig{
		ServiceName:     serviceName,
		Port:            port,
		MetricsPort:     "9090",
		IsDebug:         false,
		ShutDownTimeout: 10 * time.Second,
	}
}
