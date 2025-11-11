// main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"weriKana/api/handlers"
	"weriKana/db"
	"weriKana/services/keystore"
	"weriKana/services/mpesa"
	"weriKana/services/natsclient"
	"weriKana/services/otp"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var (
	DB       *gorm.DB
	NATS     *nats.Conn
	KeyStore *keystore.KeyStore
	OTPSvc   *otp.Service
)

func main() {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.InfoLevel)

	if err := run(logger); err != nil {
		logger.Fatalf("Application failed: %v", err)
	}
        // Connect to NATS
	nc, err := nats.Connect(os.Getenv("NATS_URL"))
	if err != nil {
		log.Fatal("NATS connect failed:", err)
	}
	defer nc.Drain()

	nc.Subscribe("bets.cashout.withdraw", func(m *nats.Msg) {
		var msg ExecutionMessage
		msgpack.Unmarshal(m.Data, &msg)

		log.Printf("Received withdrawal for %s: %d KES", msg.CustomerID, msg.TotalCents)
		for _, w := range msg.Withdrawals {
			log.Printf("  → %s: %d cents (OTP: %s)", w.BookieName, w.AmountCents, w.OTP)
			// decrypt key, login, withdraw
		}
		m.Ack()
	})

	select {}

	log.Println("Execution Producer (msgpack) is running...")

	// Example: publish a test withdrawal
	go func() {
		time.Sleep(3 * time.Second)

		testMsg := ExecutionMessage{
			ParentRef:     "test-123",
			CustomerID:    uuid.New(),
			TotalCents:    5000,
			IsReal:        true,
			IdempotencyKey: uuid.New().String(),
			RequestedAt:   time.Now(),
			Withdrawals: []WithdrawalLeg{
				{
					BookieAccountID: uuid.New(),
					BookieName:      "SportPesa",
					AmountCents:     3000,
					EncryptedKey:    "enc-key-123",
					OTP:             "123456",
					TransactionID:   uuid.New(),
				},
				{
					BookieAccountID: uuid.New(),
					BookieName:      "Betika",
					AmountCents:     2000,
					EncryptedKey:    "enc-key-456",
					OTP:             "123456",
					TransactionID:   uuid.New(),
				},
			},
		}

		data, _ := msgpack.Marshal(&testMsg)
		nc.Publish("bets.cashout.withdraw", data)
		log.Println("Test withdrawal published (msgpack)")
	}()

	// Keep alive
	select {}
}



}

func run(logger *logrus.Logger) error {
	// === Config ===
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")
	mpesaURL := getEnv("MPESA_DJANGO_API_URL", "")
	callbackURL := getEnv("MPESA_STK_CALLBACK_URL", "http://localhost:9090/mpesa/stk-callback")

	// === DB ===
	db.InitDB()
	DB = db.DB

	// === NATS ===
	var err error
	NATS, err = nats.Connect(natsURL,
		nats.Name("bankroll-api"),
		nats.ReconnectWait(time.Second),
		nats.MaxReconnects(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, err error) {
			logger.Warnf("NATS disconnected: %v", err)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			logger.Info("NATS reconnected")
		}),
	)
	if err != nil {
		return err
	}
	defer NATS.Drain()

	// === Services ===
	KeyStore = keystore.New()
	OTPSvc = otp.New(DB, NATS)

	// === Background Consumers ===
	go mpesa.StartStkSequenceConsumer(DB, NATS)
	go handlers.StartExecutionEngine(DB, NATS)

	// === Router ===
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.Logger)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// === API v1 ===
	r.Route("/api/v1", func(r chi.Router) {
		// Smart Deposit
		r.Post("/smart-deposit", handlers.SmartDeposit(DB, NATS))

		// Withdraw Flow
		r.Post("/withdraw/otp", handlers.RequestWithdrawOTP(DB, OTPSvc))
		r.Post("/smart-withdraw", handlers.SmartWithdraw(DB, KeyStore))

		// MPESA Callbacks
		r.Post("/mpesa/stk-callback", mpesa.STKCallbackHandler(DB))

		// Dev Tools
		r.Post("/dev/fake-topup", handlers.FakeTopup(DB))
	})

	// === Server ===
	srv := &http.Server{
		Addr:    getEnv("HTTP_ADDR", ":9090"),
		Handler: r,
	}

	go func() {
		logger.Infof("BankRoll API running on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server error: %v", err)
		}
	}()

	// === Graceful Shutdown ===
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	logger.Info("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced shutdown: %v", err)
	}

	NATS.Drain()
	logger.Info("Goodbye, Kenya. BankRoll is down.")
	return nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
