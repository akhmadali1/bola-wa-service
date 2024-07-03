package payment_controller

import (
	"bola-wa-service/model/payment_model"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type Payment struct {
	payment_model.Payment
	NotificationSent bool // Indicates whether the notification has been sent
}

func SendNotificationToFieldMaster(w http.ResponseWriter, r *http.Request, client *whatsmeow.Client, cronScheduler *cron.Cron, reminderMap map[string]cron.EntryID) {
	var PaymentReqBody Payment
	if err := json.NewDecoder(r.Body).Decode(&PaymentReqBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stringPhonenum := fmt.Sprintf("%s@s.whatsapp.net", PaymentReqBody.PhoneNumber)
	jid, err := types.ParseJID(stringPhonenum)
	if err != nil {
		fmt.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse MatchStart string in Jakarta timezone
	matchStartTime, err := time.ParseInLocation("Monday, 02 Jan 2006 15:04:05", PaymentReqBody.MatchStart, loc)
	if err != nil {
		fmt.Println("Error parsing MatchStart time:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Calculate the duration until 1 hour before matchStartTime
	notificationTime := matchStartTime.Add(-time.Hour)

	sendMessageToFieldMaster(w, client, jid, stringTemplate)

	if time.Now().After(notificationTime) {
		notificationTime = time.Now().In(loc).Add(time.Minute)
	}

	notificationTimeSplit := strings.Split(notificationTime.String(), " ")
	getDateAndHours := notificationTimeSplit[0] + notificationTimeSplit[1]

	if entryID, ok := reminderMap[getDateAndHours+"@"+PaymentReqBody.CustomerPhoneNumber]; ok {
		cronScheduler.Remove(entryID)
	}

	cronExpression := createCronExpression(notificationTime)

	entryID, err := cronScheduler.AddFunc(cronExpression, func() {
		// Send email notification
		stringCustomerPhonenum := fmt.Sprintf("%s@s.whatsapp.net", PaymentReqBody.CustomerPhoneNumber)
		jidCustomer, err := types.ParseJID(stringCustomerPhonenum)
		if err != nil {
			fmt.Println("Error:", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		reminderTemplate := fmt.Sprintf("Reminder: Pertandingan di lapangan *%s* akan dimulai dalam +- 1 jam.", PaymentReqBody.FieldName)
		sendMessageToFieldMaster(w, client, jidCustomer, reminderTemplate)

		PaymentReqBody.NotificationSent = true
	})

	if err != nil {
		fmt.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reminderMap[getDateAndHours+"@"+PaymentReqBody.CustomerPhoneNumber] = entryID

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Success"})
}

func DeleteUnusedCronReminders(w http.ResponseWriter, r *http.Request, cronScheduler *cron.Cron, reminderMap map[string]cron.EntryID) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		fmt.Println("Error loading location:", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now().In(loc)
	var notificationTimeArray []string
	for notificationTime, entryID := range reminderMap {
		notificationTimeSplit := strings.Split(notificationTime, "@")
		fmt.Println(notificationTimeSplit)
		matchTime, err := time.ParseInLocation("2006-01-0215:04:05.999999999", notificationTimeSplit[0], loc)
		if err != nil {
			fmt.Println("Error parsing Match time:", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if now.After(matchTime) {
			cronScheduler.Remove(entryID)
			notificationTimeArray = append(notificationTimeArray, notificationTime)
			delete(reminderMap, notificationTime)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string][]string{"Time": notificationTimeArray})
}

func createCronExpression(t time.Time) string {
	// Format the time to the cron syntax: Minute Hour DayOfMonth Month DayOfWeek
	return fmt.Sprintf("%d %d %d %d *", t.Minute(), t.Hour(), t.Day(), int(t.Month()))
}

func SendNotificationToUserRefund(w http.ResponseWriter, r *http.Request, client *whatsmeow.Client) {
	var PaymentReqBody Payment
	if err := json.NewDecoder(r.Body).Decode(&PaymentReqBody); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stringPhonenum := fmt.Sprintf("%s@s.whatsapp.net", PaymentReqBody.PhoneNumber)
	jid, err := types.ParseJID(stringPhonenum)
	if err != nil {
		fmt.Println("Error:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	sendMessageToUserRefund(w, client, jid, cancelTemplate)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Success"})
}

func sendMessageToFieldMaster(w http.ResponseWriter, client *whatsmeow.Client, jid types.JID, stringTemplate string) {
	if _, err := client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(stringTemplate),
	}); err != nil {
		fmt.Println("Error sending OTP message:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func sendMessageToUserRefund(w http.ResponseWriter, client *whatsmeow.Client, jid types.JID, stringTemplate string) {

	if _, err := client.SendMessage(context.Background(), jid, &waProto.Message{
		Conversation: proto.String(stringTemplate),
	}); err != nil {
		fmt.Println("Error sending OTP message:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}
