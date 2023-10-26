package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/brocaar/lorawan"
	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq/hstore"
)

// Device defines a LoRaWAN device according to AS
type Device struct {
	DevEUI                    lorawan.EUI64     `db:"dev_eui"`
	CreatedAt                 *time.Time        `db:"created_at"`
	UpdatedAt                 time.Time         `db:"updated_at"`
	LastSeenAt                *time.Time        `db:"last_seen_at"`
	ApplicationID             int64             `db:"application_id"` // The relation will be removed. ERTH-mod
	DeviceProfileID           uuid.UUID         `db:"device_profile_id"`
	DeviceProfileName         string            `db:"device_profile_name"`
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
	IsDisabled                bool              `db:"is_disabled"`
	KeepQueue                 bool              `db:"-"`
	BatteryUpdatedAt          sql.NullTime      `db:"battery_status_updated_at"` // ERTH-mod
	BatteryLevelUpdatedAt     sql.NullTime      `db:"battery_level_updated_at"`  // ERTH-mod
	BatteryReplacedAt         sql.NullTime      `db:"battery_replaced_at"`       // ERTH-mod
	FirstUplinkAt             sql.NullTime      `db:"first_uplink_at"`           // ERTH-mod
	UpdatedByUserAt           sql.NullTime      `db:"updated_by_userid_at"`      // ERTH-mod
	UpdatedByUserID           sql.NullInt64     `db:"updated_by_userid"`         // ERTH-mod
	AvgPER                    sql.NullFloat64   `db:"avg_per"`                   // ERTH-mod
	AvgSNR                    sql.NullFloat64   `db:"avg_snr"`                   // ERTH-mod
	AvgRSSI                   sql.NullFloat64   `db:"avg_rssi"`                  // ERTH-mod

	//  is a new relation id instead of ApplicationID
	RoutingProfileID   int64          `db:"routing_profile_id"`   // ERTH-mod, used to be asrp_id
	ServiceProfileID   uuid.UUID      `db:"service_profile_id"`   // ERTH-mod
	ServiceProfileName sql.NullString `db:"service_profile_name"` // ERTH-mod

	FCntAutomaticReset bool `db:"fcnt_automatic_reset"` // ERTH-mod
}

// AppSKeys map with DevEUI key
type AppSKeys map[lorawan.EUI64]lorawan.AES128Key

// GetDeviceFromAS returns the device matching the given DevEUI.
func GetDeviceFromAS(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) (Device, error) {
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
