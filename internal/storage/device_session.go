//go:generate protoc -I=/tmp/chirpstack-api/protobuf -I=. --go_out=. device_session.proto

package storage

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/gofrs/uuid"
	proto "github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"

	"github.com/brocaar/lorawan"
	loraband "github.com/brocaar/lorawan/band"
	"github.com/fancar/wrenches/internal/config"
)

const (
	devAddrKeyTempl                = "lora:ns:devaddr:%s"     // contains a set of DevEUIs using this DevAddr
	deviceSessionKeyTempl          = "lora:ns:%s:device:%s"   // contains the session of a DevEUI
	deviceGatewayRXInfoSetKeyTempl = "lora:ns:device:%s:gwrx" // contains gateway meta-data from the last uplink
)

// UplinkHistorySize contains the number of frames to store
const UplinkHistorySize = 20

// RXWindow defines the RX window option.
type RXWindow int8

// Available RX window options.
const (
	RX1 = iota
	RX2
)

// DeviceGatewayRXInfoSet contains the rx-info set of the receiving gateways
// for the last uplink.
type DeviceGatewayRXInfoSet struct {
	DevEUI lorawan.EUI64
	DR     int
	Items  []DeviceGatewayRXInfo
}

// DeviceGatewayRXInfo holds the meta-data of a gateway receiving the last
// uplink message.
type DeviceGatewayRXInfo struct {
	GatewayID lorawan.EUI64
	RSSI      int
	LoRaSNR   float64
	Antenna   uint32
	Board     uint32
	Context   []byte
}

// UplinkHistory contains the meta-data of an uplink transmission.
type UplinkHistory struct {
	FCnt         uint32
	MaxSNR       float64
	MaxRSSI      int32
	TXPowerIndex int
	GatewayCount int
}

// KeyEnvelope defined a key-envelope.
// type KeyEnvelope struct {
// 	KEKLabel string
// 	AESKey   []byte
// }

// DeviceSession defines a device-session.
type DeviceSession struct {
	// MAC version
	MACVersion string

	// profile ids
	DeviceProfileID  uuid.UUID
	ServiceProfileID uuid.UUID
	RoutingProfileID uuid.UUID

	// session data
	DevAddr        lorawan.DevAddr
	DevEUI         lorawan.EUI64
	JoinEUI        lorawan.EUI64
	FNwkSIntKey    lorawan.AES128Key
	SNwkSIntKey    lorawan.AES128Key
	NwkSEncKey     lorawan.AES128Key
	AppSKey        lorawan.AES128Key // mod. We decrypt data localy to put rx unencrypted payload to stream
	AppSKeyOnAS    lorawan.AES128Key // temp. As we moved it to NS
	NoAppSKey      bool              // some clients encrypt messages byself. So we skip encryption
	AppSKeyEvelope *KeyEnvelope
	FCntUp         uint32
	NFCntDown      uint32
	AFCntDown      uint32
	ConfFCnt       uint32

	// Only used by ABP activation
	SkipFCntValidation bool // deprecated. The flag had been moved to DeviceProfile
	FCntAutomaticReset bool // deprecated.

	RXWindow     RXWindow
	RXDelay      uint8
	RX1DROffset  uint8
	RX2DR        uint8
	RX2Frequency uint32

	// TXPowerIndex which the node is using. The possible values are defined
	// by the lorawan/band package and are region specific. By default it is
	// assumed that the node is using TXPower 0. This value is controlled by
	// the ADR engine.
	TXPowerIndex int

	// DR defines the (last known) data-rate at which the node is operating.
	// This value is controlled by the ADR engine.
	DR int

	// ADR defines if the device has ADR enabled.
	ADR bool

	// MinSupportedTXPowerIndex defines the minimum supported tx-power index
	// by the node (default 0).
	MinSupportedTXPowerIndex int

	// MaxSupportedTXPowerIndex defines the maximum supported tx-power index
	// by the node, or 0 when not set.
	MaxSupportedTXPowerIndex int

	// NbTrans defines the number of transmissions for each unconfirmed uplink
	// frame. In case of 0, the default value is used.
	// This value is controlled by the ADR engine.
	NbTrans uint8

	EnabledChannels       []int                    // deprecated, migrated by GetDeviceSession
	EnabledUplinkChannels []int                    // channels that are activated on the node
	ExtraUplinkChannels   map[int]loraband.Channel // extra uplink channels, configured by the user
	ChannelFrequencies    []uint32                 // frequency of each channel
	UplinkHistory         []UplinkHistory          // contains the last 20 transmissions

	// LastDevStatusRequest contains the timestamp when the last device-status
	// request was made.
	LastDevStatusRequested time.Time

	// Class-B related configuration.
	BeaconLocked      bool
	PingSlotNb        int
	PingSlotDR        int
	PingSlotFrequency uint32

	// RejoinRequestEnabled defines if the rejoin-request is enabled on the
	// device.
	RejoinRequestEnabled bool

	// RejoinRequestMaxCountN defines the 2^(C+4) uplink message interval for
	// the rejoin-request.
	RejoinRequestMaxCountN int

	// RejoinRequestMaxTimeN defines the 2^(T+10) time interval (seconds)
	// for the rejoin-request.
	RejoinRequestMaxTimeN int

	RejoinCount0               uint16
	PendingRejoinDeviceSession *DeviceSession

	// ReferenceAltitude holds the device reference altitude used for
	// geolocation.
	ReferenceAltitude float64

	// Uplink and Downlink dwell time limitations.
	UplinkDwellTime400ms   bool
	DownlinkDwellTime400ms bool

	// Max uplink EIRP limitation.
	UplinkMaxEIRPIndex uint8

	// Delayed mac-commands.
	MACCommandErrorCount map[lorawan.CID]int

	// Device is disabled.
	IsDisabled bool

	// Current Packet Error Rate.
	Per float64
}

// GetMACVersion returns the LoRaWAN mac version.
func (s DeviceSession) GetMACVersion() lorawan.MACVersion {
	if strings.HasPrefix(s.MACVersion, "1.1") {
		return lorawan.LoRaWAN1_1
	}

	return lorawan.LoRaWAN1_0
}

// DeviceSessionCSV defines a device-session in comma sep. format.
type DeviceSessionCSV struct {
	MACVersion string `csv:"MACVersion"`
	DevEUI     string `csv:"DevEUI"`
	DevAddr    string `csv:"DevAddr"`
	JoinEUI    string `csv:"JoinEUI"`

	TXPowerIndex uint32 `csv:"TXPowerIndex"`
	NbTrans      uint32 `csv:"NbTrans"`
	DR           uint32 `csv:"DR"`

	FCntUp      uint32 `csv:"FCntUp"`
	NFCntDown   uint32 `csv:"NFCntDown"`
	AFCntDown   uint32 `csv:"AFCntDown"`
	ConfFCnt    uint32 `csv:"ConfFCnt"`
	FNwkSIntKey string `csv:"FNwkSIntKey"` //lorawan.AES128Key
	SNwkSIntKey string `csv:"SNwkSIntKey"` //lorawan.AES128Key
	NwkSEncKey  string `csv:"NwkSEncKey"`  //lorawan.AES128Key

	AESKey   string `csv:"AESKey"` // it equals to AppSKey in case the KEKLabel is
	KEKLabel string `csv:"KEKLabel"`

	PingSlotNb            int                      `csv:"PingSlotNb"`
	EnabledUplinkChannels []int                    `csv:"EnabledUplinkChannels"`
	ExtraUplinkChannels   map[int]loraband.Channel `csv:"ExtraUplinkChannels"` // extra uplink channels, configured by the user

	IsDisabled bool `csv:"IsDisabled"`

	RXWindow     uint32 `csv:"RXWindow"`
	RXDelay      uint32 `csv:"RXDelay"`
	RX1DROffset  uint32 `csv:"RX1DROffset"`
	RX2Frequency uint32 `csv:"RX2Frequency"`
	RX2DR        uint32 `csv:"RX2DR"`
}

// GetDeviceSession returns the device-session for the given DevEUI.
func GetDeviceSession(ctx context.Context, client redis.UniversalClient, devEUI lorawan.EUI64) (*DeviceSession, error) {
	key := fmt.Sprintf(deviceSessionKeyTempl, config.Get().NetworkServer.Band.Name, devEUI)
	var dsPB DeviceSessionPB

	val, err := client.Get(key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return &DeviceSession{}, ErrDoesNotExist
		}
		return &DeviceSession{}, fmt.Errorf("get error %w", err)
	}

	err = proto.Unmarshal(val, &dsPB)
	if err != nil {
		return &DeviceSession{}, fmt.Errorf("unmarshal protobuf error %w", err)
	}

	return deviceSessionFromPB(&dsPB), nil
}

// SaveDeviceSession saves the device-session. In case it doesn't exist yet
// it will be created.
func SaveDeviceSession(ctx context.Context, s DeviceSession) error {
	devAddrKey := fmt.Sprintf(devAddrKeyTempl, s.DevAddr)
	devSessKey := fmt.Sprintf(deviceSessionKeyTempl, config.Get().NetworkServer.Band.Name, s.DevEUI)

	dsPB := deviceSessionToPB(s)
	b, err := proto.Marshal(dsPB)
	if err != nil {
		return fmt.Errorf("protobuf encode error: %w", err)
	}

	// Note that we must execute the DevAddr set related operations in multiple
	// tx pipelines in order to support Redis Cluster. It can not be guaranteed
	// that devAddrKey, pendingDevAddrKey and DevSessKey are on the same Cluster
	// shard.
	// fmt.Println("devAddrKey:", devAddrKey, s.DevEUI[:], deviceSessionTTL)

	pipe := RedisClient().TxPipeline()
	pipe.SAdd(devAddrKey, s.DevEUI[:])
	pipe.PExpire(devAddrKey, deviceSessionTTL)
	if _, err := pipe.Exec(); err != nil {
		return fmt.Errorf("exec error: %w", err)
	}

	if s.PendingRejoinDeviceSession != nil {
		pendingDevAddrKey := fmt.Sprintf(devAddrKeyTempl, s.PendingRejoinDeviceSession.DevAddr)

		pipe = RedisClient().TxPipeline()
		pipe.SAdd(pendingDevAddrKey, s.DevEUI[:])
		pipe.PExpire(pendingDevAddrKey, deviceSessionTTL)
		if _, err := pipe.Exec(); err != nil {
			return fmt.Errorf("exec error: %w", err)
		}
	}

	err = RedisClient().Set(devSessKey, b, deviceSessionTTL).Err()
	if err != nil {
		return fmt.Errorf("set error: %w", err)
	}

	log.WithFields(log.Fields{
		"dev_eui":  s.DevEUI,
		"dev_addr": s.DevAddr,
		// "ctx_id":   ctx.Value(logging.ContextIDKey),
	}).Info("device-session saved")

	return nil
}

// CSVfromDeviceSession converter
func CSVfromDeviceSession(d *DeviceSession) *DeviceSessionCSV {
	result := DeviceSessionCSV{
		MACVersion: d.MACVersion,

		DevEUI:  hex.EncodeToString(d.DevEUI[:]),
		DevAddr: hex.EncodeToString(d.DevAddr[:]),
		JoinEUI: hex.EncodeToString(d.JoinEUI[:]),

		TXPowerIndex: uint32(d.TXPowerIndex),
		NbTrans:      uint32(d.NbTrans),
		DR:           uint32(d.DR),

		FCntUp:      d.FCntUp,
		NFCntDown:   d.NFCntDown,
		AFCntDown:   d.AFCntDown,
		ConfFCnt:    d.ConfFCnt,
		FNwkSIntKey: hex.EncodeToString(d.FNwkSIntKey[:]),
		SNwkSIntKey: hex.EncodeToString(d.FNwkSIntKey[:]),
		NwkSEncKey:  hex.EncodeToString(d.FNwkSIntKey[:]),
		AESKey:      hex.EncodeToString(d.AppSKey[:]),

		PingSlotNb:            d.PingSlotNb,
		EnabledUplinkChannels: d.EnabledUplinkChannels,
		IsDisabled:            d.IsDisabled,

		RXWindow:     uint32(d.RXWindow),
		RXDelay:      uint32(d.RXDelay),
		RX1DROffset:  uint32(d.RX1DROffset),
		RX2Frequency: d.RX2Frequency,
		RX2DR:        uint32(d.RX2DR),
	}

	return &result
}

// ConvertDeviceSessionsToCSV converter
func ConvertDeviceSessionsToCSV(input []DeviceSession) ([]DeviceSessionCSV, error) {
	var result []DeviceSessionCSV
	for _, s := range input {
		l := CSVfromDeviceSession(&s)
		result = append(result, *l)
	}
	return result, nil
}

func deviceSessionToPB(d DeviceSession) *DeviceSessionPB {
	out := DeviceSessionPB{
		MacVersion: d.MACVersion,

		DeviceProfileId:  d.DeviceProfileID.String(),
		ServiceProfileId: d.ServiceProfileID.String(),
		RoutingProfileId: d.RoutingProfileID.String(),

		DevAddr:     d.DevAddr[:],
		DevEui:      d.DevEUI[:],
		JoinEui:     d.JoinEUI[:],
		FNwkSIntKey: d.FNwkSIntKey[:],
		SNwkSIntKey: d.SNwkSIntKey[:],
		NwkSEncKey:  d.NwkSEncKey[:],
		AppSKey:     d.AppSKey[:],
		NoAppSKey:   d.NoAppSKey,

		FCntUp:        d.FCntUp,
		NFCntDown:     d.NFCntDown,
		AFCntDown:     d.AFCntDown,
		ConfFCnt:      d.ConfFCnt,
		SkipFCntCheck: d.SkipFCntValidation,

		RxDelay:      uint32(d.RXDelay),
		Rx1DrOffset:  uint32(d.RX1DROffset),
		Rx2Dr:        uint32(d.RX2DR),
		Rx2Frequency: d.RX2Frequency,
		TxPowerIndex: uint32(d.TXPowerIndex),

		Dr:                       uint32(d.DR),
		Adr:                      d.ADR,
		MinSupportedTxPowerIndex: uint32(d.MinSupportedTXPowerIndex),
		MaxSupportedTxPowerIndex: uint32(d.MaxSupportedTXPowerIndex),
		NbTrans:                  uint32(d.NbTrans),

		ExtraUplinkChannels: make(map[uint32]*DeviceSessionPBChannel),

		LastDeviceStatusRequestTimeUnixNs: d.LastDevStatusRequested.UnixNano(),

		BeaconLocked:      d.BeaconLocked,
		PingSlotNb:        uint32(d.PingSlotNb),
		PingSlotDr:        uint32(d.PingSlotDR),
		PingSlotFrequency: d.PingSlotFrequency,

		RejoinRequestEnabled:   d.RejoinRequestEnabled,
		RejoinRequestMaxCountN: uint32(d.RejoinRequestMaxCountN),
		RejoinRequestMaxTimeN:  uint32(d.RejoinRequestMaxTimeN),

		RejoinCount_0:     uint32(d.RejoinCount0),
		ReferenceAltitude: d.ReferenceAltitude,

		UplinkDwellTime_400Ms:   d.UplinkDwellTime400ms,
		DownlinkDwellTime_400Ms: d.DownlinkDwellTime400ms,
		UplinkMaxEirpIndex:      uint32(d.UplinkMaxEIRPIndex),

		MacCommandErrorCount: make(map[uint32]uint32),

		IsDisabled:         d.IsDisabled,
		FCntAutomaticReset: d.FCntAutomaticReset,
	}

	if d.AppSKeyEvelope != nil {
		out.AppSKeyEnvelope = &KeyEnvelope{
			KekLabel: d.AppSKeyEvelope.KekLabel,
			AesKey:   d.AppSKeyEvelope.AesKey,
		}
	}

	for _, c := range d.EnabledUplinkChannels {
		out.EnabledUplinkChannels = append(out.EnabledUplinkChannels, uint32(c))
	}

	for i, c := range d.ExtraUplinkChannels {
		out.ExtraUplinkChannels[uint32(i)] = &DeviceSessionPBChannel{
			Frequency: uint32(c.Frequency),
			MinDr:     uint32(c.MinDR),
			MaxDr:     uint32(c.MaxDR),
		}
	}

	for _, c := range d.ChannelFrequencies {
		out.ChannelFrequencies = append(out.ChannelFrequencies, uint32(c))
	}

	for _, h := range d.UplinkHistory {
		out.UplinkAdrHistory = append(out.UplinkAdrHistory, &DeviceSessionPBUplinkADRHistory{
			FCnt:         h.FCnt,
			MaxSnr:       float32(h.MaxSNR),
			TxPowerIndex: uint32(h.TXPowerIndex),
			GatewayCount: uint32(h.GatewayCount),
			MaxRssi:      h.MaxRSSI,
		})
	}

	if d.PendingRejoinDeviceSession != nil {
		dsPB := deviceSessionToPB(*d.PendingRejoinDeviceSession)
		b, err := proto.Marshal(dsPB)
		if err != nil {
			log.WithField("dev_eui", d.DevEUI).WithError(err).Error("protobuf encode error")
		}

		out.PendingRejoinDeviceSession = b
	}

	for k, v := range d.MACCommandErrorCount {
		out.MacCommandErrorCount[uint32(k)] = uint32(v)
	}

	return &out
}

func deviceSessionFromPB(d *DeviceSessionPB) *DeviceSession {
	dpID, _ := uuid.FromString(d.DeviceProfileId)
	rpID, _ := uuid.FromString(d.RoutingProfileId)
	spID, _ := uuid.FromString(d.ServiceProfileId)

	out := &DeviceSession{
		MACVersion: d.MacVersion,

		DeviceProfileID:  dpID,
		ServiceProfileID: spID,
		RoutingProfileID: rpID,

		FCntUp:             d.FCntUp,
		NFCntDown:          d.NFCntDown,
		AFCntDown:          d.AFCntDown,
		ConfFCnt:           d.ConfFCnt,
		SkipFCntValidation: d.SkipFCntCheck,

		RXDelay:      uint8(d.RxDelay),
		RX1DROffset:  uint8(d.Rx1DrOffset),
		RX2DR:        uint8(d.Rx2Dr),
		RX2Frequency: d.Rx2Frequency,
		TXPowerIndex: int(d.TxPowerIndex),

		DR:                       int(d.Dr),
		ADR:                      d.Adr,
		MinSupportedTXPowerIndex: int(d.MinSupportedTxPowerIndex),
		MaxSupportedTXPowerIndex: int(d.MaxSupportedTxPowerIndex),
		NbTrans:                  uint8(d.NbTrans),

		ExtraUplinkChannels: make(map[int]loraband.Channel),

		BeaconLocked:      d.BeaconLocked,
		PingSlotNb:        int(d.PingSlotNb),
		PingSlotDR:        int(d.PingSlotDr),
		PingSlotFrequency: d.PingSlotFrequency,

		RejoinRequestEnabled:   d.RejoinRequestEnabled,
		RejoinRequestMaxCountN: int(d.RejoinRequestMaxCountN),
		RejoinRequestMaxTimeN:  int(d.RejoinRequestMaxTimeN),

		RejoinCount0:      uint16(d.RejoinCount_0),
		ReferenceAltitude: d.ReferenceAltitude,

		UplinkDwellTime400ms:   d.UplinkDwellTime_400Ms,
		DownlinkDwellTime400ms: d.DownlinkDwellTime_400Ms,
		UplinkMaxEIRPIndex:     uint8(d.UplinkMaxEirpIndex),

		MACCommandErrorCount: make(map[lorawan.CID]int),

		IsDisabled:         d.IsDisabled,
		FCntAutomaticReset: d.FCntAutomaticReset,
		NoAppSKey:          d.NoAppSKey,
	}

	if d.LastDeviceStatusRequestTimeUnixNs > 0 {
		out.LastDevStatusRequested = time.Unix(0, d.LastDeviceStatusRequestTimeUnixNs)
	}

	copy(out.DevAddr[:], d.DevAddr)
	copy(out.DevEUI[:], d.DevEui)
	copy(out.JoinEUI[:], d.JoinEui)
	copy(out.FNwkSIntKey[:], d.FNwkSIntKey)
	copy(out.SNwkSIntKey[:], d.SNwkSIntKey)
	copy(out.NwkSEncKey[:], d.NwkSEncKey)
	copy(out.AppSKey[:], d.AppSKey)

	if d.AppSKeyEnvelope != nil {
		out.AppSKeyEvelope = &KeyEnvelope{
			KekLabel: d.AppSKeyEnvelope.KekLabel,
			AesKey:   d.AppSKeyEnvelope.AesKey,
		}
	}

	for _, c := range d.EnabledUplinkChannels {
		out.EnabledUplinkChannels = append(out.EnabledUplinkChannels, int(c))
	}

	for i, c := range d.ExtraUplinkChannels {
		out.ExtraUplinkChannels[int(i)] = loraband.Channel{
			Frequency: c.Frequency,
			MinDR:     int(c.MinDr),
			MaxDR:     int(c.MaxDr),
		}
	}

	for _, c := range d.ChannelFrequencies {
		out.ChannelFrequencies = append(out.ChannelFrequencies, c)
	}

	for _, h := range d.UplinkAdrHistory {
		out.UplinkHistory = append(out.UplinkHistory, UplinkHistory{
			FCnt:         h.FCnt,
			MaxSNR:       float64(h.MaxSnr),
			TXPowerIndex: int(h.TxPowerIndex),
			GatewayCount: int(h.GatewayCount),
			MaxRSSI:      h.MaxRssi,
		})
	}

	if len(d.PendingRejoinDeviceSession) != 0 {
		var dsPB DeviceSessionPB
		if err := proto.Unmarshal(d.PendingRejoinDeviceSession, &dsPB); err != nil {
			log.WithField("dev_eui", out.DevEUI).WithError(err).Error("decode pending rejoin device-session error")
		} else {
			ds := deviceSessionFromPB(&dsPB)
			out.PendingRejoinDeviceSession = ds
		}
	}

	for k, v := range d.MacCommandErrorCount {
		out.MACCommandErrorCount[lorawan.CID(k)] = int(v)
	}

	return out
}
