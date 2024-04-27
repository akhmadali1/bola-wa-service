package payment_controller

import (
	"bola-wa-service/model/payment_model"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/binary/proto"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type Payment struct {
	payment_model.Payment
	NotificationSent bool // Indicates whether the notification has been sent
}

var notificationLock sync.Mutex // Mutex to synchronize access to the notification status

func SendNotificationToFieldMaster(ctx *gin.Context, client *whatsmeow.Client) {
	var PaymentReqBody Payment
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

	// Parse MatchStart string to time.Time
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		fmt.Println("Error loading location:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"Error": "Failed to load location"})
		return
	}

	// Parse MatchStart string in Jakarta timezone
	matchStartTime, err := time.ParseInLocation("Monday, 02 Jan 2006 15:04:05", PaymentReqBody.MatchStart, loc)
	if err != nil {
		fmt.Println("Error parsing MatchStart time:", err)
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
		return
	}

	// Calculate the duration until 1 hour before matchStartTime
	notificationTime := matchStartTime.Add(-time.Hour)

	go func() {
		sendMessageToFieldMaster(ctx, client, jid, stringTemplate)
	}()

	go func() {
		for !PaymentReqBody.NotificationSent {
			// Check if the current time is after the notification time
			if time.Now().After(notificationTime) {
				// Send the notification
				stringCustomerPhonenum := fmt.Sprintf("%s@s.whatsapp.net", PaymentReqBody.CustomerPhoneNumber)
				jidCustomer, err := types.ParseJID(stringCustomerPhonenum)
				if err != nil {
					fmt.Println("Error:", err)
					ctx.JSON(http.StatusBadRequest, gin.H{"Error": err})
					return
				}

				reminderTemplate := fmt.Sprintf("Reminder: Pertandingan di lapangan *%s* akan dimulai dalam +- 1 jam.", PaymentReqBody.FieldName)
				sendMessageToFieldMaster(ctx, client, jidCustomer, reminderTemplate)
				// Update the notification status after sending the notification
				PaymentReqBody.NotificationSent = true
			}
			// Sleep for some time before checking again
			time.Sleep(time.Second)
		}
	}()

	ctx.JSON(http.StatusOK, gin.H{"message": "Success"})
}

func SendNotificationToUserRefund(ctx *gin.Context, client *whatsmeow.Client) {
	var PaymentReqBody Payment
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

	// Check if the notification has already been sent
	notificationLock.Lock()
	if !PaymentReqBody.NotificationSent {
		go func() {
			sendMessageToUserRefund(ctx, client, jid, cancelTemplate)
			// Update the notification status after sending the notification
			PaymentReqBody.NotificationSent = true
		}()
	}
	notificationLock.Unlock()

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

func sendMessageToUserRefund(ctx *gin.Context, client *whatsmeow.Client, jid types.JID, stringTemplate string) {

	if _, err := client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(stringTemplate),
	}); err != nil {
		fmt.Println("Error sending OTP message:", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"Error": "Failed to send OTP message"})
	}
}
