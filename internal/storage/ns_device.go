package storage

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	// log "github.com/sirupsen/logrus"

	"github.com/brocaar/lorawan"
)

// DeviceMode defines the mode in which the device operates.
type DeviceMode string

// Available device modes.
const (
	DeviceModeA DeviceMode = "A"
	DeviceModeB DeviceMode = "B"
	DeviceModeC DeviceMode = "C"
)

// NSDevice defines a LoRaWAN device in NS database.
type NSDevice struct {
	DevEUI            lorawan.EUI64 `db:"dev_eui"`
	CreatedAt         time.Time     `db:"created_at"`
	UpdatedAt         time.Time     `db:"updated_at"`
	DeviceProfileID   uuid.UUID     `db:"device_profile_id"`
	ServiceProfileID  uuid.UUID     `db:"service_profile_id"`
	RoutingProfileID  uuid.UUID     `db:"routing_profile_id"`
	SkipFCntCheck     bool          `db:"skip_fcnt_check"`
	ReferenceAltitude float64       `db:"reference_altitude"`
	Mode              DeviceMode    `db:"mode"`
	IsDisabled        bool          `db:"is_disabled"`
	KeepQueue         bool          `db:"keep_queue"`
}

// DeviceActivation defines the device-activation for a LoRaWAN device.
type DeviceActivation struct {
	ID          int64             `db:"id"`
	CreatedAt   time.Time         `db:"created_at"`
	DevEUI      lorawan.EUI64     `db:"dev_eui"`
	JoinEUI     lorawan.EUI64     `db:"join_eui"`
	DevAddr     lorawan.DevAddr   `db:"dev_addr"`
	FNwkSIntKey lorawan.AES128Key `db:"f_nwk_s_int_key"`
	SNwkSIntKey lorawan.AES128Key `db:"s_nwk_s_int_key"`
	NwkSEncKey  lorawan.AES128Key `db:"nwk_s_enc_key"`
	DevNonce    lorawan.DevNonce  `db:"dev_nonce"`
	JoinReqType lorawan.JoinType  `db:"join_req_type"`
}

// GetDeviceFromNS returns the device matching the given DevEUI.
func GetDeviceFromNS(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) (NSDevice, error) {
	var d NSDevice
	err := sqlx.Get(db, &d, "select * from device where dev_eui = $1", devEUI[:])
	if err != nil {
		return d, handlePSQLError(err, "select error")
	}

	return d, nil
}
