package api

import (
	"bola-wa-service/routes"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	// _ "github.com/mattn/go-sqlite3"
	_ "github.com/lib/pq"
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

func OpenConnection() *sql.DB {

	host := os.Getenv("DB_HOST")
	port := 5432
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PW")
	dbname := os.Getenv("DB_NAME")

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	fmt.Println(psqlInfo)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(10)
	return db
}

// func main() {
// 	dbLog := waLog.Stdout("Database", "DEBUG", true)
// 	// dbPath, err := filepath.Abs("otpdbtemp.db")
// 	// if err != nil {
// 	// 	panic(err)
// 	// }

// 	// container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), dbLog)
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	err := godotenv.Load()
// 	if err != nil {
// 		panic("Error loading .env file")
// 	}

// db := OpenConnection()

// // Ensure the schema is created
// err = EnsureSchema(db)
// if err != nil {
// 	panic(fmt.Sprintf("Failed to create schema: %v", err))
// }

// // Create the container with the PostgreSQL connection
// container := sqlstore.NewWithDB(db, "postgres", dbLog)

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
	// dbPath, err := filepath.Abs("otpdbtemp.db")
	// if err != nil {
	// 	panic(err)
	// }

	// container, err := sqlstore.New("sqlite3", fmt.Sprintf("file:%s?_foreign_keys=on", dbPath), dbLog)
	// if err != nil {
	// 	panic(err)
	// }

	db := OpenConnection()

	// Ensure the schema is created
	err := EnsureSchema(db)
	if err != nil {
		panic(fmt.Sprintf("Failed to create schema: %v", err))
	}

	// Create the container with the PostgreSQL connection
	container := sqlstore.NewWithDB(db, "postgres", dbLog)

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

func EnsureSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS whatsmeow_device (
		jid TEXT PRIMARY KEY,
		registration_id BIGINT NOT NULL CHECK ( registration_id >= 0 AND registration_id < 4294967296 ),
		noise_key bytea NOT NULL CHECK ( length(noise_key) = 32 ),
		identity_key bytea NOT NULL CHECK ( length(identity_key) = 32 ),
		signed_pre_key bytea NOT NULL CHECK ( length(signed_pre_key) = 32 ),
		signed_pre_key_id INTEGER NOT NULL CHECK ( signed_pre_key_id >= 0 AND signed_pre_key_id < 16777216 ),
		signed_pre_key_sig bytea NOT NULL CHECK ( length(signed_pre_key_sig) = 64 ),
		adv_key bytea NOT NULL,
		adv_details bytea NOT NULL,
		adv_account_sig bytea NOT NULL CHECK ( length(adv_account_sig) = 64 ),
		adv_device_sig bytea NOT NULL CHECK ( length(adv_device_sig) = 64 ),
		platform TEXT NOT NULL DEFAULT '',
		business_name TEXT NOT NULL DEFAULT '',
		push_name TEXT NOT NULL DEFAULT '',
		adv_account_sig_key bytea CHECK ( length(adv_account_sig_key) = 32 ),
		facebook_uuid uuid
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_message_secrets (
		our_jid TEXT,
		chat_jid TEXT,
		sender_jid TEXT,
		message_id TEXT,
		key bytea NOT NULL,
		PRIMARY KEY (our_jid, chat_jid, sender_jid, message_id),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_privacy_tokens (
		our_jid TEXT,
		their_jid TEXT,
		token bytea NOT NULL,
		timestamp BIGINT NOT NULL,
		PRIMARY KEY (our_jid, their_jid)
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_chat_settings (
		our_jid TEXT,
		chat_jid TEXT,
		muted_until BIGINT NOT NULL DEFAULT 0,
		pinned BOOLEAN NOT NULL DEFAULT false,
		archived BOOLEAN NOT NULL DEFAULT false,
		PRIMARY KEY (our_jid, chat_jid),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_contacts (
		our_jid TEXT,
		their_jid TEXT,
		first_name TEXT,
		full_name TEXT,
		push_name TEXT,
		business_name TEXT,
		PRIMARY KEY (our_jid, their_jid),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_app_state_mutation_macs (
		jid TEXT,
		name TEXT,
		version BIGINT,
		index_mac bytea CHECK ( length(index_mac) = 32 ),
		value_mac bytea NOT NULL CHECK ( length(value_mac) = 32 ),
		PRIMARY KEY (jid, name, version, index_mac),
		FOREIGN KEY (jid, name) REFERENCES whatsmeow_app_state_version(jid, name) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_app_state_version (
		jid TEXT,
		name TEXT,
		version BIGINT NOT NULL,
		hash bytea NOT NULL CHECK ( length(hash) = 128 ),
		PRIMARY KEY (jid, name),
		FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_sender_keys (
		our_jid TEXT,
		chat_id TEXT,
		sender_id TEXT,
		sender_key bytea NOT NULL,
		PRIMARY KEY (our_jid, chat_id, sender_id),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_app_state_sync_keys (
		jid TEXT,
		key_id bytea,
		key_data bytea NOT NULL,
		timestamp BIGINT NOT NULL,
		fingerprint bytea NOT NULL,
		PRIMARY KEY (jid, key_id),
		FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_sessions (
		our_jid TEXT,
		their_id TEXT,
		session bytea,
		PRIMARY KEY (our_jid, their_id),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_pre_keys (
		jid TEXT,
		key_id INTEGER CHECK ( key_id >= 0 AND key_id < 16777216 ),
		key bytea NOT NULL CHECK ( length(key) = 32 ),
		uploaded BOOLEAN NOT NULL,
		PRIMARY KEY (jid, key_id),
		FOREIGN KEY (jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_identity_keys (
		our_jid TEXT,
		their_id TEXT,
		identity bytea NOT NULL CHECK ( length(identity) = 32 ),
		PRIMARY KEY (our_jid, their_id),
		FOREIGN KEY (our_jid) REFERENCES whatsmeow_device(jid) ON DELETE CASCADE ON UPDATE CASCADE
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_version (
		version INTEGER
	);
	`

	_, err := db.Exec(schema)

	return err
}
