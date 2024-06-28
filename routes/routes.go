package routes

import (
	"bola-wa-service/controller/otp_controller"
	"bola-wa-service/controller/payment_controller"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	limiter "github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	"go.mau.fi/whatsmeow"
)

func SetupRoutes(client *whatsmeow.Client, cronScheduler *cron.Cron, reminderMap map[string]cron.EntryID) *gin.Engine {
	route := gin.Default()

	// Note: Rate Limiter
	// * 5 reqs/second: "5-S"
	// * 10 reqs/minute: "10-M"
	// * 1000 reqs/hour: "1000-H"
	// * 2000 reqs/day: "2000-D"

	// limit to 1000 requests per second. if exceed, will return http 429 (too many req)
	rate, err := limiter.NewRateFromFormatted("1000-S")
	if err != nil {
		return route
	}

	store := memory.NewStore()

	// Create a new middleware with the limiter instance using in memory golang.
	middlewares := mgin.NewMiddleware(limiter.New(store, rate))

	// Forward / Save Client ip to go memory.
	route.ForwardedByClientIP = true

	// Use Middleware rate limiter
	route.Use(middlewares)

	route.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*", "http://localhost:8081", "https://ntxxm6jj-8081.asse.devtunnels.ms"},
		AllowCredentials: true,
		AllowMethods:     []string{"POST", "PUT", "PATCH", "DELETE", "GET", "OPTIONS", "TRACE", "CONNECT"},
		AllowHeaders:     []string{"Authorization", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Origin", "Content-Type", "Content-Length", "Date", "origin", "Origins", "x-requested-with", "access-control-allow-methods", "access-control-allow-credentials", "apikey"},
		ExposeHeaders:    []string{"Content-Length"},
	}))

	otp := route.Group("/otp")
	{
		otp.POST("send", func(ctx *gin.Context) {
			otp_controller.SendOTP(ctx, client)
		})
	}

	payment := route.Group("/payment")
	{
		payment.POST("send/fieldmaster", func(ctx *gin.Context) {
			payment_controller.SendNotificationToFieldMaster(ctx, client, cronScheduler, reminderMap)
		})
		payment.POST("send/user/refund", func(ctx *gin.Context) {
			payment_controller.SendNotificationToUserRefund(ctx, client)
		})
	}

	maintain := route.Group("/maintan")
	{
		maintain.GET("cron/reminder", func(ctx *gin.Context) {
			payment_controller.DeleteUnusedCronReminders(ctx, cronScheduler, reminderMap)
		})
	}

	return route
}
