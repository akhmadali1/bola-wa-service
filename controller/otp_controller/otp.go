package otp_controller

import (
	"bola-wa-service/model/otp_model"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

func SendOTP(w http.ResponseWriter, r *http.Request, client *whatsmeow.Client) {
	var OTPCredentials otp_model.OTPModel
	if err := json.NewDecoder(r.Body).Decode(&OTPCredentials); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stringPhonenum := fmt.Sprintf("%s@s.whatsapp.net", OTPCredentials.PhoneNumber)
	jid, err := types.ParseJID(stringPhonenum)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	OTPCode, err := GenerateOTPCode(6)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	go sendOTPMessage(client, jid, OTPCode)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"otp": OTPCode})
}

func sendOTPMessage(client *whatsmeow.Client, jid types.JID, otpCode string) {
	stringOTP := fmt.Sprintf("JANGAN MEMBERITAHU KODE RAHASIA INI KE SIAPAPUN termasuk admin BOLA. WASPADA TERHADAP KASUS PENIPUAN! KODE VERIFIKASI: %s", otpCode)

	if _, err := client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(stringOTP),
	}); err != nil {
		fmt.Println("Error sending OTP message:", err)
	}
}

func GenerateOTPCode(length int) (string, error) {
	seed := "0123456789"
	byteSlice := make([]byte, length)

	for i := 0; i < length; i++ {
		max := big.NewInt(int64(len(seed)))
		num, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}

		byteSlice[i] = seed[num.Int64()]
	}

	return string(byteSlice), nil
}
