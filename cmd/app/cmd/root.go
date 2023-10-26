package cmd

import (
	"bytes"
	"io/ioutil"
	"time"

	"github.com/fancar/wrenches/internal/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var version string

// Execute executes the root command.
func Execute(v string) {
	version = v
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

var rootCmd = &cobra.Command{
	Use:   "wrenches",
	Short: "wrenches iot-server tools",
	Long:  `this are tools for iot-netserver based on chirpstack project`,
	// > documentation & support: !!! left empty !!!
	// > source & copyright information: !!! left empty !!!  `,
	// RunE: run,
}

func init() {

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "path to configuration file (optional). Default config.toml")
	rootCmd.PersistentFlags().Int("log-level", 4, "debug=5, info=4, error=2, fatal=1, panic=0")

	viper.BindPFlag("general.log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	viper.SetDefault("redis.servers", []string{"localhost:6379"})

	// viper.SetDefault("ns.band_name", "RU864")
	viper.SetDefault("ns.device_session_ttl", time.Hour*24*31)

	viper.SetDefault("ns.band.name", "RU864")
	viper.SetDefault("ns.band.uplink_max_eirp", -1)

	viper.SetDefault("ns.network_settings.installation_margin", 10)
	viper.SetDefault("ns.network_settings.rx1_delay", 1)
	viper.SetDefault("ns.network_settings.rx2_frequency", 0)
	viper.SetDefault("ns.network_settings.rx2_dr", 0)
	viper.SetDefault("ns.network_settings.gateway_prefer_min_margin", 10)
	viper.SetDefault("ns.network_settings.downlink_tx_power", 0)
	viper.SetDefault("ns.network_settings.disable_adr", false)
	viper.SetDefault("ns.network_settings.max_mac_command_error_count", 3)

	viper.SetDefault("ns.postgre.dsn", "postgres://localhost/chirpstack_ns?sslmode=disable")
	viper.SetDefault("ns.postgre.max_idle_connections", 2)
	viper.SetDefault("ns.postgre.max_open_connections", 0)

	viper.SetDefault("as.postgre.dsn", "postgres://localhost/chirpstack_as?sslmode=disable")
	viper.SetDefault("as.postgre.max_idle_connections", 2)
	viper.SetDefault("as.postgre.max_open_connections", 0)

	viper.SetDefault("clickhouse.host", "localhost")
	viper.SetDefault("clickhouse.port", "9000")
	viper.SetDefault("clickhouse.username", "default")
	viper.SetDefault("clickhouse.password", "")
	viper.SetDefault("clickhouse.database", "iot_flow")
	viper.SetDefault("clickhouse.debug", false)
	viper.SetDefault("clickhouse.automigrate", true)
	viper.SetDefault("clickhouse.default_schema.ttl", 12)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)

	rootCmd.AddCommand(plCryptCmd)
	// plCryptCmd.PersistentFlags().StringVarP(&plCryptData, "decrypt", "i", "", "decrypt data. Hex String Byte Array")
	plCryptCmd.PersistentFlags().StringVarP(&plCryptAppSessonKey, "session-key", "s", "", "Application Session Key")
	plCryptCmd.PersistentFlags().StringVarP(&plCryptDevAddr, "devaddr", "a", "", "specify device DevAddr")
	plCryptCmd.PersistentFlags().Uint32VarP(&plCryptFCnt, "fCnt", "f", 0, "specify device fCnt")
	plCryptCmd.Flags().BoolVarP(&plCryptDecrypt, "decrypt", "d", false, "decrypt data. (By default data will be encrypted)")

	rootCmd.AddCommand(getSessionsCmd)
	getSessionsCmd.Flags().StringVarP(&gsOutputFormat, "output-format", "o", "json", "output format json/csv. Default: json")

	rootCmd.AddCommand(setSessionsCmd)
	setSessionsCmd.PersistentFlags().IntVarP(&upCntIncrease, "up-cnt-increase", "u", 0, "the number to increase FCntUp counter (required)")
	setSessionsCmd.PersistentFlags().IntVarP(&downCntIncrease, "down-cnt-increase", "d", 0, "the number to increase NFCntDown counter (required)")
	// setSessionsCmd.MarkPersistentFlagRequired("up-cnt-increase")
	setSessionsCmd.MarkPersistentFlagRequired("down-cnt-increase")

	rootCmd.AddCommand(restoreSessionsCmd)
	rootCmd.AddCommand(restoreSessionFromDumpCmd)

	restoreSessionFromDumpCmd.PersistentFlags().StringVarP(&DateTimeOfRedisDump, "redis-dump-datetime", "d", "", "the datetime when second redis dump had been taken. ex '2006-01-02 15:04:05 -0700'")
	restoreSessionFromDumpCmd.MarkPersistentFlagRequired("redis-dump-datetime")

}

func initConfig() {
	config.Version = version
	if cfgFile != "" {
		b, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			log.WithError(err).WithField("config", cfgFile).Fatal("error loading config file")
		}
		viper.SetConfigType("toml")
		if err := viper.ReadConfig(bytes.NewBuffer(b)); err != nil {
			log.WithError(err).WithField("config", cfgFile).Fatal("error loading config file")
		}
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.AddConfigPath("$HOME/.config/wrenches")
		viper.AddConfigPath("/etc/wrenches")
		if err := viper.ReadInConfig(); err != nil {
			switch err.(type) {
			case viper.ConfigFileNotFoundError:
			default:
				log.WithError(err).Fatal("read configuration file error")
			}
		}
	}
	var cfg config.Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.WithError(err).Fatal("unmarshal config error")
	}
	config.Set(cfg)
}
