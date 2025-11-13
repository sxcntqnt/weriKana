// api/handlers/sharp_profile.go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "weriKana/models"
)

// GetSharpProfile retrieves the customer's SharpProfile metrics
func GetSharpProfile(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        customerIDStr := c.Locals("customer_id").(string)
        accountType := c.Locals("account_type").(string)
        customerID, err := uuid.Parse(customerIDStr)
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid customer_id"})
        }
        var profile models.SharpProfile
        if err := db.Where("customer_id = ? AND asset_class = ?", customerID, accountType).First(&profile).Error; err != nil {
            return c.Status(404).JSON(fiber.Map{"error": "SharpProfile not found"})
        }
        return c.Status(200).JSON(fiber.Map{
            "customer_id":         customerID,
            "asset_class":         profile.AssetClass,
            "real_ev":             profile.RealEV,
            "fake_ev":             profile.FakeEV,
            "real_sharpe_ratio":   profile.RealSharpeRatio,
            "fake_sharpe_ratio":   profile.FakeSharpeRatio,
            "real_hit_rate":       profile.RealHitRate,
            "fake_hit_rate":       profile.FakeHitRate,
            "real_max_drawdown":   profile.RealMaxDrawdown,
            "fake_max_drawdown":   profile.FakeMaxDrawdown,
            "real_kelly_fraction": profile.RealKellyFraction,
            "fake_kelly_fraction": profile.FakeKellyFraction,
            "real_trade_volume":   profile.RealTradeVolume,
            "fake_trade_volume":   profile.FakeTradeVolume,
            "risk_score":          profile.RiskScore,
            "preferred_markets":   profile.PreferredMarkets,
        })
    }
}
