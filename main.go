// main.go — weriKana API v2025 — The Final Form
// BalanceEngine + SecureBus + M-Pesa + OTP + Vault + Keycloak
// Nairobi, Kenya — Where Legends Are Built

package main

import (
	"context"
	"encoding/base64"
	"os"
	"os/signal"
	"syscall"
	"time"

	"weriKana/api/handlers"
	"weriKana/db"
	"weriKana/internal/bank"
        "weriKana/internal/bankd"
        "weriKana/internal/appcontext"
	"weriKana/middleware"
	"weriKana/routes"
	"weriKana/service/dd_rr"
	"weriKana/service/keystore"
	"weriKana/service/mpesa"
	"weriKana/service/natsAnish"
	"weriKana/service/otp"
        "weriKana/service/uzeey"

	"github.com/gofiber/fiber/v2"
	"github.com/go-redis/redis/v8"
	"github.com/hashicorp/vault/api"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
)

var (
	secureConn    *bank.SecureConn
	balanceEngine *bank.BalanceEngine
	logger        = logrus.New()
)

func main() {
	// Set up logger
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetLevel(logrus.InfoLevel)

	// ========================================
	// 1. Configuration — JWT_SECRET IS DEAD
	// ========================================
	_ = getEnv("JWT_SECRET", "") // Just to silence unused warning

	natsURL := getEnv("NATS_URL", nats.DefaultURL)
	dbURL := getEnv("DATABASE_URL", "postgres://localhost:5432/werikana?sslmode=disable")
	redisURL := getEnv("REDIS_URL", "redis://localhost:6379/0")

	// ========================================
	// 2. Core Systems — PERFECT
	// ========================================
	database, err := db.Init(dbURL)
	if err != nil {
		logger.Fatalf("DB failed: %v", err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		logger.Fatalf("Failed to access underlying DB connection: %v", err)
	}
	defer sqlDB.Close() // This closes the *sql.DB, not the GORM instance

	redisClient := redis.NewClient(&redis.Options{
		Addr:     parseRedisURL(redisURL),
		Password: getEnv("REDIS_PASSWORD", ""),
		DB:       0,
	})
	defer redisClient.Close()

	nc, err := nats.Connect(natsURL,
		nats.Name("weriKana-api"),
		nats.ReconnectWait(time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.WithError(err).Warn("NATS disconnected")
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info("NATS reconnected")
		}),
	)
	if err != nil {
		logger.Fatalf("NATS failed: %v", err)
	}
	defer nc.Drain()

	// ========================================
	// 3. BalanceEngine — PERFECT
	// ========================================
	balanceEngine = bank.NewBalanceEngine(database, redisClient)
	defer balanceEngine.Shutdown()
	logger.Info("BalanceEngine warming up — this is normal and elite")

	// ========================================
	// 4. SecureBus — PERFECT
	// ========================================
	myIdentity, err := loadIdentityFromVault()
	if err != nil {
		logger.Fatalf("Failed to load SecureBus identity: %v", err)
	}

	executionPubKeyB64 := getEnv("EXECUTION_PUBKEY", "")
	if executionPubKeyB64 == "" {
		logger.Fatal("EXECUTION_PUBKEY is required")
	}
	executionPubKeyBytes, _ := base64.StdEncoding.DecodeString(executionPubKeyB64)
	executionPubKey, err := bank.PublicKeyFromBytes(executionPubKeyBytes)
	if err != nil {
		logger.Fatalf("Invalid EXECUTION_PUBKEY: %v", err)
	}
	secureConn, meta, err := bank.Dial(
		getEnv("SECUREBUS_URL", "wss://127.0.0.1:4223"),
		myIdentity,
		&executionPubKey,
		nil, // Fresh connection; pass *bank.SessionMetadata for resume
	)
	if err != nil {
		logger.Fatalf("SecureBus connection failed: %v", err)
	}
	defer secureConn.Close()

	logger.WithFields(logrus.Fields{
		"remote_static": meta.RemoteStatic,
		"session_tag":   meta.SessionTag,
		"0rtt_ready":    len(meta.ResumeToken) > 0,
	}).Info("SecureBus connected — execution engine online")

	// ========================================
	// 5. KeyStore — PERFECT (this is your crown jewel)
	// ========================================
	keyStore, err := keystore.NewKeyStore(
		getEnv("VAULT_ADDR", "http://127.0.0.1:8200"),
		getEnv("VAULT_TOKEN", "root"),
		getEnv("VAULT_PATH", "secret/data/werikana"), // fixed typo: "secretory" → "secret"
		getEnv("KEYCLOAK_URL", "http://localhost:8080"),
		getEnv("KEYCLOAK_REALM", "werikana"),
	)
	if err != nil {
		logger.Fatalf("KeyStore failed: %v", err)
	}

	// ========================================
	// 6. Initialize services
	// ========================================
	otpSvc := otp.New()
	dd_rr.InitCryptoEngine()

	mpesa.Init(mpesa.Config{
		BaseURL:        getEnv("MPESA_DJANGO_API_URL", "https://mpesa-api.yourapp.com"),
		APIToken:       getEnv("MPESA_API_TOKEN", ""),
		STKCallbackURL: getEnv("MPESA_STK_CALLBACK_URL", "https://yourdomain.com/mpesa/stk-callback"),
	})
	mpesa.SetDB(database)
	natsAnish.StartExecutionConsumer(database,nc)

	// ========================================
	// 7. AppContext (Fiber)
	// ========================================
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler,
	})

	ctx := &appcontext.AppContext{
		App:           app,
		DB:            database,
		KeyStore:      keyStore,      // ← Your real auth source
		OTPSvc:        otpSvc,
		NATS:          nc,
		SecureBus:     secureConn,
		BalanceEngine: balanceEngine,
	}
        app.Use(middleware.InjectAppContext(ctx))
	routes.SetupRoutes(ctx)

	// ========================================
	// 8. Start background workers
	// ========================================
	go uzeey.StartStkSequenceConsumer(database, nc)
	go handlers.StartWithdrawalConsumer(database, nc)
        go listenForExecutionFills(nc, secureConn)
        // Start SSH server for partner banks (port 2222)
	gitRepoPath := getEnv("GIT_REPO_PATH", "./authorized_keys.git") // Add env for git repo
	go func() {
		logger.Info("Starting SSH Bank Bus — :2222")
		if err := bankd.StartSSHServer(ctx, gitRepoPath); err != nil { // Fixed: Pass gitRepoPath, check err
			logger.Fatalf("SSH server failed: %v", err)
		}
	}()
	// Start the HTTP API (Fiber)
	go func() {
		logger.Info("weriKana API starting — :9090")
		if err := app.Listen(":9090"); err != nil {
			logger.Fatalf("HTTP server failed: %v", err)
		}
	}()

	// ========================================
	// 9. Graceful Shutdown
	// ========================================
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("Shutting down — Nairobi style")
        shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	// Shutdown procedures
	app.ShutdownWithContext(shutdownCtx)
	defer cancel()

	if err := bankd.ShutdownSSH(shutdownCtx); err != nil {
		logger.Printf("Failed to gracefully shutdown SSH server: %v", err)
	}


	secureConn.Close()
	balanceEngine.Shutdown()
	nc.Drain()

	logger.Info("Shutdown complete. Balances safe. Tokens secure. AFRICA wins.")
}

// ========================================
// SecureBus Fill Listener
// ========================================
func listenForExecutionFills(nc *nats.Conn, secureConn *bank.SecureConn) {
	for {
		var fill struct {
			Symbol string  `msgpack:"symbol"`
			Price  float64 `msgpack:"price"`
			Qty    int     `msgpack:"qty"`
			Status string  `msgpack:"status"`
		}
		topic, err := secureConn.Receive(context.Background(), &fill)
		if err != nil {
			logger.WithError(err).Error("SecureBus receive failed")
			time.Sleep(2 * time.Second)
			continue
		}
		if topic == "execution.fill" {
			logger.WithFields(logrus.Fields{
				"symbol": fill.Symbol,
				"price":  fill.Price,
				"qty":    fill.Qty,
			}).Info("EXECUTION FILL RECEIVED")
			if err := bank.BroadcastFill(nc, fill.Symbol, fill.Price, fill.Qty, fill.Status); err != nil {
				logger.WithError(err).Error("Failed to broadcast fill")
			}
		}
	}
}

// ========================================
// Helpers
// ========================================
func loadIdentityFromVault() (bank.IdentityKey, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return bank.IdentityKey{}, err
	}
	client.SetToken(getEnv("VAULT_TOKEN", "root"))

	secret, err := client.Logical().Read("secret/data/securebus/nodes/werikana-api")
	if secret == nil || secret.Data["data"] == nil {
		return bank.IdentityKey{}, err
	}

	data := secret.Data["data"].(map[string]interface{})
	privB64 := data["private_key"].(string)
	pubB64 := data["public_key"].(string)

	privBytes, _ := base64.StdEncoding.DecodeString(privB64)
	pubBytes, _ := base64.StdEncoding.DecodeString(pubB64)

	var key bank.IdentityKey
	copy(key.Private[:], privBytes)
	copy(key.Public[:], pubBytes)
	return key, nil
}

func parseRedisURL(url string) string {
	if url == "" {
		return "localhost:6379"
	}
	// redis://[:password@]host[:port][/db]
	return url // redis-go handles it
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

