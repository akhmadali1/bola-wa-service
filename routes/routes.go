package routes

import (
	"bola-wa-service/controller/otp_controller"
	"bola-wa-service/controller/payment_controller"
	"net/http"

	"github.com/robfig/cron/v3"
	"github.com/ulule/limiter/v3"
	mhttp "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"go.mau.fi/whatsmeow"
)

func SetupRoutes(client *whatsmeow.Client, cronScheduler *cron.Cron, reminderMap map[string]cron.EntryID) http.Handler {
	mux := http.NewServeMux()

	// Rate limiter configuration
	rate, err := limiter.NewRateFromFormatted("1000-S")
	if err != nil {
		panic(err)
	}
	store := memory.NewStore()
	rateLimiter := mhttp.NewMiddleware(limiter.New(store, rate))

	// CORS configuration
	corsHandler := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Apply middleware and set routes
	mux.Handle("/otp/send", corsHandler(rateLimiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otp_controller.SendOTP(w, r, client)
	}))))

	mux.Handle("/payment/send/fieldmaster", corsHandler(rateLimiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payment_controller.SendNotificationToFieldMaster(w, r, client, cronScheduler, reminderMap)
	}))))

	mux.Handle("/payment/send/user/refund", corsHandler(rateLimiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payment_controller.SendNotificationToUserRefund(w, r, client)
	}))))

	mux.Handle("/maintan/cron/reminder", corsHandler(rateLimiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payment_controller.DeleteUnusedCronReminders(w, r, cronScheduler, reminderMap)
	}))))

	mux.Handle("/health", corsHandler(rateLimiter.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			http.Error(w, "wa not healthy", http.StatusInternalServerError)
		} else {
			w.Write([]byte(`{"status": "healthy"}`))
		}
	}))))

	return mux
}
