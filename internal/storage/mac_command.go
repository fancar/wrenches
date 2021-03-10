package storage

import (
	"context"
	"fmt"
	// log "github.com/sirupsen/logrus"

	// "github.com/brocaar/chirpstack-network-server/internal/logging"
	"github.com/brocaar/lorawan"
)

const (
	macCommandQueueTempl   = "lora:ns:device:%s:mac:queue"
	macCommandPendingTempl = "lora:ns:device:%s:mac:pending:%d"
)

// MACCommandBlock defines a block of MAC commands that must be handled
// together.
type MACCommandBlock struct {
	CID         lorawan.CID
	External    bool // command was enqueued by an external service
	MACCommands MACCommands
}

// MACCommands holds a slice of MACCommand items.
type MACCommands []lorawan.MACCommand

// FlushMACCommandQueue flushes the mac-command queue for the given DevEUI.
func FlushMACCommandQueue(ctx context.Context, devEUI lorawan.EUI64) error {
	key := fmt.Sprintf(macCommandQueueTempl, devEUI)

	err := RedisClient().Del(key).Err()
	if err != nil {
		return fmt.Errorf("flush mac-command queue error: %w", err)
	}

	return nil
}
