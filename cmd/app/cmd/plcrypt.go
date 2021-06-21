package cmd

import (
	"encoding/hex"
	"fmt"
	"github.com/brocaar/lorawan"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// var plCryptData string         // data to encrypt or decrypt
var plCryptAppSessonKey string // Application Session Key
var plCryptDevAddr string
var plCryptFCnt uint32
var plCryptDecrypt bool // if true - decrypt, otherwise encrypt

var plCryptCmd = &cobra.Command{
	Use:   "plcrypt",
	Short: "encrypt or decrypt data payload. Use 'plcrypt -h' for details",
	Long: `
	for example:
		to encrypt 0011 with fCount 3 for DevAddr 006a0f43:
			plcrypt -a 006a0f43 -f 4 -s 49e319a5a5f18aeaf0dcea2904ebe58b 0011

		to decrypt:
		plcrypt -d -a 006a0f43 -f 11 -s 49e319a5a5f18aeaf0dcea2904ebe58b af
	`,
	Args: cobra.MinimumNArgs(1),
	Run:  plCrypt,
}

func plCrypt(cmd *cobra.Command, args []string) {
	var AppSKey lorawan.AES128Key
	var DevAddr lorawan.DevAddr
	job := "Encrypted"

	if plCryptDecrypt {
		job = "Decrypted"
	}

	key, err := hex.DecodeString(plCryptAppSessonKey)
	if err != nil {
		log.WithError(err).Error("Can't convert the Session Key")
		return
	}
	devaddr, err := hex.DecodeString(plCryptDevAddr)
	if err != nil {
		log.WithError(err).Error("Can't convert DevAddr")
		return
	}
	data, err := hex.DecodeString(args[0])
	if err != nil {
		log.WithError(err).Error("Can't convert data to byte array")
		return
	}

	copy(AppSKey[:], key)
	copy(DevAddr[:], devaddr)

	result, err := lorawan.EncryptFRMPayload(AppSKey, plCryptDecrypt, DevAddr, plCryptFCnt, data)
	if err != nil {
		msg := "Can't encrypt data"
		if plCryptDecrypt {
			msg = "Can't decrypt data"
		}
		log.WithError(err).Error(msg)
		return
	}
	fmt.Printf("Input data: %s \n", args[0])
	fmt.Printf("%s RESULT : %s \n", job, hex.EncodeToString(result))
}
