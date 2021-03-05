package cmd

import (
	"context"
	"fmt"
	"sync"

	log "github.com/sirupsen/logrus"

	"github.com/fancar/wrenches/internal/config"
	"github.com/fancar/wrenches/internal/storage"
)

func setLogLevel(ctx context.Context, wg *sync.WaitGroup) error {
	log.SetLevel(log.Level(uint8(config.C.General.LogLevel)))
	return nil
}

func initLogger() {
	log.SetLevel(log.Level(uint8(config.C.General.LogLevel)))
	log.WithFields(log.Fields{
		"version":  version,
		"loglevel": config.C.General.LogLevel,
		// "docs":    "https://www. ... .su/",
	}).Info("[WRENCH] iot-server | additional tools have been started!")
}

func setupStorage() error {
	if err := storage.Setup(config.C); err != nil {
		return fmt.Errorf("setup storage error %w", err)
	}
	return nil
}
