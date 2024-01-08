package payment_controller

import (
	"bola-wa-service/model/payment_model"
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

// func SendNotificationToFieldMaster(ctx *gin.Context, client *whatsmeow.Client) {
// var PaymentReqBody payment_model.Payment
// if err := ctx.ShouldBindJSON(&PaymentReqBody); err != nil {
// 	fmt.Println(err)
// 	ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
// 	return
// }
// 	stringPhonenum := fmt.Sprintf("%s@s.whatsapp.net", PaymentReqBody.PhoneNumber)
// 	jid, err := types.ParseJID(stringPhonenum)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
// 		return
// 	}
// if PaymentReqBody.Note == "" {
// 	PaymentReqBody.Note = "-"
// }
// stringTemplate := fmt.Sprintf("Dear *%s*,\n"+
// 	"Kami senang untuk memberitahu Anda bahwa lapangan Anda telah berhasil dibooking oleh pelanggan kami. Berikut detail pemesanan:\n\n"+
// 	"Nama Pelanggan: *%s*\n"+
// 	"Nama Tempat: *%s*\n"+
// 	"Nama Lapangan: *%s*\n"+
// 	"Durasi Pertandingan: *%d Jam*\n"+
// 	"Waktu Mulai Pertandingan: *%s*\n"+
// 	"Waktu Selesai Pertandingan: *%s*\n"+
// 	"Olahraga: *%s*\n"+
// 	"Harga yang Dibayarkan: *%s*\n"+
// 	"Note: *%s*\n\n"+
// 	"Harap pastikan lapangan dalam kondisi yang baik untuk pertandingan tersebut. Pelanggan kami sangat menantikan pengalaman bermain yang menyenangkan di lapangan Anda.\n"+
// 	"Terima kasih atas kerja sama Anda, dan jangan ragu untuk menghubungi kami jika ada pertanyaan atau informasi tambahan yang diperlukan.\n\n"+
// 	"Salam hormat,\n"+
// 	"Admin Bola", PaymentReqBody.FieldMasterName, PaymentReqBody.CustomerName, PaymentReqBody.FieldName, PaymentReqBody.SubFieldName, PaymentReqBody.CountHours, PaymentReqBody.MatchStart, PaymentReqBody.MatchEnd, PaymentReqBody.CategoryField, PaymentReqBody.AmountFormatted, PaymentReqBody.Note)

// 	client.SendMessage(context.Background(), jid, &waProto.Message{
// 		Conversation: proto.String(stringTemplate),
// 	})
// 	ctx.JSON(http.StatusOK, gin.H{"message": "Success"})
// }

func SendNotificationToFieldMaster(ctx *gin.Context, client *whatsmeow.Client) {
	var PaymentReqBody payment_model.Payment
	if err := ctx.ShouldBindJSON(&PaymentReqBody); err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
		return
	}

	stringPhonenum := fmt.Sprintf("%s@s.whatsapp.net", PaymentReqBody.PhoneNumber)
	jid, err := types.ParseJID(stringPhonenum)
	if err != nil {
		fmt.Println("Error:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
		return
	}

	if PaymentReqBody.Note == "" {
		PaymentReqBody.Note = "-"
	}
	stringTemplate := fmt.Sprintf("Dear *%s*,\n"+
		"Kami senang untuk memberitahu Anda bahwa lapangan Anda telah berhasil dibooking oleh pelanggan kami. Berikut detail pemesanan:\n\n"+
		"Nama Pelanggan: *%s*\n"+
		"Nama Tempat: *%s*\n"+
		"Nama Lapangan: *%s*\n"+
		"Durasi Pertandingan: *%d Jam*\n"+
		"Waktu Mulai Pertandingan: *%s*\n"+
		"Waktu Selesai Pertandingan: *%s*\n"+
		"Olahraga: *%s*\n"+
		"Harga yang Dibayarkan: *%s*\n"+
		"Note: *%s*\n\n"+
		"Harap pastikan lapangan dalam kondisi yang baik untuk pertandingan tersebut. Pelanggan kami sangat menantikan pengalaman bermain yang menyenangkan di lapangan Anda.\n"+
		"Terima kasih atas kerja sama Anda, dan jangan ragu untuk menghubungi kami jika ada pertanyaan atau informasi tambahan yang diperlukan.\n\n"+
		"Salam hormat,\n"+
		"Admin Bola", PaymentReqBody.FieldMasterName, PaymentReqBody.CustomerName, PaymentReqBody.FieldName, PaymentReqBody.SubFieldName, PaymentReqBody.CountHours, PaymentReqBody.MatchStart, PaymentReqBody.MatchEnd, PaymentReqBody.CategoryField, PaymentReqBody.AmountFormatted, PaymentReqBody.Note)

	go func() {
		sendMessageToFieldMaster(ctx, client, jid, stringTemplate)
	}()

	ctx.JSON(http.StatusOK, gin.H{"message": "Success"})
}

func sendMessageToFieldMaster(ctx *gin.Context, client *whatsmeow.Client, jid types.JID, stringTemplate string) {

	if _, err := client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(stringTemplate),
	}); err != nil {
		fmt.Println("Error sending OTP message:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"Error": "Failed to send OTP message"})
	}
}

func SendNotificationToUserRefund(ctx *gin.Context, client *whatsmeow.Client) {
	var PaymentReqBody payment_model.Payment
	if err := ctx.ShouldBindJSON(&PaymentReqBody); err != nil {
		fmt.Println(err)
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
		return
	}

	stringPhonenum := fmt.Sprintf("%s@s.whatsapp.net", PaymentReqBody.PhoneNumber)
	jid, err := types.ParseJID(stringPhonenum)
	if err != nil {
		fmt.Println("Error:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
		return
	}

	if PaymentReqBody.Note == "" {
		PaymentReqBody.Note = "-"
	}

	cancelTemplate := fmt.Sprintf("Dear *%s*,\n"+
		"Kami mohon memberitahu bahwa pemesanan lapangan untuk pertandingan berikut ini telah dibatalkan:\n\n"+
		"Nama Tempat: *%s*\n"+
		"Nama Lapangan: *%s*\n"+
		"Durasi Pertandingan: *%d Jam*\n"+
		"Waktu Mulai Pertandingan: *%s*\n"+
		"Waktu Selesai Pertandingan: *%s*\n"+
		"Olahraga: *%s*\n"+
		"Harga yang Dikembalikan: *%s*\n"+
		"Alasan: *%s*\n\n"+
		"Kami memahami bahwa ini dapat menimbulkan ketidaknyamanan, dan kami meminta maaf atas ketidaknyamanan ini. Silakan menghubungi admin kami untuk informasi lebih lanjut mengenai pengembalian pembayaran.\n\n"+
		"Terima kasih atas pengertian Anda. Kami berharap dapat melayani Anda di lain waktu.\n\n"+
		"Salam hormat,\n"+
		"Admin Bola", PaymentReqBody.FieldMasterName, PaymentReqBody.FieldName, PaymentReqBody.SubFieldName, PaymentReqBody.CountHours, PaymentReqBody.MatchStart, PaymentReqBody.MatchEnd, PaymentReqBody.CategoryField, PaymentReqBody.AmountFormatted, PaymentReqBody.Note)

	go func() {
		sendMessageToUserRefund(ctx, client, jid, cancelTemplate)
	}()

	ctx.JSON(http.StatusOK, gin.H{"message": "Success"})
}

func sendMessageToUserRefund(ctx *gin.Context, client *whatsmeow.Client, jid types.JID, stringTemplate string) {

	if _, err := client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(stringTemplate),
	}); err != nil {
		fmt.Println("Error sending OTP message:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"Error": "Failed to send OTP message"})
	}
}
