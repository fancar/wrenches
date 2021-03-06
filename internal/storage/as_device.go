package storage

import (
	"context"
	// "strings"
	// "fmt"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq/hstore"
	// "github.com/pkg/errors"
	// log "github.com/sirupsen/logrus"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/codes"

	// "github.com/brocaar/chirpstack-api/go/v3/ns"
	// "github.com/brocaar/chirpstack-application-server/internal/backend/networkserver"
	// "github.com/brocaar/chirpstack-application-server/internal/config"
	// "github.com/brocaar/chirpstack-application-server/internal/logging"
	"github.com/brocaar/lorawan"
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

// GetDevice returns the device matching the given DevEUI.
func GetDevice(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) (Device, error) {
	var d Device
	err := sqlx.Get(db, &d, "select * from device where dev_eui = $1", devEUI[:])
	if err != nil {
		return d, handlePSQLError(err, "select error")
	}

	return d, nil
}
