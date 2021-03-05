package cmd

import (
	"bytes"
	"io/ioutil"

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

	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "path to configuration file (optional)")
	rootCmd.PersistentFlags().Int("log-level", 4, "debug=5, info=4, error=2, fatal=1, panic=0")

	viper.BindPFlag("general.log_level", rootCmd.PersistentFlags().Lookup("log-level"))

	viper.SetDefault("redis.servers", []string{"localhost:6379"})

	viper.SetDefault("ns.postgre.dsn", "postgres://localhost/chirpstack_ns?sslmode=disable")
	viper.SetDefault("ns.postgre.max_idle_connections", 2)
	viper.SetDefault("ns.postgre.max_open_connections", 0)

	viper.SetDefault("as.postgre.dsn", "postgres://localhost/chirpstack_as?sslmode=disable")
	viper.SetDefault("as.postgre.max_idle_connections", 2)
	viper.SetDefault("as.postgre.max_open_connections", 0)

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(getSessionsCmd)

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

	if err := viper.Unmarshal(&config.C); err != nil {
		log.WithError(err).Fatal("unmarshal config error")
	}
}
