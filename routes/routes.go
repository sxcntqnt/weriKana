package routes

import (
    "weriKana/api/handlers"
    "weriKana/middleware"
    "weriKana/service/mpesa"
    "weriKana/internal/appcontext"
    "weriKana/service/uzeey"
)

func SetupRoutes(ctx *appcontext.AppContext) {
    app := ctx.App
    db := ctx.DB
    keyStore := ctx.KeyStore
    otpSvc := ctx.OTPSvc
    nc := ctx.NATS
    balanceEngine := ctx.BalanceEngine

    v1 := app.Group("/api/v1")

    // ====================== PUBLIC ROUTES ======================
    v1.Post("/login", handlers.KeycloakLogin(keyStore))
    v1.Post("/withdraw/otp", handlers.RequestWithdrawOTP(db, otpSvc))
    v1.Post("/otp/request", handlers.RequestWithdrawOTP(db, otpSvc)) // if you want both endpoints
    v1.Post("/otp/verify", handlers.VerifyOTP(otpSvc))
    v1.Post("/mpesa/stk-callback", mpesa.STKCallbackHandler(db))

    // ====================== PROTECTED ROUTES ======================
    authorized := v1.Group("/", middleware.KeycloakAuth(keyStore))

    // Account
    authorized.Get("/account", handlers.GetAccount(db))
    authorized.Post("/account/fake-topup", handlers.FakeTopup(db))

    // ====================== Balance Routes ======================
    // You need to instantiate BalanceHandler with balanceEngine
    balanceHandler := handlers.NewBalanceHandler(balanceEngine)

    // Customer routes
    authorized.Get("/me", balanceHandler.GetMyBalance)

    // Admin routes
    authorized.Get("/balances", balanceHandler.AdminGetAllBalances)
    authorized.Get("/balance-engine/metrics", balanceHandler.GetBalanceEngineMetrics)

    // ====================== Deposits ======================
    authorized.Post("/deposit/account", handlers.AccountDeposit(db))
    authorized.Post("/deposit/smart", handlers.SmartDeposit(db, nc))
    authorized.Post("/account/smart-deposit", handlers.SmartDeposit(db, nc)) // you had this twice

    // ====================== Withdrawals ======================
    authorized.Post("/account/smart-withdraw", handlers.SmartWithdraw(db, keyStore, otpSvc))

    // ====================== Assets & Profile ======================
    authorized.Get("/asset-nexus", handlers.GetAssetNexus(db))
    authorized.Get("/sharp-profile", handlers.GetSharpProfile(db))

    // Start background NATS consumer
    go uzeey.StartStkSequenceConsumer(db, nc)
}

