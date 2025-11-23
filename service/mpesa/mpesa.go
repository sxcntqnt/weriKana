// service/mpesa/mpesa.go
package mpesa

import (
    "log"
    "net/http"
    "os"
    "time"

    "gorm.io/gorm"
    "github.com/gofiber/fiber/v2"
)

// Config — public config struct
type Config struct {
    BaseURL        string
    APIToken       string // optional
    STKCallbackURL string
    HTTPClient     *http.Client
}

// internal global state (set once)
var (
    config Config
    db     *gorm.DB
)

// Init MUST be called once at startup
func Init(cfg Config) {
    if cfg.BaseURL == "" {
        cfg.BaseURL = os.Getenv("MPESA_DJANGO_API_URL")
    }
    if cfg.BaseURL == "" {
        log.Fatal("MPESA_DJANGO_API_URL is required")
    }

    if cfg.APIToken == "" {
        cfg.APIToken = os.Getenv("MPETA_API_TOKEN")
    }

    if cfg.STKCallbackURL == "" {
        cfg.STKCallbackURL = os.Getenv("MPESA_STK_CALLBACK_URL")
    }
    if cfg.STKCallbackURL == "" {
        log.Fatal("MPESA_STK_CALLBACK_URL is required")
    }

    if cfg.HTTPClient == nil {
        cfg.HTTPClient = &http.Client{Timeout: 30 * time.Second}
    }

    config = cfg
    log.Printf("M-Pesa service initialized → %s", config.BaseURL)
}

// SetDB — required before using callback handler
func SetDB(database *gorm.DB) {
    db = database
}

// CallbackHandler — renamed to avoid conflict, adapted for Fiber
func CallbackHandler() fiber.Handler {
    // Ensure db is initialized before using the handler
    if db == nil {
        log.Fatal("mpesa.SetDB() must be called before using callback")
    }
    return STKCallbackHandler(db) // Use the refactored STKCallbackHandler that works with Fiber
}


// Optional helper
func StartConsumers(database *gorm.DB) {
    SetDB(database)
    log.Println("M-Pesa background consumers ready")
}
