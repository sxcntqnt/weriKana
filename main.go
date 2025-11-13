// main.go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"
    "weriKana/api/handlers"
    "weriKana/db"
    "weriKana/routes"
    "weriKana/service/keystore"
    "weriKana/service/mpesa"
    "weriKana/service/natsAnish"
    "weriKana/service/otp"
    "weriKana/service/dd_rr"
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "github.com/nats-io/nats.go"
    "github.com/sirupsen/logrus"
    "github.com/vmihailenco/msgpack/v5"
    "gorm.io/gorm"
)

// Config holds application configuration
type Config struct {
    HTTPAddr          string
    NATSURL           string
    DatabaseURL       string
    MpesaURL          string
    MpesaCallbackURL  string
    JWTSecret         string
    PrivateKey        string
    PublicKey         string
}

// App holds application dependencies
type App struct {
    Config     Config
    DB         *gorm.DB
    NATS       *nats.Conn
    KeyStore   *keystore.KeyStore
    OTPSvc     *otp.Service
    Logger     *logrus.Logger
    Server     *fiber.App
    Crypto     *securewithdrawal.CryptoEngine
}

// NewConfig loads configuration from environment variables
func NewConfig() Config {
    return Config{
        HTTPAddr:         getEnv("HTTP_ADDR", ":9090"),
        NATSURL:          getEnv("NATS_URL", "nats://localhost:4222"),
        DatabaseURL:      getEnv("DATABASE_URL", "postgres://localhost:5432/werikana"),
        MpesaURL:         getEnv("MPESA_DJANGO_API_URL", ""),
        MpesaCallbackURL: getEnv("MPESA_STK_CALLBACK_URL", "http://localhost:9090/mpesa/stk-callback"),
        JWTSecret:        getEnv("JWT_SECRET", "your-secret-key"),
        PrivateKey:       getEnv("PRIVATE_KEY", `-----BEGIN PRIVATE KEY-----
MC4CAQAwBQYDK2VwBCIEIO...
-----END PRIVATE KEY-----`),
        PublicKey:        getEnv("PUBLIC_KEY", `-----BEGIN PUBLIC KEY-----
MCow utopBQYDK2VwAyEA...
-----END PUBLIC KEY-----`),
    }
}

// NewApp initializes the application
func NewApp(cfg Config) (*App, error) {
    // Initialize logger
    logger := logrus.New()
    logger.SetFormatter(&logrus.JSONFormatter{})
    logger.SetOutput(os.Stdout)
    logger.SetLevel(logrus.InfoLevel)

    // Initialize database
    db, err := db.Init(cfg.DatabaseURL)
    if err != nil {
        logger.WithError(err).Error("Failed to initialize database")
        return nil, err
    }

    // Initialize NATS
    nc, err := nats.Connect(cfg.NATSURL,
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
        logger.WithError(err).Error("Failed to connect to NATS")
        return nil, err
    }

    // Initialize crypto engine
    crypto, err := securewithdrawal.NewCryptoEngine([]byte(cfg.PrivateKey), []byte(cfg.PublicKey))
    if err != nil {
        logger.WithError(err).Error("Failed to initialize crypto engine")
        return nil, err
    }

    // Initialize services
    keyStore := keystore.New()
    otpSvc := otp.New(db, nc)
    mpesa.Init(mpesa.Config{URL: cfg.MpesaURL, CallbackURL: cfg.MpesaCallbackURL})
    natsAnish.Init(nc)

    // Create Fiber app
    app := fiber.New(fiber.Config{
        ErrorHandler: func(c *fiber.Ctx, err error) error {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        },
    })

    return &App{
        Config:   cfg,
        DB:       db,
        NATS:     nc,
        KeyStore: keyStore,
        OTPSvc:   otpSvc,
        Logger:   logger,
        Server:   app,
        Crypto:   crypto,
    }, nil
}

// Start runs the application
func (a *App) Start() error {
    // Start background consumers
    go mpesa.StartStkSequenceConsumer(a.DB, a.NATS)
    go handlers.StartStkSequenceConsumer(a.DB, a.NATS) // From handlers/natsConsumer.go
    go handlers.ListenForWithdrawals(a.DB, a.NATS, a.Crypto) // Updated to pass Crypto
    go handlers.StartExecutionEngine(a.DB, a.NATS)

    // Setup routes
    routes.SetupRoutes(a.Server, a.DB, a.Config.JWTSecret, a.KeyStore, a.OTPSvc, a.NATS, a.Crypto)

    // Start server
    a.Logger.Infof("BankRoll API running on %s", a.Config.HTTPAddr)
    go func() {
        if err := a.Server.Listen(a.Config.HTTPAddr); err != nil && err != fiber.ErrServerShutdown {
            a.Logger.Fatalf("Server error: %v", err)
        }
    }()

    // Handle graceful shutdown
    stop := make(chan os.Signal, 1)
    signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
    <-stop
    a.Logger.Info("Shutting down gracefully...")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := a.Server.Shutdown(); err != nil {
        a.Logger.Errorf("Server forced shutdown: %v", err)
    }
    if err := a.NATS.Drain(); err != nil {
        a.Logger.Errorf("NATS drain failed: %v", err)
    }
    a.Logger.Info("Goodbye, Kenya. BankRoll is down.")
    return nil
}

// TestWithdrawal publishes a test withdrawal for development
func (a *App) TestWithdrawal() {
    go func() {
        time.Sleep(3 * time.Second)
        testMsg := handlers.ExecutionMessage{
            ParentRef:      "test-123",
            CustomerID:     uuid.New(),
            TotalCents:     5000,
            IsReal:         true,
            IdempotencyKey: uuid.New().String(),
            RequestedAt:    time.Now(),
            Withdrawals: []handlers.WithdrawalLeg{
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
        data, err := msgpack.Marshal(&testMsg)
        if err != nil {
            a.Logger.Errorf("Test withdrawal marshal failed: %v", err)
            return
        }
        if err := a.NATS.Publish("bets.cashout.withdraw", data); err != nil {
            a.Logger.Errorf("Test withdrawal publish failed: %v", err)
        }
        a.Logger.Info("Test withdrawal published (msgpack)")
    }()
}

func main() {
    cfg := NewConfig()
    app, err := NewApp(cfg)
    if err != nil {
        log.Fatalf("Application failed: %v", err)
    }

    // Run test withdrawal in development mode
    if getEnv("ENV", "prod") == "dev" {
        app.TestWithdrawal()
    }

    if err := app.Start(); err != nil {
        app.Logger.Fatalf("Application failed: %v", err)
    }
}

func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}
