package cmd

import (
	"context"
	// "encoding/csv"
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	// "reflect"
	// "strconv"
	"strings"

	// "github.com/gocarina/gocsv"
	"github.com/brocaar/lorawan"
	loraband "github.com/brocaar/lorawan/band"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/fancar/wrenches/internal/band"
	"github.com/fancar/wrenches/internal/config"
	"github.com/fancar/wrenches/internal/storage"
)

var restoreSessionsCmd = &cobra.Command{
	Use:   "restore-sessions path/to/deveui/list/file.txt",
	Short: "restore session according to data in databases. Use 'restore-sessions help' for details",
	Long: `
	the command tries to restore sessions to inMemory storage (redis)
	from dev_eui list from the file name passed as an argument
	Works only for devices that exist in db and have some frames logged in clickhouse!
	- All devices that don't present at the platform will be skipped!
	- please note that you must increase up\down counters by your own if needed
	  (-u, -d params)
	`,
	Args: cobra.MinimumNArgs(1),
	Run:  restoreSessions,
}

// var upCntIncrease int   // to increment FCntUp
// var downCntIncrease int // to increment NFCntDown

type restoreSession struct {
	cfg       config.Config
	ctx       context.Context
	inputFile string
	inputData []storage.DeviceSessionCSV
	devList   []lorawan.EUI64
	sessions  []storage.DeviceSession
	// deviceSessions []storage.DeviceSession
	// Devices        []lorawan.EUI64
	// DeviceSessions []storage.DeviceSession
	// AppSKeys       storage.AppSKeys
	// Data           []byte
}

func restoreSessions(cmd *cobra.Command, args []string) {
	log.SetLevel(log.Level(uint8(config.Get().General.LogLevel)))

	// ctx, cancel := context.WithCancel(context.Background())
	c := restoreSession{
		ctx:       context.Background(),
		cfg:       config.Get(),
		inputFile: args[0],
	}

	for _, f := range []func() error{
		c.setupBand,
		c.setupStorages,
		c.parseInputFile,
		c.prepareDeviceSessions,
		c.createDeviceSessions,
		// c.printStartMessage,
	} {
		if err := f(); err != nil {
			// if err == ErrAbort {
			// 	return nil
			// }
			log.Fatal(err)
		}
	}
}

func (c *restoreSession) setupBand() error {
	if err := band.Setup(c.cfg); err != nil {
		return fmt.Errorf("unable setup band %w", err)
	}

	return nil
}

func (c *restoreSession) setupStorages() error {
	if err := storage.Setup(c.cfg); err != nil {
		return fmt.Errorf("setup storage error %w", err)
	}
	return nil
}

func (c *restoreSession) parseInputFile() error {
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
func (c *restoreSession) prepareDeviceSessions() error {
	succeed := 0
	for _, devEUI := range c.devList {
		lf := log.Fields{"DevEUI": devEUI}

		log.WithFields(lf).Debug("Looking for the device in databases ...")

		asDevice, err := storage.GetDeviceFromAS(c.ctx, storage.AppServer(), devEUI)
		if err != nil {
			log.WithFields(lf).Errorf("Skipped. Unable to get the device from AS: %v", err)
			continue
		}

		d, err := storage.GetDeviceFromNS(c.ctx, storage.NetServer(), devEUI)
		if err != nil {
			log.WithFields(lf).Errorf("Skipped. Unable to get the device from NS: %v", err)
			continue
		}

		nsDP, err := storage.GetDeviceProfileFromNS(c.ctx, storage.NetServer(), d.DeviceProfileID)
		if err != nil {
			log.WithFields(lf).Errorf("Skipped. Unable to get the device profile %s from NS: %v", d.DeviceProfileID, err)
			continue
		}

		nsDA, err := storage.GetLastDeviceActivation(c.ctx, storage.NetServer(), devEUI)
		if err != nil {
			log.WithFields(lf).Errorf("Skipped. Unable to get the device activation from NS: %v", err)
			continue
		}

		if asDevice.DevAddr != nsDA.DevAddr {
			lf["as_devaddr"] = asDevice.DevAddr
			lf["ns_devaddr"] = nsDA.DevAddr
			log.WithFields(lf).Errorf("Skipped. DevAddr values in database(AS) and in last device_activation(NS) are not equal")
			continue
		}

		lastRX, err := storage.GetLastFrameForDevEui(storage.Clickhouse(), devEUI, "RX")
		if err != nil {
			log.WithFields(lf).Errorf("Skipped. unable to get last uplink from clickhouse")
			continue
		}
		lastTX, err := storage.GetLastFrameForDevEui(storage.Clickhouse(), devEUI, "TX")
		if err != nil {
			log.WithFields(lf).Errorf("Skipped. unable to get last uplink from clickhouse")
			continue
		}

		s := storage.DeviceSession{
			// as data
			AppSKey: asDevice.AppSKey,
			ADR:     true,

			// data according to device settings
			DevEUI:           devEUI,
			DeviceProfileID:  d.DeviceProfileID,
			ServiceProfileID: d.ServiceProfileID,
			RoutingProfileID: d.RoutingProfileID,
			IsDisabled:       d.IsDisabled,

			MACVersion: nsDP.MACVersion,

			// session parameters
			DevAddr:     nsDA.DevAddr,
			JoinEUI:     nsDA.JoinEUI,
			FNwkSIntKey: nsDA.FNwkSIntKey,
			SNwkSIntKey: nsDA.SNwkSIntKey,
			NwkSEncKey:  nsDA.NwkSEncKey,

			// data based on last frames
			FCntUp:              lastRX.FCntUp + 1,
			ConfFCnt:            lastTX.ConfFCnt, // on confirmed dl ack equals dq fcnt
			ExtraUplinkChannels: make(map[int]loraband.Channel),
		}

		if s.GetMACVersion() == lorawan.LoRaWAN1_0 {
			s.NFCntDown = lastTX.NFCntDown + 1
		} else {
			s.AFCntDown = lastTX.AFCntDown + 1
		}

		if nsDP.PingSlotPeriod != 0 {
			s.PingSlotNb = (1 << 12) / nsDP.PingSlotPeriod
		}

		s.RXWindow = storage.RX1
		s.RX2DR = uint8(c.cfg.NetworkServer.NetworkSettings.RX2DR)
		s.RX1DROffset = uint8(c.cfg.NetworkServer.NetworkSettings.RX1DROffset)
		s.RXDelay = uint8(c.cfg.NetworkServer.NetworkSettings.RX1Delay)

		if nsDP.RX1DROffset != -1 {
			s.RX1DROffset = uint8(nsDP.RX1DROffset)
		}
		// s.RX2DR = -1
		if nsDP.RX2DataRate != -1 {
			s.RX2DR = uint8(nsDP.RX2DataRate)
		}

		s.NbTrans = 3
		s.TXPowerIndex = 0
		s.DR = 0

		s.RX2Frequency = band.Band().GetDefaults().RX2Frequency
		s.EnabledUplinkChannels = band.Band().GetStandardUplinkChannelIndices()

		if cfList := band.Band().GetCFList(nsDP.MACVersion); cfList != nil && cfList.CFListType == lorawan.CFListChannel {
			channelPL, ok := cfList.Payload.(*lorawan.CFListChannelPayload)
			if !ok {
				return fmt.Errorf("expected *lorawan.CFListChannelPayload, got %T", cfList.Payload)
			}

			for _, f := range channelPL.Channels {
				if f == 0 {
					continue
				}

				i, err := band.Band().GetUplinkChannelIndex(f, false)
				if err != nil {
					// if this happens, something is really wrong
					log.WithError(err).WithFields(log.Fields{
						"frequency": f,
					}).Error("unknown cflist frequency")
					continue
				}

				// add extra channel to enabled channels
				s.EnabledUplinkChannels = append(s.EnabledUplinkChannels, i)

				// add extra channel to extra uplink channels, so that we can
				// keep track on frequency and data-rate changes
				c, err := band.Band().GetUplinkChannel(i)
				if err != nil {
					return fmt.Errorf("unable to get uplink channel %d: %w", i, err)
				}
				s.ExtraUplinkChannels[i] = c
			}
		}
		succeed++
		c.sessions = append(c.sessions, s)
		jsonStr, err := json.MarshalIndent(s, "", "    ")
		if err != nil {
			return fmt.Errorf("Error marshaling JSON: %w", err)
		}

		log.WithFields(lf).Debugf("device_session to upload: %s", jsonStr)
		// if err := storage.SaveDeviceSession(ctx.ctx, s); err != nil {
		// 	return fmt.Errorf("save node-session error: %w", err)
		// }

		// if err := storage.FlushMACCommandQueue(ctx.ctx, s.DevEUI); err != nil {
		// 	return fmt.Errorf("flush mac-command queue error: %s", err)
		// }
	}
	log.Infof("prepareDeviceSessions: prepared %d out of %d", succeed, len(c.devList))
	return nil
}

func (c *restoreSession) createDeviceSessions() error {

	if len(c.sessions) == 0 {
		return fmt.Errorf("Not a single session has been assembled. Nothing to restore")
	}
	succeed := 0
	for _, s := range c.sessions {
		lf := log.Fields{"DevEUI": s.DevEUI}
		_, err := storage.GetDeviceSession(c.ctx, storage.RedisClient(), s.DevEUI)
		if err == nil {
			log.WithFields(lf).Warn("a device session exists for the device. Skipped")
			continue
		}
		if err != nil && err != storage.ErrDoesNotExist {
			return fmt.Errorf("unable to check if device session exists for %s: %w", s.DevEUI, err)
		}

		if err := storage.SaveDeviceSession(c.ctx, s); err != nil {
			return fmt.Errorf("save node-session error: %w", err)
		}

		if err := storage.FlushMACCommandQueue(c.ctx, s.DevEUI); err != nil {
			return fmt.Errorf("flush mac-command queue error: %s", err)
		}
		succeed++
		log.WithFields(lf).Debug("a device session saved. Flushed Mac-command queue")
	}

	log.Infof("createDeviceSessions: created %d sesions out of prepared %d. Total in file:%d", succeed, len(c.sessions), len(c.devList))

	// if err := storage.FlushMACCommandQueue(ctx.ctx, s.DevEUI); err != nil {
	// 	return fmt.Errorf("flush mac-command queue error: %s", err)
	// }
	return nil
}

// ******************************** CSV stuff *********************************************

// compute field indexes by csv-tags of the item stucture
// func computeColumnIndexes(item interface{}, columnNames []string) (map[string]int, error) {
// 	result := map[string]int{}
// 	v := reflect.ValueOf(item).Elem()
// 	if !v.CanAddr() {
// 		return result, fmt.Errorf("computeColumnIndexes: cannot assign to the item passed, item must be a pointer in order to assign")
// 	}

// 	// It's possible we can cache this, which is why precompute all these ahead of time.
// 	findName := func(t reflect.StructTag) (string, error) {
// 		if jt, ok := t.Lookup("csv"); ok {
// 			return strings.Split(jt, ",")[0], nil
// 		}
// 		return "", fmt.Errorf("computeColumnIndexes: tag provided does not define a CSV tag %s", t)
// 	}

// 	// collecting fieldnames by tags
// 	for i := 0; i < v.NumField(); i++ {
// 		typeField := v.Type().Field(i)
// 		tag := typeField.Tag
// 		jname, _ := findName(tag)
// 		result[jname] = i
// 	}
// 	return result, nil
// }

// // setField - sets the item with value according to fieldName (tag)
// func setField(item interface{}, fieldNum int, value interface{}) error {
// 	v := reflect.ValueOf(item).Elem()
// 	if !v.CanAddr() {
// 		return fmt.Errorf("cannot assign to the item passed, item must be a pointer in order to assign")
// 	}

// 	vf := reflect.ValueOf(value) // it allways string tho ...
// 	if vf.IsValid() {
// 		fieldVal := v.Field(fieldNum)
// 		// fieldType := fieldVal.Type()

// 		t, err := stringTypeConverter(fieldVal, vf.Interface().(string))
// 		if err != nil {
// 			return err
// 		}
// 		fieldVal.Set(t)
// 		return nil
// 		// 	// v.Elem().Field(i).Set(kValue.Convert(typeOfS.Field(i).Type))
// 	}
// 	return fmt.Errorf("field '%s' is invalid", vf)
// }

// func stringTypeConverter(wanted reflect.Value, toConvert string) (reflect.Value, error) {
// 	switch wanted.Kind() {
// 	case reflect.Int:
// 		i, err := strconv.Atoi(toConvert)
// 		if err != nil {
// 			return reflect.Value{}, fmt.Errorf("Can't convert value %s to integer: %w", toConvert, err)
// 		}
// 		result := reflect.ValueOf(i)
// 		return result, nil

// 	case reflect.Uint32:
// 		i, err := strconv.Atoi(toConvert)
// 		if err != nil {
// 			return reflect.Value{}, fmt.Errorf("Can't convert value %s to uint32: %w", toConvert, err)
// 		}
// 		result := reflect.ValueOf(uint32(i))
// 		return result, nil

// 	case reflect.Slice:
// 		arr := strings.Split(toConvert, ",")
// 		var ints []int
// 		for _, n := range arr {
// 			i, err := strconv.Atoi(n)
// 			if err != nil {
// 				return reflect.Value{}, fmt.Errorf("Can't convert value %s to uint32: %w", toConvert, err)
// 			}
// 			ints = append(ints, i)
// 		}

// 		result := reflect.ValueOf(ints)
// 		return result, nil

// 	case reflect.Bool:
// 		i, err := strconv.ParseBool(toConvert)
// 		if err != nil {
// 			return reflect.Value{}, fmt.Errorf("Can't convert value %s to boolean: %w", toConvert, err)
// 		}
// 		result := reflect.ValueOf(i)
// 		return result, nil

// 	case reflect.String:
// 		return reflect.ValueOf(toConvert), nil
// 		// return toConvert.Convert(wanted.Type()), nil

// 	default:
// 		return reflect.Value{}, fmt.Errorf("Can't convert '%s'. Unsupported type wanted: '%s' ", toConvert, wanted.Kind())
// 	}
// }
