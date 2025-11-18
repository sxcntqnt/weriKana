package routes

import (
    "github.com/gofiber/fiber/v2"
    "github.com/nats-io/nats.go"
    "weriKana/api/handlers"
    "weriKana/middleware"
    "weriKana/service/dd_rr"
    "gorm.io/gorm"
)

// SetupRoutes configures the API routes for the Fiber app
func SetupRoutes(app *fiber.App, db *gorm.DB, secretKey string, keyStore *handlers.KeyStore, otpSvc *handlers.OTPService, nc *nats.Conn, crypto *securewithdrawal.CryptoEngine) {
    // API group version 1
    v1 := app.Group("/api/v1")

    // Public routes (no JWT required)
    v1.Post("/token", handlers.Login(db, secretKey))                      // Login to get JWT
    v1.Post("/withdraw/otp", handlers.RequestWithdrawOTP(db, otpSvc))     // Request OTP for withdrawal
    // In your routes file
    v1.Post("/login", handlers.KeycloakLogin(app.KeyStore))

    // In routes.SetupRoutes
    v1.Post("/otp/request", handlers.RequestWithdrawOTP(app.DB, app.OTPSvc))
    v1.Post("/otp/verify", handlers.VerifyOTP(app.OTPSvc))

    // Authorized routes (require JWT)
    authorized := v1.Group("/", middleware.AuthMiddleware(secretKey))

    // Account-related routes
    authorized.Get("/account", handlers.GetAccount(db))                   // Get account details
    authorized.Post("/account/smart-deposit", handlers.SmartDeposit(db, nc)) // Smart deposit
    authorized.Post("/withdraw/smart", securewithdrawal.SmartWithdraw(app.DB, app.Crypto, app.OTPSvc))
    // In routes.SetupRoutes
    authorized.Post("/deposit/account", handlers.AccountDeposit(app.DB))
    authorized.Post("/deposit/smart", handlers.SmartDeposit(app.DB, app.NATS))
    authorized.Post("/account/trade", handlers.PlaceTrade(db))            // Place a trade
    authorized.Post("/account/fake-topup", handlers.FakeTopup(db))        // Fake balance top-up

    // Asset and Nexus-related routes
    authorized.Get("/asset-nexus", handlers.GetAssetNexus(db))            // Get asset nexus data
    authorized.Get("/sharp-profile", handlers.GetSharpProfile(db))        // Get sharp profile data

    // Smart deposit and withdraw routes
    authorized.Post("/account/smart-deposit", handlers.SmartDeposit(db)) // Smart deposit
    authorized.Post("/account/smart-withdraw", handlers.SmartWithdraw(db, keyStore, crypto)) // Smart withdraw with CryptoEngine

    // Start NATS consumer for MPESA STK sequence (background task)
    go handlers.StartStkSequenceConsumer(db, nc)
}

