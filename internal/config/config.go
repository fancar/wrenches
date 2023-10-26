package config

import (
	"time"

	"github.com/brocaar/lorawan/band"
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

	RedisSecond struct {
		Servers    []string `mapstructure:"servers"`
		Cluster    bool     `mapstructure:"cluster"`
		MasterName string   `mapstructure:"master_name"`
		PoolSize   int      `mapstructure:"pool_size"`
		Password   string   `mapstructure:"password"`
		Database   int      `mapstructure:"database"`
	} `mapstructure:"redis_second"`

	NetworkServer struct {
		// BandName         band.Name     `mapstructure:"band_name"`
		Band struct {
			Name                   band.Name `mapstructure:"name"`
			UplinkDwellTime400ms   bool      `mapstructure:"uplink_dwell_time_400ms"`
			DownlinkDwellTime400ms bool      `mapstructure:"downlink_dwell_time_400ms"`
			UplinkMaxEIRP          float32   `mapstructure:"uplink_max_eirp"`
			RepeaterCompatible     bool      `mapstructure:"repeater_compatible"`
		} `mapstructure:"band"`
		NetworkSettings struct {
			InstallationMargin      float64  `mapstructure:"installation_margin"`
			RXWindow                int      `mapstructure:"rx_window"`
			RX1Delay                int      `mapstructure:"rx1_delay"`
			RX1DROffset             int      `mapstructure:"rx1_dr_offset"`
			RX2DR                   int      `mapstructure:"rx2_dr"`
			RX2Frequency            int64    `mapstructure:"rx2_frequency"`
			RX2PreferOnRX1DRLt      int      `mapstructure:"rx2_prefer_on_rx1_dr_lt"`
			RX2PreferOnLinkBudget   bool     `mapstructure:"rx2_prefer_on_link_budget"`
			GatewayPreferMinMargin  float64  `mapstructure:"gateway_prefer_min_margin"`
			DownlinkTXPower         int      `mapstructure:"downlink_tx_power"`
			EnabledUplinkChannels   []int    `mapstructure:"enabled_uplink_channels"`
			DisableMACCommands      bool     `mapstructure:"disable_mac_commands"`
			DisableADR              bool     `mapstructure:"disable_adr"`
			MaxMACCommandErrorCount int      `mapstructure:"max_mac_command_error_count"`
			ADRPlugins              []string `mapstructure:"adr_plugins"`

			ExtraChannels []struct {
				Frequency uint32 `mapstructure:"frequency"`
				MinDR     int    `mapstructure:"min_dr"`
				MaxDR     int    `mapstructure:"max_dr"`
			} `mapstructure:"extra_channels"`

			ClassB struct {
				PingSlotDR        int    `mapstructure:"ping_slot_dr"`
				PingSlotFrequency uint32 `mapstructure:"ping_slot_frequency"`
			} `mapstructure:"class_b"`

			RejoinRequest struct {
				Enabled   bool `mapstructure:"enabled"`
				MaxCountN int  `mapstructure:"max_count_n"`
				MaxTimeN  int  `mapstructure:"max_time_n"`
			} `mapstructure:"rejoin_request"`
		} `mapstructure:"network_settings"`

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

	// handy-rusty storage
	ClickHouse struct {
		Servers       []string `mapstructure:"servers"`
		Host          string   `mapstructure:"host"`
		Port          string   `mapstructure:"port"`
		Debug         bool     `mapstructure:"debug"`
		UserName      string   `mapstructure:"username"`
		Password      string   `mapstructure:"password"`
		DataBase      string   `mapstructure:"database"`
		Automigrate   bool     `mapstructure:"automigrate"`
		DefaultSchema struct {
			TTL int `mapstructure:"ttl"`
		} `mapstructure:"default_schema"`
	} `mapstructure:"clickhouse"`
	// Prometheus struct {
	// 	Bind string `mapstructure:"bind"`
	// } `mapstructure:"prometheus"`
}

// C holds the global configuration.
var c Config

// Get
func Get() Config {
	return c
}

// Set
func Set(cfg Config) {
	c = cfg
	return
}
