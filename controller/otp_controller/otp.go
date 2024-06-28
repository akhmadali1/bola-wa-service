package otp_controller

import (
	"bola-wa-service/model/otp_model"
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// func SendOTP(ctx *gin.Context, client *whatsmeow.Client) {
// 	var OTPCredentials otp_model.OTPModel
// 	if err := ctx.ShouldBindJSON(&OTPCredentials); err != nil {
// 		fmt.Println(err)
// 		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
// 		return
// 	}
// 	stringPhonenum := fmt.Sprintf("%s@s.whatsapp.net", OTPCredentials.PhoneNumber)
// 	jid, err := types.ParseJID(stringPhonenum)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
// 		return
// 	}
// 	OTPCode, err := GenerateOTPCode(6)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
// 		return
// 	}
// 	stringOTP := fmt.Sprintf("JANGAN MEMBERITAHU KODE RAHASIA INI KE SIAPAPUN termasuk admin BOLA. WASPADA TERHADAP KASUS PENIPUAN! KODE VERIFIKASI untuk masuk: %s", OTPCode)
// 	client.SendMessage(context.Background(), jid, &waProto.Message{
// 		Conversation: proto.String(stringOTP),
// 	})
// 	ctx.JSON(http.StatusOK, gin.H{"otp": OTPCode})
// }

func SendOTP(ctx *gin.Context, client *whatsmeow.Client) {
	var OTPCredentials otp_model.OTPModel
	if err := ctx.ShouldBindJSON(&OTPCredentials); err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
		return
	}

	stringPhonenum := fmt.Sprintf("%s@s.whatsapp.net", OTPCredentials.PhoneNumber)
	jid, err := types.ParseJID(stringPhonenum)
	if err != nil {
		fmt.Println("Error:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
		return
	}

	OTPCode, err := GenerateOTPCode(6)
	if err != nil {
		fmt.Println("Error:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
		return
	}

	go func() {
		sendOTPMessage(ctx, client, jid, OTPCode)
	}()

	ctx.JSON(http.StatusOK, gin.H{"otp": OTPCode})
}

func sendOTPMessage(ctx *gin.Context, client *whatsmeow.Client, jid types.JID, otpCode string) {
	stringOTP := fmt.Sprintf("JANGAN MEMBERITAHU KODE RAHASIA INI KE SIAPAPUN termasuk admin BOLA. WASPADA TERHADAP KASUS PENIPUAN! KODE VERIFIKASI: %s", otpCode)

	if _, err := client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(stringOTP),
	}); err != nil {
		fmt.Println("Error sending OTP message:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"Error": "Failed to send OTP message"})
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
