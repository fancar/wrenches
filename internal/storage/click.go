package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/brocaar/lorawan"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/fancar/wrenches/internal/config"
)

// DeviceLogItem is for imems to store in device_frames table
type DeviceLogItem struct {
	OrganizationID int64     `db:"organizationID"`
	GwOrgID        int64     `db:"gwOrgID"`
	Direction      string    `db:"direction"`
	DateTime       time.Time `db:"DateTime"`
	Date           time.Time `db:"Date"`
	MType          string    `db:"mType"`
	DevAddr        string    `db:"devAddr"`
	DevEUI         string    `db:"devEUI"`
	RxRssi         int32     `db:"rxRssi"`
	RxSnr          float64   `db:"rxSnr"`
	RxChannel      uint32    `db:"rxChannel"`
	RxRfChain      uint32    `db:"rxRfChain"`
	Gw             string    `db:"gw"`
	TxInfo         string    `db:"txInfo"`
	RxInfo         string    `db:"rxInfo"`
	PhyPayloadJSON string    `json:"PhyPayloadJson" db:"phyPayloadJson"`
	Airtime        float64   `db:"airtime"`
	Esp            float64   `db:"esp"`
	Late           uint8     `db:"late"`
	Class          string    `db:"class"`
	FPort          uint32    `db:"fPort"`
	FCnt           uint64    `db:"fCnt"`
	Mic            string    `db:"mic"`
	PhyPayload     string    `db:"phyPayload"` // phy payload | source byte array
	MacPayload     string    `db:"macPayload"`
	FrmPayload     string    `db:"frmPayload"` // decrypted user-data
	SpFact         uint32    `db:"spFact"`     // spreading factor
	ADR            uint8     `db:"adr"`
	FCntUp         uint32    `db:"FCntUp"`
	NFCntDown      uint32    `db:"NFCntDown"`
	AFCntDown      uint32    `db:"AFCntDown"`
	ConfFCnt       uint32    `db:"ConfFCnt"`
	Limit          uint32    `db:"limit"`
	Per            float32   `db:"per"`
	RedisID        string    `db:"redisID"`
}

// Connect makes new connection
func Connect() (*sql.Tx, error) {
	conn := hrDB
	if conn == nil {
		return &sql.Tx{}, errors.New("storage/clickhouse: not connected")
	}
	tx, err := conn.Begin()
	if err != nil {
		return &sql.Tx{}, errors.Wrap(err, "can't begin clickhouse session")
	}
	return tx, nil
}

// InitClickhouse connect to clickhouse.  We are storing lat lon table there
func InitClickhouse(cfg config.Config) (*sqlx.DB, error) {
	c := cfg.ClickHouse
	var err error

	if len(c.Servers) == 0 {
		if c.Host == "" || c.Port == "" {
			return nil, fmt.Errorf("InitClickhouse: you must specify at least one of clickhouse servers")
		}
		c.Servers = []string{c.Host + ":" + c.Port}
	}

	// init sql.DB using clickhouse lib
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: c.Servers,
		Auth: clickhouse.Auth{
			Database: c.DataBase,
			Username: c.UserName,
			Password: c.Password,
		},
		Debug: c.Debug,
		// TLS: &tls.Config{
		// 	InsecureSkipVerify: true,
		// },
		// Settings: clickhouse.Settings{
		// 	// "max_execution_time": 60,
		// },
		// DialTimeout: 5 * time.Second,
		// Compression: &clickhouse.Compression{
		// 	clickhouse.CompressionLZ4,
		// },
		// BlockBufferSize: 10,
	})
	// conn.SetMaxIdleConns(5)
	// conn.SetMaxOpenConns(10)
	// conn.SetConnMaxLifetime(time.Hour)

	// convert it to sqlx.DB
	db := sqlx.NewDb(conn, "clickhouse")

	for {
		err = db.Ping()
		if err != nil {
			if exception, ok := err.(*clickhouse.Exception); ok {
				if exception.Code == 516 { // bad userpass
					return db, err
				}

				log.Errorf("storage/clickhouse: code:%d msg:%s  will retry in 2s\n", exception.Code, exception.Message) // exception.StackTrace
			} else {
				log.WithError(err).Warning("storage: ping ClickHouse database error, will retry in 2s")

			}
			time.Sleep(2 * time.Second)

		} else {
			break
		}
	}

	log.Info(fmt.Sprintf("storage/clickhouse: connected to server(s): %s", c.Servers))
	return db, err
}

func GetLastFrameForDevEui(db *sqlx.DB, devEUI lorawan.EUI64, d string) (DeviceLogItem, error) {
	var result DeviceLogItem

	query := "SELECT * FROM device_frames WHERE devEUI=(?) AND direction=(?) ORDER BY DateTime DESC LIMIT 1"
	err := db.Get(&result, query, devEUI.String(), d)
	if err != nil {
		return DeviceLogItem{}, err
	}
	return result, nil
}
