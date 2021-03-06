package config

import (
	"time"
)

// Version defines the version.
var Version string

// Config defines the configuration.
type Config struct {
	General struct {
		LogLevel int `mapstructure:"log_level"`
	}

	Redis struct {
		Servers    []string `mapstructure:"servers"`
		Cluster    bool     `mapstructure:"cluster"`
		MasterName string   `mapstructure:"master_name"`
		PoolSize   int      `mapstructure:"pool_size"`
		Password   string   `mapstructure:"password"`
		Database   int      `mapstructure:"database"`
	} `mapstructure:"redis"`

	NetworkServer struct {
		DeviceSessionTTL time.Duration `mapstructure:"device_session_ttl"`
		PostgreSQL       struct {
			DSN                string `mapstructure:"dsn"`
			MaxOpenConnections int    `mapstructure:"max_open_connections"`
			MaxIdleConnections int    `mapstructure:"max_idle_connections"`
		} `mapstructure:"postgre"`
	} `mapstructure:"ns"`

	ApplicationServer struct {
		PostgreSQL struct {
			DSN                string `mapstructure:"dsn"`
			MaxOpenConnections int    `mapstructure:"max_open_connections"`
			MaxIdleConnections int    `mapstructure:"max_idle_connections"`
		} `mapstructure:"postgre"`
	} `mapstructure:"as"`

	// Prometheus struct {
	// 	Bind string `mapstructure:"bind"`
	// } `mapstructure:"prometheus"`
}

// C holds the global configuration.
var C Config
