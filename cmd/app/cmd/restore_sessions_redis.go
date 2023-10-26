package cmd

import (
	"context"
	// "encoding/csv"
	"bufio"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
	// "reflect"
	// "strconv"
	"strings"

	// "github.com/gocarina/gocsv"
	"github.com/brocaar/lorawan"
	// loraband "github.com/brocaar/lorawan/band"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/fancar/wrenches/internal/band"
	"github.com/fancar/wrenches/internal/config"
	"github.com/fancar/wrenches/internal/storage"
)

var DateTimeOfRedisDump string

var restoreSessionFromDumpCmd = &cobra.Command{
	Use:   "restore-sessions-from-dump path/to/deveui/list/file.txt",
	Short: "restore session according to data in databases with redis dump. Use 'restore-sessions-from-dump help' for details",
	Long: `
	the command tries to restore sessions to inMemory storage (redis)
	for dev_eui list given in file. It copies device-sessions from dump in case there are no joins after date_time given
	! It works only for known devices without device-session in redis with 
	- All devices that don't present on the platform will be skipped!
	- counters will be improved automatically in case there are packets in clickhouse database
	`,
	Args: cobra.MinimumNArgs(1),
	Run:  restoreSessionsFromDump,
}

type restoreSessionFromDump struct {
	cfg       config.Config
	ctx       context.Context
	inputFile string
	inputData []storage.DeviceSessionCSV
	devList   []lorawan.EUI64
	sessions  []storage.DeviceSession
	dumpDT    time.Time
}

func restoreSessionsFromDump(cmd *cobra.Command, args []string) {
	log.SetLevel(log.Level(uint8(config.Get().General.LogLevel)))

	// ctx, cancel := context.WithCancel(context.Background())
	c := restoreSessionFromDump{
		ctx:       context.Background(),
		cfg:       config.Get(),
		inputFile: args[0],
	}

	for _, f := range []func() error{
		c.parseDateTime,
		c.setupBand,
		c.setupStorages,
		c.parseInputFile,
		c.prepareDeviceSessions,
		c.createDeviceSessions,
	} {
		if err := f(); err != nil {
			// if err == ErrAbort {
			// 	return nil
			// }
			log.Fatal(err)
		}
	}
}

func (c *restoreSessionFromDump) parseDateTime() error {
	if DateTimeOfRedisDump == "" {
		return fmt.Errorf("You must specify datetime of your redis dump")
	}

	layout := "2006-01-02 15:04:05 -0700"
	// dateString := "2018-12-17 12:55:50 +0300"
	t, err := time.Parse(layout, DateTimeOfRedisDump)
	if err != nil {
		return fmt.Errorf("can't parse datetime: %w", err)
	}
	c.dumpDT = t

	return nil
}

func (c *restoreSessionFromDump) setupBand() error {
	if err := band.Setup(c.cfg); err != nil {
		return fmt.Errorf("unable setup band %w", err)
	}

	return nil
}

func (c *restoreSessionFromDump) setupStorages() error {
	if err := storage.Setup(c.cfg); err != nil {
		return fmt.Errorf("setup storage error %w", err)
	}

	if err := storage.SetupSecondRedis(c.cfg); err != nil {
		return fmt.Errorf("setup storage error %w", err)
	}

	return nil
}

func (c *restoreSessionFromDump) parseInputFile() error {
	log.Debugf("reading dev_eui list from %s ...", c.inputFile)
	file, err := os.Open(c.inputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	fileScanner := bufio.NewScanner(file)
	fileScanner.Split(bufio.ScanLines)
	var result []lorawan.EUI64

	for fileScanner.Scan() {
		var devEUI lorawan.EUI64
		text := fileScanner.Text()
		// s := strings.TrimLeft(fileScanner.Text(),'\\x')
		text = strings.ReplaceAll(text, " ", "")
		text = strings.TrimLeft(text, "\\x")

		s, err := hex.DecodeString(text)
		if err != nil {
			return fmt.Errorf("parseInputFile: can't parse dev_eui=%s: %w", text, err)
		}

		copy(devEUI[:], s)
		result = append(result, devEUI)
	}

	if len(result) == 0 {
		return fmt.Errorf("dev_eui not found in file %s", c.inputFile)
	}

	// ctx.inputData = result
	log.Infof("parseInputFile: got %d dev_eui(s) from file: %s ", len(result), c.inputFile)
	c.devList = result

	return nil
}

// get devices from local db and prepare sessions for devices that exist
func (c *restoreSessionFromDump) prepareDeviceSessions() error {
	succeed := 0
	no_old_ds := 0
	ds_exists := 0
	joins_exists := 0
	for _, devEUI := range c.devList {
		lf := log.Fields{"DevEUI": devEUI}

		log.WithFields(lf).Debug("Looking for the device-session in redis ...")

		// skip if ds exists allready
		_, err := storage.GetDeviceSession(c.ctx, storage.RedisClient(), devEUI)
		if err == nil {
			log.WithFields(lf).Info("a device session exists for the device. Skipped")
			ds_exists++
			continue
		}
		if err != nil && err != storage.ErrDoesNotExist {
			return fmt.Errorf("unable to check if device session exists for %s: %w", devEUI, err)
		}

		// look in older dump for device_session
		s, err := storage.GetDeviceSession(c.ctx, storage.RedisSecondClient(), devEUI)
		if err != nil {
			if err == storage.ErrDoesNotExist {
				log.WithFields(lf).Info("no device session found in old dump. Skipped")
				no_old_ds++
				continue
			}
			return fmt.Errorf("unable to check if device session exists in dump for %s: %w", devEUI, err)
		}

		nsDA, err := storage.GetLastDeviceActivation(c.ctx, storage.NetServer(), devEUI)
		if err != nil {
			if err != storage.ErrDoesNotExist {
				return fmt.Errorf("unable to get if device activation for %s: %w", devEUI, err)
			}
			log.WithFields(lf).Debugf("No device activations found")

		} else {
			if nsDA.CreatedAt.After(c.dumpDT) {
				log.WithFields(lf).Infof("There were joins for the device after %s. Skipped", c.dumpDT)
				joins_exists++
				continue
			}
		}

		lastRX, err := storage.GetLastFrameForDevEui(storage.Clickhouse(), devEUI, "RX")
		if err != nil {
			if err != sql.ErrNoRows {
				return fmt.Errorf("unable to get last uplink from clickhouse for %s: %w", devEUI, err)
			}
			log.WithFields(lf).Debugf("no uplinks in clickhouse. Going to use FCntUp=%d from backup", s.FCntUp)
		} else {
			s.FCntUp = lastRX.FCntUp + 1
			log.WithFields(lf).Debugf("found uplink in clickhouse. Going to use FCntUp=%d", s.FCntUp)
		}

		lastTX, err := storage.GetLastFrameForDevEui(storage.Clickhouse(), devEUI, "TX")
		if err != nil {
			if err != sql.ErrNoRows {
				return fmt.Errorf("unable to get last downlink from clickhouse  for %s: %w", devEUI, err)
			}
			log.WithFields(lf).Debugf("no downlinks in clickhouse. Going to use NFCntDown=%d, AFCntDown=%d from backup", s.NFCntDown, s.AFCntDown)
		} else {
			if s.GetMACVersion() == lorawan.LoRaWAN1_0 {
				s.NFCntDown = lastTX.NFCntDown + 1
				log.WithFields(lf).Debugf("found downlink in clickhouse. Going to use NFCntDown=%d", s.NFCntDown)
			} else {
				s.AFCntDown = lastTX.AFCntDown + 1
				log.WithFields(lf).Debugf("found downlink in clickhouse. Going to use AFCntDown=%d", s.AFCntDown)
			}

		}

		succeed++
		c.sessions = append(c.sessions, *s)
		jsonStr, err := json.MarshalIndent(s, "", "    ")
		if err != nil {
			return fmt.Errorf("Error marshaling JSON: %w", err)
		}

		log.WithFields(lf).Debugf("device_session to upload: %s", jsonStr)
	}
	log.WithFields(log.Fields{
		"succeed":      succeed,
		"no_old_ds":    no_old_ds,
		"ds_exists":    ds_exists,
		"joins_exists": joins_exists,
		"total":        len(c.devList),
	}).Infof("prepareDeviceSessions: prepared %d out of %d", succeed, len(c.devList))
	return nil
}

func (c *restoreSessionFromDump) createDeviceSessions() error {

	if len(c.sessions) == 0 {
		return fmt.Errorf("Not a single session has been assembled. Nothing to restore")
	}
	succeed := 0
	for _, s := range c.sessions {
		lf := log.Fields{"DevEUI": s.DevEUI}

		if err := storage.SaveDeviceSession(c.ctx, s); err != nil {
			return fmt.Errorf("save node-session error: %w", err)
		}

		// if err := storage.FlushMACCommandQueue(c.ctx, s.DevEUI); err != nil {
		// 	return fmt.Errorf("flush mac-command queue error: %s", err)
		// }
		succeed++
		log.WithFields(lf).Debug("a device session saved. Flushed Mac-command queue")
	}

	log.Infof("createDeviceSessions: created %d sesions out of prepared %d. Total in file:%d", succeed, len(c.sessions), len(c.devList))

	return nil
}
