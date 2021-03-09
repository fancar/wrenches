package storage

import (
	"context"
	"time"

	"github.com/brocaar/lorawan"
	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq/hstore"
)

// Device defines a LoRaWAN device.
type Device struct {
	DevEUI                    lorawan.EUI64     `db:"dev_eui"`
	CreatedAt                 time.Time         `db:"created_at"`
	UpdatedAt                 time.Time         `db:"updated_at"`
	LastSeenAt                *time.Time        `db:"last_seen_at"`
	ApplicationID             int64             `db:"application_id"`
	DeviceProfileID           uuid.UUID         `db:"device_profile_id"`
	Name                      string            `db:"name"`
	Description               string            `db:"description"`
	SkipFCntCheck             bool              `db:"-"`
	ReferenceAltitude         float64           `db:"-"`
	DeviceStatusBattery       *float32          `db:"device_status_battery"`
	DeviceStatusMargin        *int              `db:"device_status_margin"`
	DeviceStatusExternalPower bool              `db:"device_status_external_power_source"`
	DR                        *int              `db:"dr"`
	Latitude                  *float64          `db:"latitude"`
	Longitude                 *float64          `db:"longitude"`
	Altitude                  *float64          `db:"altitude"`
	DevAddr                   lorawan.DevAddr   `db:"dev_addr"`
	AppSKey                   lorawan.AES128Key `db:"app_s_key"`
	Variables                 hstore.Hstore     `db:"variables"`
	Tags                      hstore.Hstore     `db:"tags"`
	IsDisabled                bool              `db:"-"`
}

// AppSKeys map with DevEUI key
type AppSKeys map[lorawan.EUI64]lorawan.AES128Key

// GetDevice returns the device matching the given DevEUI.
func GetDevice(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) (Device, error) {
	var d Device
	err := sqlx.Get(db, &d, "select * from device where dev_eui = $1", devEUI[:])
	if err != nil {
		return d, handlePSQLError(err, "select error")
	}

	return d, nil
}

// GetAllAppSKeys returns AppSKeys for all DevEUIs.
// func GetAllAppSKeys(ctx context.Context, db sqlx.Queryer) (Device, error) {
// 	var d []Device
// 	err := sqlx.Select(db, &d, "select * from device where dev_eui = $1", devEUI[:])
// 	if err != nil {
// 		return d, handlePSQLError(err, "select error")
// 	}

// 	return d, nil
// }

// GetAppSKeys returns AppSKeys for given DevEUIs.
func GetAppSKeys(ctx context.Context, db sqlx.Queryer, devEUIs []lorawan.EUI64) (AppSKeys, error) {
	result := make(AppSKeys)
	query, args, err := sqlx.In("SELECT dev_eui,app_s_key FROM device WHERE dev_eui IN (?);", devEUIs)
	if err != nil {
		return result, handlePSQLError(err, "select error")
	}

	query = sqlx.Rebind(sqlx.DOLLAR, query)
	rows, err := db.Query(query, args...)
	// rows, err := db.Query(query)
	if err != nil {
		return result, handlePSQLError(err, "query error")
	}

	for rows.Next() {
		var eui lorawan.EUI64
		var key lorawan.AES128Key

		if err := rows.Scan(&eui, &key); err != nil {
			return result, handlePSQLError(err, "scan row error")
		}

		result[eui] = key
	}

	return result, nil
}
