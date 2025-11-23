// middleware/appcontext.go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "gorm.io/gorm"
    "github.com/nats-io/nats.go"

    "weriKana/internal/appcontext"
    "weriKana/internal/bank"     // for bank.SecureConn
    "weriKana/service/keystore"  // for keystore.KeyStore
    "weriKana/service/otp"       // for otp.Service
    "weriKana/models"            // for models.Customer, models.SportsAccount, etc.
)           

// InjectAppContext makes the global AppContext available in every handler
// via c.Locals("app") → *appcontext.AppContext
func InjectAppContext(ac *appcontext.AppContext) fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Locals("app", ac)
		return c.Next()
	}
}

// AppFrom is the shortest way to get it in any handler
func AppFrom(c *fiber.Ctx) *appcontext.AppContext {
	return c.Locals("app").(*appcontext.AppContext)
}

// Optional convenience shortcuts (use everywhere)
func DB(c *fiber.Ctx) *gorm.DB {
	return AppFrom(c).DB
}

func NATS(c *fiber.Ctx) *nats.Conn {
	return AppFrom(c).NATS
}

func SecureBus(c *fiber.Ctx) *bank.SecureConn {
	return AppFrom(c).SecureBus
}

func BalanceEngine(c *fiber.Ctx) *bank.BalanceEngine {
	return AppFrom(c).BalanceEngine
}

func KeyStore(c *fiber.Ctx) *keystore.KeyStore {
	return AppFrom(c).KeyStore
}

func OTP(c *fiber.Ctx) *otp.Service {
	return AppFrom(c).OTPSvc
}

// Request-scoped helpers (already in appcontext package, just re-export for beauty)
func WithAccount(c *fiber.Ctx, acc any) {
	ctx := appcontext.WithAccount(c.UserContext(), acc)
	c.SetUserContext(ctx)
}

func AccountFrom(c *fiber.Ctx) any {
	return appcontext.AccountFrom(c.UserContext())
}

func WithCustomer(c *fiber.Ctx, cust *models.Customer) {
	ctx := appcontext.WithCustomer(c.UserContext(), cust)
	c.SetUserContext(ctx)
}

func CustomerFrom(c *fiber.Ctx) *models.Customer {
	return appcontext.CustomerFrom(c.UserContext())
}
