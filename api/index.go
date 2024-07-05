package api

import (
	"bola-wa-service/routes"
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var (
	client         *whatsmeow.Client
	cronScheduler  *cron.Cron
	reminderMap    = make(map[string]cron.EntryID)
	log            waLog.Logger
	logLevel       = "INFO"
	debugLogs      = flag.Bool("debug", false, "Enable debug logs?")
	pairRejectChan = make(chan bool, 1)
	httpServer     *http.Server
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

func init() {
	// err := godotenv.Load()
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	flag.Parse()
	if *debugLogs {
		logLevel = "DEBUG"
	}
	log = waLog.Stdout("Main", logLevel, true)

	dbLog := waLog.Stdout("Database", logLevel, true)
	db := OpenConnection()

	errDB := EnsureSchema(db)
	if errDB != nil {
		panic(fmt.Sprintf("Failed to create schema: %v", errDB))
	}

	storeContainer := sqlstore.NewWithDB(db, "postgres", dbLog)
	device, err := storeContainer.GetFirstDevice()
	if err != nil {
		log.Errorf("Failed to get device: %v", err)
		return
	}

	client = whatsmeow.NewClient(device, waLog.Stdout("Client", logLevel, true))
	var isWaitingForPair atomic.Bool
	client.PrePairCallback = func(jid types.JID, platform, businessName string) bool {
		isWaitingForPair.Store(true)
		defer isWaitingForPair.Store(false)
		log.Infof("Pairing %s (platform: %q, business name: %q). Type r within 3 seconds to reject pair", jid, platform, businessName)
		select {
		case reject := <-pairRejectChan:
			if reject {
				log.Infof("Rejecting pair")
				return false
			}
		case <-time.After(3 * time.Second):
		}
		log.Infof("Accepting pair")
		return true
	}

	ch, err := client.GetQRChannel(context.Background())
	if err != nil {
		if !errors.Is(err, whatsmeow.ErrQRStoreContainsID) {
			log.Errorf("Failed to get QR channel: %v", err)
		}
	} else {
		go func() {
			for evt := range ch {
				if evt.Event == "code" {
					fmt.Println("QR code:", evt.Code)
				} else {
					fmt.Println("Login event:", evt.Event)
				}
			}
		}()
	}

	err = client.Connect()
	if err != nil {
		log.Errorf("Failed to connect: %v", err)
		return
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		panic(err)
	}

	cronScheduler = cron.New(cron.WithLocation(loc))
	go cronScheduler.Start()
	defer cronScheduler.Stop()

	router := routes.SetupRoutes(client, cronScheduler, reminderMap)
	server := &http.Server{
		Handler: router,
	}

	httpServer = server

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	sig := <-signalChan
	fmt.Printf("Received signal: %v\n", sig)
	client.Disconnect()
	cronScheduler.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}
}

func Handler(w http.ResponseWriter, r *http.Request) {
	httpServer.Handler.ServeHTTP(w, r)
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
