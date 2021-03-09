package cmd

import (
	"context"
	"encoding/csv"
	"fmt"
	"github.com/fancar/wrenches/internal/storage"
	"strconv"
	// "github.com/gocarina/gocsv"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	// "io"
	"os"
	"reflect"
	"strings"
)

var setSessionsCmd = &cobra.Command{
	Use:   "set-sessions path/to/file.csv",
	Short: "save session to server-storages. Only for devices that exists in db!",
	Long: `
	the command saves sessions to inMemory storage (redis)
	and put application session keys to app-server database.
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

			// fmt.Println("setField: ", head[j], item)
			err := setField(&ds, fieldNum, item)
			if err != nil {
				return fmt.Errorf("Unable to set field '%s' with item '%s': %w", head[j], item, err)
			}
		}
		result = append(result, ds)
	}

	ctx.inputData = result
	log.Debug(fmt.Sprintf("Found rows in file! %d", len(records)))
	log.Debug(fmt.Sprintf("rows in result! %d", len(result)))

	// fmt.Println("RECORDS: ", records)
	// fmt.Println("RESULT: ", result)
	return nil
}

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
