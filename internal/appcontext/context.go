// internal/appcontext/appcontext.go
package appcontext

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"

	"weriKana/models"
	"weriKana/service/keystore"
	"weriKana/service/otp"
        "weriKana/internal/bank"
)

// ============================================================================
// 1. Global app-wide context (injected into handlers via closure or middleware)
// ============================================================================
type AppContext struct {
	App           *fiber.App
	DB            *gorm.DB
	SecretKey     string
	KeyStore      *keystore.KeyStore
	OTPSvc        *otp.Service
	NATS          *nats.Conn
	SecureBus     *bank.SecureConn
	BalanceEngine *bank.BalanceEngine
}

// ============================================================================
// 2. Request-scoped context keys & helpers (breaks the import cycle)
// ============================================================================

type ctxKey string

const (
	AccountKey  ctxKey = "account"  // value will be one of the *Account types
	SharpKey ctxKey = "sharp" // value is *models.Sharp
)

// ——— Account (polymorphic – can hold SportsAccount, StockAccount, etc.) ———
func WithAccount(ctx context.Context, acc any) context.Context {
	return context.WithValue(ctx, AccountKey, acc)
}

func AccountFrom(ctx context.Context) any {
	if v := ctx.Value(AccountKey); v != nil {
		return v
	}
	return nil
}

// ——— Customer ———
func WithCustomer(ctx context.Context, c *models.Sharp) context.Context {
	return context.WithValue(ctx, SharpKey, c)
}

func CustomerFrom(ctx context.Context) *models.Sharp {
	if v := ctx.Value(SharpKey); v != nil {
		if cust, ok := v.(*models.Sharp); ok {
			return cust
		}
	}
	return nil
}

// Optional: typed helpers if you hate `any`
func SportsAccountFrom(ctx context.Context) *models.SportsAccount {
	if v := AccountFrom(ctx); v != nil {
		if acc, ok := v.(*models.SportsAccount); ok {
			return acc
		}
	}
	return nil
}

// You can add StockAccountFrom, ForexAccountFrom, etc. the same way
