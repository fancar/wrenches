package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	// log "github.com/sirupsen/logrus"

	"github.com/brocaar/lorawan"

	"github.com/brocaar/chirpstack-api/go/v3/ns"
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

// NSDeviceProfile defines the backend.DeviceProfile from Network-server
type NSDeviceProfile struct {
	CreatedAt          time.Time              `db:"created_at"`
	UpdatedAt          time.Time              `db:"updated_at"`
	ID                 uuid.UUID              `db:"device_profile_id"`
	SupportsClassB     bool                   `db:"supports_class_b"`
	ClassBTimeout      int                    `db:"class_b_timeout"` // Unit: seconds
	PingSlotPeriod     int                    `db:"ping_slot_period"`
	PingSlotDR         int                    `db:"ping_slot_dr"`
	PingSlotFreq       uint32                 `db:"ping_slot_freq"` // in Hz
	SupportsClassC     bool                   `db:"supports_class_c"`
	ClassCTimeout      int                    `db:"class_c_timeout"`      // Unit: seconds
	MACVersion         string                 `db:"mac_version"`          // Example: "1.0.2" [LW102]
	RegParamsRevision  string                 `db:"reg_params_revision"`  // Example: "B" [RP102B]
	RXDelay1           uint32                 `db:"rx_delay_1"`           // original param. Using for ABP init only
	RXDROffset1        uint32                 `db:"rx_dr_offset_1"`       // original param. Using for ABP init only
	RXDataRate2        uint32                 `db:"rx_data_rate_2"`       // original param. Using for ABP init only   Unit: bits-per-second
	RXFreq2            uint32                 `db:"rx_freq_2"`            // original param. Using for ABP init only   In Hz
	FactoryPresetFreqs []uint32               `db:"factory_preset_freqs"` // In Hz
	MaxEIRP            int                    `db:"max_eirp"`             // In dBm
	MaxDutyCycle       int                    `db:"max_duty_cycle"`       // Example: 10 indicates 10%
	SupportsJoin       bool                   `db:"supports_join"`
	RFRegion           string                 `db:"rf_region"`
	Supports32bitFCnt  bool                   `db:"supports_32bit_fcnt"`
	ADRAlgorithmID     string                 `db:"adr_algorithm_id"` // deprecated //TODO: remove
	JoinAcceptDelay1   uint32                 `db:"join_accept_delay1"`
	JoinAcceptDelay2   uint32                 `db:"join_accept_delay2"`
	FCntAutomaticReset bool                   `db:"fcnt_auto_reset"` // allows pass low fCnts for ABP devices
	CmdSwitches        []*ns.MacCommandSwitch `db:-`
	CmdsEnabled        string                 `db:"cmds_enabled"`
	RX1Delay           int                    `db:"rx1_delay"`
	RX1DROffset        int                    `db:"rx1_dr_offset"`
	RX2DataRate        int                    `db:"rx2_data_rate"` // Unit: bits-per-second
	RX2Freq            int                    `db:"rx2_freq"`      // In Hz
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
	err := sqlx.Get(db, &d, "select * FROM device WHERE dev_eui = $1", devEUI[:])
	if err != nil {
		return d, handlePSQLError(err, "select error")
	}

	return d, nil
}

// GetDeviceProfileFromNS returns the device profile matching
// func GetDeviceProfileFromNS(ctx context.Context, db sqlx.Queryer, uuid uuid.UUID) (NSDeviceProfile, error) {
// 	var d NSDeviceProfile
// 	err := sqlx.Get(db, &d, "SELECT * FROM device_profile WHERE device_profile_id = $1", uuid)
// 	if err != nil {
// 		return d, handlePSQLError(err, "select error")
// 	}

// 	return d, nil
// }

// GetLastDeviceActivation returns the last device activation event
func GetLastDeviceActivation(ctx context.Context, db sqlx.Queryer, devEUI lorawan.EUI64) (DeviceActivation, error) {
	var d DeviceActivation
	err := sqlx.Get(db, &d, "SELECT * FROM device_activation WHERE dev_eui = $1 ORDER BY created_at DESC LIMIT 1", devEUI)
	if err != nil {
		return d, handlePSQLError(err, "select error")
	}

	return d, nil
}

// GetDeviceProfile returns the device-profile matching the given id.
func GetDeviceProfileFromNS(ctx context.Context, db sqlx.Queryer, id uuid.UUID) (NSDeviceProfile, error) {
	var dp NSDeviceProfile

	row := db.QueryRowx(`
        select
            created_at,
            updated_at,

            device_profile_id,
            supports_class_b,
            class_b_timeout,
            ping_slot_period,
            ping_slot_dr,
            ping_slot_freq,
            supports_class_c,
            class_c_timeout,
            mac_version,
            reg_params_revision,
            rx_delay_1,
            rx_dr_offset_1,
            rx_data_rate_2,
            rx_freq_2,
            factory_preset_freqs,
            max_eirp,
            max_duty_cycle,
            supports_join,
            rf_region,
            supports_32bit_fcnt,
			adr_algorithm_id,
			join_accept_delay1,
			join_accept_delay2,
			fcnt_auto_reset,
			cmds_enabled,
		    rx1_delay,
            rx1_dr_offset,
            rx2_data_rate,
            rx2_freq
        from device_profile
        where
            device_profile_id = $1
        `, id)

	var factoryPresetFreqs []int64

	err := row.Scan(
		&dp.CreatedAt,
		&dp.UpdatedAt,
		&dp.ID,
		&dp.SupportsClassB,
		&dp.ClassBTimeout,
		&dp.PingSlotPeriod,
		&dp.PingSlotDR,
		&dp.PingSlotFreq,
		&dp.SupportsClassC,
		&dp.ClassCTimeout,
		&dp.MACVersion,
		&dp.RegParamsRevision,
		&dp.RXDelay1,
		&dp.RXDROffset1,
		&dp.RXDataRate2,
		&dp.RXFreq2,
		pq.Array(&factoryPresetFreqs),
		&dp.MaxEIRP,
		&dp.MaxDutyCycle,
		&dp.SupportsJoin,
		&dp.RFRegion,
		&dp.Supports32bitFCnt,
		&dp.ADRAlgorithmID,
		&dp.JoinAcceptDelay1,
		&dp.JoinAcceptDelay2,
		&dp.FCntAutomaticReset,
		&dp.CmdsEnabled,
		&dp.RX1Delay,
		&dp.RX1DROffset,
		&dp.RX2DataRate,
		&dp.RX2Freq,
	)
	if err != nil {
		return dp, handlePSQLError(err, "select error")
	}

	for _, f := range factoryPresetFreqs {
		dp.FactoryPresetFreqs = append(dp.FactoryPresetFreqs, uint32(f))
	}

	switches, err := getSwitchesByBitMask(dp.CmdsEnabled)
	if err != nil {
		return dp, err
	}

	dp.CmdSwitches = switches

	return dp, nil
}

func getSwitchesByBitMask(s string) ([]*ns.MacCommandSwitch, error) {
	if len(s) != 128 {
		return nil, fmt.Errorf("storage:getSwitchesByBitMask: expected bitmask len is 128. Got %d", len(s))
	}

	result := []*ns.MacCommandSwitch{}
	runes := []rune(s)

	for i, r := range runes {
		res := ns.MacCommandSwitch{
			Cid: uint32(i + 1),
		}
		if r == 49 { // rune == 49 means enabled: string(r) == "1"
			res.Enabled = true
		}

		result = append(result, &res)
	}
	return result, nil
}
