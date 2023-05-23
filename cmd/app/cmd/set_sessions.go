package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/fancar/wrenches/internal/config"
	"github.com/fancar/wrenches/internal/storage"
	"strconv"
	// "github.com/gocarina/gocsv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	// "io"
	"encoding/hex"
	"os"
	"reflect"
	"strings"
)

var setSessionsCmd = &cobra.Command{
	Use:   "set-sessions path/to/file.csv",
	Short: "save session to server-storages. Use 'set-sessions help' for details",
	Long: `
	the command saves sessions to inMemory storage (redis)
	and put application session keys to app-server database.
	Only for devices that exists in db!
	- All devices that don't present at the platform will be skipped!
	- please note that you must increase up\down counters by your own!
	  (-u, -d params) in case it can take some time to move the devices.
	`,
	Args: cobra.MinimumNArgs(1),
	Run:  setSessions,
}

var upCntIncrease int   // to increment FCntUp
var downCntIncrease int // to increment NFCntDown

type setSessionCtx struct {
	ctx       context.Context
	inputFile string
	inputData []storage.DeviceSessionCSV
	// deviceSessions []storage.DeviceSession
	// Devices        []lorawan.EUI64
	// DeviceSessions []storage.DeviceSession
	// AppSKeys       storage.AppSKeys
	// Data           []byte
}

func ssParseArgsToCtx(args []string) (*setSessionCtx, error) {

	ctx := context.Background()
	// ctx, cancel := context.WithCancel(context.Background())
	result := setSessionCtx{
		ctx:       ctx,
		inputFile: args[0],
	}
	return &result, nil
}

func setSessions(cmd *cobra.Command, args []string) {
	setLogLevel()

	ctx, err := ssParseArgsToCtx(args)
	if err != nil {
		log.WithError(err).Error("can't parse arguments")
		return
	}

	tasks := []func(*setSessionCtx) error{
		printSetSessionsStartMessage,
		parseInputFile,
		setupStorageSS,
		prepareAndSaveDeviceSessions,
	}

	for _, t := range tasks {
		if err := t(ctx); err != nil {
			log.Fatal(err)
		}
	}
}

func printSetSessionsStartMessage(ctx *setSessionCtx) error {
	// log.WithFields(log.Fields{
	// 	"device cnt": len(ctx.Devices),
	// })
	log.Info("Setting device-sessions from file ...")
	return nil
}

func setupStorageSS(ctx *setSessionCtx) error {
	if err := storage.Setup(config.Get()); err != nil {
		return fmt.Errorf("setup storage error %w", err)
	}
	return nil
}

func parseInputFile(ctx *setSessionCtx) error {
	file, err := os.Open(ctx.inputFile)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("Unable to parse file %s: %w", ctx.inputFile, err)
	}

	result := []storage.DeviceSessionCSV{}

	head := records[0]
	// fmt.Println("head:", head)
	records = records[1:]

	columnIdxes, err := computeColumnIndexes(&storage.DeviceSessionCSV{}, head)
	if err != nil {
		return err
	}

	for _, r := range records {
		ds := storage.DeviceSessionCSV{}
		for j, item := range r {
			fieldNum, ok := columnIdxes[head[j]]
			if !ok {
				return fmt.Errorf("field %s does not exist within the provided item", head[j])
			}

			err := setField(&ds, fieldNum, item)
			if err != nil {
				return fmt.Errorf("Unable to set field '%s' with item '%s': %w", head[j], item, err)
			}
		}
		result = append(result, ds)
	}

	ctx.inputData = result
	log.Debug(fmt.Sprintf("Found rows in file! %d", len(records)))

	return nil
}

// get devices from local db and prepare sessions for devices that exist and save to redis
func prepareAndSaveDeviceSessions(ctx *setSessionCtx) error {
	// var items []storage.DeviceSession
	for _, row := range ctx.inputData {
		s := storage.DeviceSession{}
		DevEUI, err := hex.DecodeString(row.DevEUI)
		if err != nil {
			log.WithField("DevEUI", row.DevEUI).Error("Can't decode DevEUI hex from string. Row Skipped")
			continue
		}

		copy(s.DevEUI[:], DevEUI[:])
		log.WithField("DevEUI", row.DevEUI).Debug("Looking if the device exists in local ns-db ...")

		d, err := storage.GetDeviceFromNS(ctx.ctx, storage.NetServer(), s.DevEUI)
		if err != nil {
			log.WithField("DevEUI", row.DevEUI).Error("Row Skipped. Unable to get device-session: %s", err)
			continue
		}
		// log.WithField("DeviceProfileID", d.DeviceProfileID).Debug("Got it!")

		s.MACVersion = row.MACVersion

		s.DeviceProfileID = d.DeviceProfileID
		s.ServiceProfileID = d.ServiceProfileID
		s.RoutingProfileID = d.RoutingProfileID

		// sesion params
		DevAddr, err := hex.DecodeString(row.DevAddr)
		JoinEUI, err := hex.DecodeString(row.JoinEUI)
		FNwkSIntKey, err := hex.DecodeString(row.FNwkSIntKey)
		SNwkSIntKey, err := hex.DecodeString(row.SNwkSIntKey)
		NwkSEncKey, err := hex.DecodeString(row.NwkSEncKey)

		copy(s.DevAddr[:], DevAddr[:])
		copy(s.JoinEUI[:], JoinEUI[:])
		copy(s.FNwkSIntKey[:], FNwkSIntKey[:])
		copy(s.SNwkSIntKey[:], SNwkSIntKey[:])
		copy(s.NwkSEncKey[:], NwkSEncKey[:])

		s.FCntUp = row.FCntUp + uint32(upCntIncrease)
		s.NFCntDown = row.NFCntDown + uint32(downCntIncrease)
		s.AFCntDown = row.AFCntDown
		s.ConfFCnt = row.ConfFCnt

		if row.AESKey != "" {
			AESKey, err := hex.DecodeString(row.AESKey)
			if err != nil {
				log.WithError(err).WithField("DevEUI", row.DevEUI).Error("Unable to decode hex session params hex-str required. Skipped")
				continue
			}
			s.AppSKeyEvelope = &storage.KeyEnvelope{
				KekLabel: row.KEKLabel,
				AesKey:   AESKey,
			}
		}

		s.PingSlotNb = row.PingSlotNb
		s.IsDisabled = row.IsDisabled

		s.RXWindow = storage.RXWindow(row.RXWindow)
		s.RXDelay = uint8(row.RXDelay)
		s.RX1DROffset = uint8(row.RX1DROffset)
		s.RX2Frequency = row.RX2Frequency
		s.RX2DR = uint8(row.RX2DR)

		s.NbTrans = uint8(row.NbTrans)
		s.TXPowerIndex = int(row.TXPowerIndex)
		s.DR = int(row.DR)

		s.EnabledUplinkChannels = append(s.EnabledUplinkChannels, row.EnabledUplinkChannels...)

		if err := storage.SaveDeviceSession(ctx.ctx, s); err != nil {
			return fmt.Errorf("save node-session error: %w", err)
		}

		if err := storage.FlushMACCommandQueue(ctx.ctx, s.DevEUI); err != nil {
			return fmt.Errorf("flush mac-command queue error: %s", err)
		}
	}
	return nil
}

// func createDeviceSession(ctx *setSessionCtx, s *storage.DeviceSession) error {
// 	if err := storage.SaveDeviceSession(ctx.ctx, &s); err != nil {
// 		return fmt.Errorf("save node-session error: %w", err)
// 	}

// 	// if err := storage.FlushMACCommandQueue(ctx.ctx, s.DevEUI); err != nil {
// 	// 	return fmt.Errorf("flush mac-command queue error: %s", err)
// 	// }
// 	return nil
// }

// ******************************** CSV stuff *********************************************

// compute field indexes by csv-tags of the item stucture
func computeColumnIndexes(item interface{}, columnNames []string) (map[string]int, error) {
	result := map[string]int{}
	v := reflect.ValueOf(item).Elem()
	if !v.CanAddr() {
		return result, fmt.Errorf("computeColumnIndexes: cannot assign to the item passed, item must be a pointer in order to assign")
	}

	// It's possible we can cache this, which is why precompute all these ahead of time.
	findName := func(t reflect.StructTag) (string, error) {
		if jt, ok := t.Lookup("csv"); ok {
			return strings.Split(jt, ",")[0], nil
		}
		return "", fmt.Errorf("computeColumnIndexes: tag provided does not define a CSV tag %s", t)
	}

	// collecting fieldnames by tags
	for i := 0; i < v.NumField(); i++ {
		typeField := v.Type().Field(i)
		tag := typeField.Tag
		jname, _ := findName(tag)
		result[jname] = i
	}
	return result, nil
}

// setField - sets the item with value according to fieldName (tag)
func setField(item interface{}, fieldNum int, value interface{}) error {
	v := reflect.ValueOf(item).Elem()
	if !v.CanAddr() {
		return fmt.Errorf("cannot assign to the item passed, item must be a pointer in order to assign")
	}

	vf := reflect.ValueOf(value) // it allways string tho ...
	if vf.IsValid() {
		fieldVal := v.Field(fieldNum)
		// fieldType := fieldVal.Type()

		t, err := stringTypeConverter(fieldVal, vf.Interface().(string))
		if err != nil {
			return err
		}
		fieldVal.Set(t)
		return nil
		// 	// v.Elem().Field(i).Set(kValue.Convert(typeOfS.Field(i).Type))
	}
	return fmt.Errorf("field '%s' is invalid", vf)
}

func stringTypeConverter(wanted reflect.Value, toConvert string) (reflect.Value, error) {
	switch wanted.Kind() {
	case reflect.Int:
		i, err := strconv.Atoi(toConvert)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Can't convert value %s to integer: %w", toConvert, err)
		}
		result := reflect.ValueOf(i)
		return result, nil

	case reflect.Uint32:
		i, err := strconv.Atoi(toConvert)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Can't convert value %s to uint32: %w", toConvert, err)
		}
		result := reflect.ValueOf(uint32(i))
		return result, nil

	case reflect.Slice:
		arr := strings.Split(toConvert, ",")
		var ints []int
		for _, n := range arr {
			i, err := strconv.Atoi(n)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("Can't convert value %s to uint32: %w", toConvert, err)
			}
			ints = append(ints, i)
		}

		result := reflect.ValueOf(ints)
		return result, nil

	case reflect.Bool:
		i, err := strconv.ParseBool(toConvert)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("Can't convert value %s to boolean: %w", toConvert, err)
		}
		result := reflect.ValueOf(i)
		return result, nil

	case reflect.String:
		return reflect.ValueOf(toConvert), nil
		// return toConvert.Convert(wanted.Type()), nil

	default:
		return reflect.Value{}, fmt.Errorf("Can't convert '%s'. Unsupported type wanted: '%s' ", toConvert, wanted.Kind())
	}
}
