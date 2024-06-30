package api

import (
	"bola-wa-service/routes"
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/robfig/cron/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var (
	app           *gin.Engine
	client        *whatsmeow.Client
	cronScheduler *cron.Cron
	reminderMap   = make(map[string]cron.EntryID)
)

// func main() {
// 	dbLog := waLog.Stdout("Database", "DEBUG", true)
// 	dbPath, err := filepath.Abs("otpdbtemp.db")
// 	if err != nil {
// 		panic(err)
// 	}

// 	container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), dbLog)
// 	if err != nil {
// 		panic(err)
// 	}

// 	deviceStore, err := container.GetFirstDevice()
// 	if err != nil {
// 		panic(err)
// 	}

// 	clientLog := waLog.Stdout("Client", "DEBUG", false)
// 	client = whatsmeow.NewClient(deviceStore, clientLog)

// 	if client.Store.ID == nil {
// 		qrChan, _ := client.GetQRChannel(context.Background())
// 		err = client.Connect()
// 		if err != nil {
// 			panic(err)
// 		}
// 		for evt := range qrChan {
// 			if evt.Event == "code" {
// 				fmt.Println("QR code:", evt.Code)
// 			} else {
// 				fmt.Println("Login event:", evt.Event)
// 			}
// 		}
// 	} else {
// 		err = client.Connect()
// 		if err != nil {
// 			panic(err)
// 		}
// 	}

// 	loc, err := time.LoadLocation("Asia/Jakarta")
// 	if err != nil {
// 		panic(err)
// 	}

// 	cronScheduler = cron.New(cron.WithLocation(loc))

// 	// stop scheduler tepat sebelum fungsi berakhir
// 	go cronScheduler.Start()
// 	defer cronScheduler.Stop()

// 	router := routes.SetupRoutes(client, cronScheduler, reminderMap)
// 	router.Run(":8073")
// 	go monitorConnection()

// 	// Use a buffered channel for signals to prevent SA1017 warning
// 	signalChan := make(chan os.Signal, 1)
// 	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

// 	// Use a simple channel receive instead of select with a single case (S1000)
// 	sig := <-signalChan
// 	fmt.Printf("Received signal: %v\n", sig)
// 	client.Disconnect()
// 	cronScheduler.Stop()
// 	os.Exit(0)
// }

func monitorConnection() {
	for {
		time.Sleep(time.Second * 10) // Adjust the interval based on your requirements
		if !client.IsConnected() {
			fmt.Println("Connection lost. Restarting service...")
			restartService()
		}
	}
}

func restartService() {
	cmd := exec.Command("sudo", "service", "bola-wa-service", "restart")
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error restarting service:", err)
	}
}

func init() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	dbPath, err := filepath.Abs("otpdbtemp.db")
	if err != nil {
		panic(err)
	}

	container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", false)
	client = whatsmeow.NewClient(deviceStore, clientLog)

	if client.Store.ID == nil {
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				fmt.Println("QR code:", evt.Code)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(err)
	}

	cronScheduler = cron.New(cron.WithLocation(loc))

	// stop scheduler tepat sebelum fungsi berakhir
	go cronScheduler.Start()
	defer cronScheduler.Stop()

	router := routes.SetupRoutes(client, cronScheduler, reminderMap)
	app = router
	go monitorConnection()

	// Use a buffered channel for signals to prevent SA1017 warning
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	// Use a simple channel receive instead of select with a single case (S1000)
	sig := <-signalChan
	fmt.Printf("Received signal: %v\n", sig)
	client.Disconnect()
	cronScheduler.Stop()
	os.Exit(0)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	app.ServeHTTP(w, r)
}
