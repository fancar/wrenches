package cmd

import (
	"bytes"
	"context"
	"fmt"
	// "os"
	// "os/signal"
	// "syscall"
	// "sync"
	// "encoding/csv"
	"encoding/hex"
	"encoding/json"
	"github.com/mohae/struct2csv"
	"io/ioutil"
	"strings"
	"time"

	"github.com/brocaar/lorawan"
	// "github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/fancar/wrenches/internal/config"
	"github.com/fancar/wrenches/internal/storage"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var getSessionsCmd = &cobra.Command{
	Use:   "get-sessions devEui1,devEui2,devEui3",
	Short: "get sessions, and store the data in csv. Use 'set-sessions help' for details",
	Long: `
	gets sessions for the list of devices and saves the output in file
	`,
	Run: getSessions,
}

var gsOutputFormat string // default - csv

type getSessionCtx struct {
	ctx            context.Context
	Devices        []lorawan.EUI64
	DeviceSessions []storage.DeviceSession
	AppSKeys       storage.AppSKeys // from AS. TEMP
	Data           []byte
}

func gsParseArgsToCtx(args []string) (*getSessionCtx, error) {

	if len(args) == 0 {
		return &getSessionCtx{}, fmt.Errorf("please specify at least one devEui as argument")
	}
	if strings.ToLower(args[0]) == "all" {
		return &getSessionCtx{}, fmt.Errorf("all - not supported yet")
	}

	devEuiStrArr := strings.Split(args[0], ",")
	var devEUIs []lorawan.EUI64

	for _, s := range devEuiStrArr {
		decoded, err := hex.DecodeString(s)
		if err != nil {
			return &getSessionCtx{}, fmt.Errorf("can't decode string %s: %w", s, err)
		}
		var devEui lorawan.EUI64
		copy(devEui[:], decoded[:])
		devEUIs = append(devEUIs, devEui)
	}

	ctx := context.Background()
	// ctx, cancel := context.WithCancel(context.Background())
	result := getSessionCtx{
		ctx:     ctx,
		Devices: devEUIs,
	}
	return &result, nil
}

func getSessions(cmd *cobra.Command, args []string) {
	setLogLevel()

	ctx, err := gsParseArgsToCtx(args)
	if err != nil {
		log.WithError(err).Error("can't parse arguments")
		return
	}

	tasks := []func(*getSessionCtx) error{
		checkOutputFormatGS,
		setupStorageGS,
		printGetSessionsStartMessage,
		// getAppSessionKeysFromAppServer,
		getDeviceSessionsfromRedis,
		marshalData,
		writeDataToFile,
	}

	for _, t := range tasks {
		if err := t(ctx); err != nil {
			log.Fatal(err)
		}
	}

	// exitChan := make(chan struct{})
	// sigChan := make(chan os.Signal)
	// signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	// log.WithField("signal", <-sigChan).Info("signal received")
	// go func() {
	// 	log.Warning("stopping application ...")
	// 	// if err := something.Stop(); err != nil {
	// 	// 	log.Fatal(err)
	// 	// }
	// 	exitChan <- struct{}{}
	// }()
	// select {
	// case <-exitChan:
	// case s := <-sigChan:
	// 	log.WithField("signal", s).Info("signal received, terminating")
	// }
}

func checkOutputFormatGS(ctx *getSessionCtx) error {
	gsOutputFormat = strings.ToLower(gsOutputFormat)
	switch gsOutputFormat {
	case "csv":
		log.Debug("the result will be stored in csv format")
	case "json":
		log.Debug("the result will be stored in json format")
	default:
		return fmt.Errorf("unknown format: '%s'. Please use json/csv", gsOutputFormat)
	}
	return nil
}

func setupStorageGS(ctx *getSessionCtx) error {
	if err := storage.Setup(config.Get()); err != nil {
		return fmt.Errorf("setup storage error %w", err)
	}
	return nil
}

func printGetSessionsStartMessage(ctx *getSessionCtx) error {
	log.WithFields(log.Fields{
		"device cnt": len(ctx.Devices),
	}).Info("Getting device-sessions from redis storage ...")
	return nil
}

func getAppSessionKeysFromAppServer(ctx *getSessionCtx) error {
	log.Info("Getting session keys from AppServer db ...")
	keys, err := storage.GetAppSKeys(ctx.ctx, storage.AppServer(), ctx.Devices)
	if err != nil {
		log.WithError(err).Errorf("getAppSessionKeysFromAppServer ")
		return nil
	}
	ctx.AppSKeys = keys
	log.Debug("Got session keys from AppServer db")
	return nil
}

func getDeviceSessionsfromRedis(ctx *getSessionCtx) error {
	var items []storage.DeviceSession

	for _, devEUI := range ctx.Devices {
		s, err := storage.GetDeviceSession(ctx.ctx, storage.RedisClient(), devEUI)
		if err != nil {
			log.WithError(err).WithFields(log.Fields{
				"dev_eui": devEUI,
			}).Error("can't get device-session")
		} else {
			items = append(items, *s)
		}
	}

	ctx.DeviceSessions = items

	if len(items) > 0 && len(items) != len(ctx.Devices) {
		log.Warnf("Got %d device_sessions while amount of devices you asked for is: %d", len(items), len(ctx.Devices))
	}

	return nil

}

func marshalData(ctx *getSessionCtx) error {
	switch gsOutputFormat {
	case "json":
		empJSON, err := json.MarshalIndent(ctx.DeviceSessions, "", "  ")
		if err != nil {
			log.WithError(err).Error("[marshalData] can't MarshalIndent")
		}
		ctx.Data = empJSON
		// fmt.Printf("DeviceSession[0] %s\n", string(empJSON))

	case "csv":
		buff := &bytes.Buffer{}
		writer := struct2csv.NewWriter(buff)
		csv, err := storage.ConvertDeviceSessionsToCSV(ctx.DeviceSessions)

		if err != nil {
			return fmt.Errorf("[marshalData] can't convert data to csv: %w", err)
		}

		err = writer.WriteStructs(csv)
		if err != nil {
			return fmt.Errorf("[marshalData] can't prepare csv: %w", err)
		}
		ctx.Data = buff.Bytes()
	default:
		return fmt.Errorf("[marshalData] unknown format selected: %s", gsOutputFormat)
	}

	return nil

	// fmt.Println(items[0].PendingRejoinDeviceSession)
}

func writeDataToFile(ctx *getSessionCtx) error {
	if gsOutputFormat == "json" {
		if ctx.Data != nil {
			fmt.Println("\n", string(ctx.Data))
		}
		return nil
	}

	fname := fmt.Sprintf("sessions_%s.%s", time.Now().Format("1504-02012006"), gsOutputFormat)
	err := ioutil.WriteFile(fname, ctx.Data, 0644)
	if err != nil {
		return fmt.Errorf("can't write to file: %w", err)
	}
	log.WithField("filename", fname).Info("Done! Results have been saved to the file")
	return nil
}
