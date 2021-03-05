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
	"github.com/fancar/wrenches/internal/storage"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var getSessionsCmd = &cobra.Command{
	Use:   "get-sessions",
	Short: "get  sessions, and store the data in csv",
	Long: `the app gets from inMemory storage (redis)
	`,
	Run: getSessions,
}

type getSessionCtx struct {
	ctx            context.Context
	Devices        []lorawan.EUI64 // array of devEui strings
	DeviceSessions []storage.DeviceSession
}

func parseArgsToCtx(args []string) (*getSessionCtx, error) {
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
	initLogger()
	ctx, err := parseArgsToCtx(args)
	if err != nil {
		log.WithError(err).Error("can't parse arguments")
		return
	}
	setupStorage()
	tasks := []func(*getSessionCtx) error{
		// setLogLevel,
		printGetSessionsStartMessage,
		getDeviceSessionsfromRedis,
		writeCSV,
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

func printGetSessionsStartMessage(ctx *getSessionCtx) error {
	log.WithFields(log.Fields{
		"device cnt": len(ctx.Devices),
	}).Info("Getting device-sessions from network-server storage ...")
	return nil
}

func getDeviceSessionsfromRedis(ctx *getSessionCtx) error {
	var items []storage.DeviceSession

	for _, devEUI := range ctx.Devices {
		s, err := storage.GetDeviceSession(ctx.ctx, devEUI)
		if err != nil {
			// TODO: in case not found, remove the DevEUI from the list
			log.WithFields(log.Fields{
				"dev_eui": devEUI,
				// "ctx_id":   ctx.Value(logging.ContextIDKey),
			}).Error("get device-session error: %s", err)
		} else {
			items = append(items, s)
		}

		// It is possible that the "main" device-session maps to a different
		// devAddr as the PendingRejoinDeviceSession is set (using the devAddr
		// that is used for the lookup).
		// if s.DevAddr == devAddr {
		// 	items = append(items, s)
		// }

		// When a pending rejoin device-session context is set and it has
		// the given devAddr, add it to the items list.
		// if s.PendingRejoinDeviceSession != nil && s.PendingRejoinDeviceSession.DevAddr == devAddr {
		// 	items = append(items, *s.PendingRejoinDeviceSession)
		// }
	}

	ctx.DeviceSessions = items

	if len(items) > 0 {
		empJSON, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			log.WithError(err).Error("can't MarshalIndent")
		}
		fmt.Printf("DeviceSession[0] %s\n", string(empJSON))

		// fmt.Println(items[0].PendingRejoinDeviceSession)
	}

	log.WithField("items_len", len(items)).Debug("Got sessions from Redis!")

	return nil

}

func writeCSV(ctx *getSessionCtx) error {

	// w := csv.NewWriter(os.Stdout)
	// headers := my_struct.GetHeaders()

	// proc := storage.ProcessCSV(data)

	buff := &bytes.Buffer{}
	writer := struct2csv.NewWriter(buff)
	err := writer.WriteStructs(ctx.DeviceSessions)
	if err != nil {
		return fmt.Errorf("can't prepare csv: %w", err)
	}

	fname := fmt.Sprintf("sessions_%s.csv", time.Now().Format("1504-02012006"))
	err = ioutil.WriteFile(fname, buff.Bytes(), 0644)
	if err != nil {
		return fmt.Errorf("can't write csv file: %w", err)
	}

	fmt.Println(buff)

	return nil
}
