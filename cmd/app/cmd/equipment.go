package cmd

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/fancar/wrenches/internal/config"
	"github.com/fancar/wrenches/internal/storage"
)

func setLogLevel() {
	// err := log.SetLevel(log.Level(uint8(config.Get().General.LogLevel)))
	// err := log.SetLevel(log.Level(5))
	// if err != nil {
	// 	log.WithError(err).Error("can't set loglevel")
	// }

	log.WithFields(log.Fields{
		"version":  version,
		"loglevel": config.Get().General.LogLevel,
		// "docs":    "https://www. ... .su/",
	}).Info("[WRENCH] iot-server | additional tools have been started!")
}

func initLogger() {
	log.SetLevel(log.Level(uint8(config.Get().General.LogLevel)))
	log.WithFields(log.Fields{
		"version":  version,
		"loglevel": config.Get().General.LogLevel,
		// "docs":    "https://www. ... .su/",
	}).Info("[WRENCH] iot-server | additional tools have been started!")
}

func setupStorage() error {
	if err := storage.Setup(config.Get()); err != nil {
		return fmt.Errorf("setup storage error %w", err)
	}
	return nil
}
