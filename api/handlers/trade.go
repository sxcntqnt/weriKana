// api/handlers/trade.go
package handlers

import (
    "fmt"
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "weriKana/models"
)

// PlaceTrade handles placing a trade, updating account and profile
func PlaceTrade(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var input struct {
            AmountCents int64   `json:"amount_cents"`
            IsReal      bool    `json:"is_real"`
            EV          float64 `json:"ev"`
            Metadata    JSONMap `json:"metadata"`
        }
        if err := c.BodyParser(&input); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": err.Error()})
        }
        if input.AmountCents <= 0 {
            return c.Status(400).JSON(fiber.Map{"error": "Amount must be positive"})
        }
        customerIDStr := c.Locals("customer_id").(string)
        sharpIDStr := c.Locals("sharp_id").(string)
        accountType := c.Locals("account_type").(string)
        customerID, err := uuid.Parse(customerIDStr)
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid customer_id"})
        }
        sharpID, err := uuid.Parse(sharpIDStr)
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid sharp_id"})
        }
        var profile models.SharpProfile
        if err := db.Where("customer_id = ? AND asset_class = ?", customerID, accountType).First(&profile).Error; err != nil {
            return c.Status(404).JSON(fiber.Map{"error": "SharpProfile not found"})
        }
        var transaction models.Transaction
        switch accountType {
        case "sharp":
            var acc models.SharpAccount
            if err := db.Where("customer_id = ? AND sharp_id = ? AND sharp_profile_id = ?", customerID, sharpID, profile.ID).First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            if input.IsReal && acc.RealBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient real balance"})
            }
            if !input.IsReal && acc.FakeBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient fake balance"})
            }
            if input.IsReal {
                acc.RealBalanceCents -= input.AmountCents
                profile.RealTradeVolume += input.AmountCents
                profile.RealEV += input.EV
            } else {
                acc.FakeBalanceCents -= input.AmountCents
                profile.FakeTradeVolume += input.AmountCents
                profile.FakeEV += input.EV
            }
            if err := db.Save(&acc).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update account"})
            }
            if err := db.Save(&profile).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update profile"})
            }
            transaction = models.Transaction{
                SharpAccountID: acc.ID,
                CustomerID:     customerID,
                Type:           models.TransactionTypeTrade,
                AmountCents:    input.AmountCents,
                IsReal:         input.IsReal,
                Currency:       "KES",
                Status:         models.StatusSuccess,
                Metadata:       JSONMap{"ev": input.EV, "metadata": input.Metadata},
                Reference:      fmt.Sprintf("TRD-%s", uuid.New().String()[:8]),
            }
        case "sports":
            var acc models.SportsAccount
            if err := db.Where("customer_id = ? AND sharp_id = ? AND sharp_profile_id = ?", customerID, sharpID, profile.ID).First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            if input.IsReal && acc.RealBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient real balance"})
            }
            if !input.IsReal && acc.FakeBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient fake balance"})
            }
            if input.IsReal {
                acc.RealBalanceCents -= input.AmountCents
                profile.RealTradeVolume += input.AmountCents
                profile.RealEV += input.EV
            } else {
                acc.FakeBalanceCents -= input.AmountCents
                profile.FakeTradeVolume += input.AmountCents
                profile.FakeEV += input.EV
            }
            acc.TradeHistory = input.Metadata
            if err := db.Save(&acc).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update account"})
            }
            if err := db.Save(&profile).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update profile"})
            }
            transaction = models.Transaction{
                SportsAccountID: acc.ID,
                CustomerID:      customerID,
                Type:            models.TransactionTypeTrade,
                AmountCents:     input.AmountCents,
                IsReal:          input.IsReal,
                Currency:        "KES",
                Status:          models.StatusSuccess,
                Metadata:        JSONMap{"ev": input.EV, "metadata": input.Metadata},
                Reference:       fmt.Sprintf("TRD-%s", uuid.New().String()[:8]),
            }
        case "stock":
            var acc models.StockAccount
            if err := db.Where("customer_id = ? AND sharp_id = ? AND sharp_profile_id = ?", customerID, sharpID, profile.ID).First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            if input.IsReal && acc.RealBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient real balance"})
            }
            if !input.IsReal && acc.FakeBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient fake balance"})
            }
            if input.IsReal {
                acc.RealBalanceCents -= input.AmountCents
                profile.RealTradeVolume += input.AmountCents
                profile.RealEV += input.EV
            } else {
                acc.FakeBalanceCents -= input.AmountCents
                profile.FakeTradeVolume += input.AmountCents
                profile.FakeEV += input.EV
            }
            acc.Portfolio = input.Metadata
            if err := db.Save(&acc).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update account"})
            }
            if err := db.Save(&profile).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update profile"})
            }
            transaction = models.Transaction{
                StockAccountID: acc.ID,
                CustomerID:     customerID,
                Type:           models.TransactionTypeTrade,
                AmountCents:    input.AmountCents,
                IsReal:         input.IsReal,
                Currency:       "KES",
                Status:         models.StatusSuccess,
                Metadata:       JSONMap{"ev": input.EV, "metadata": input.Metadata},
                Reference:      fmt.Sprintf("TRD-%s", uuid.New().String()[:8]),
            }
        case "forex":
            var acc models.ForexAccount
            if err := db.Where("customer_id = ? AND sharp_id = ? AND sharp_profile_id = ?", customerID, sharpID, profile.ID).First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            if input.IsReal && acc.RealBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient real balance"})
            }
            if !input.IsReal && acc.FakeBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient fake balance"})
            }
            if input.IsReal {
                acc.RealBalanceCents -= input.AmountCents
                profile.RealTradeVolume += input.AmountCents
                profile.RealEV += input.EV
            } else {
                acc.FakeBalanceCents -= input.AmountCents
                profile.FakeTradeVolume += input.AmountCents
                profile.FakeEV += input.EV
            }
            acc.OpenPositions = input.Metadata
            if err := db.Save(&acc).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update account"})
            }
            if err := db.Save(&profile).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update profile"})
            }
            transaction = models.Transaction{
                ForexAccountID: acc.ID,
                CustomerID:     customerID,
                Type:           models.TransactionTypeTrade,
                AmountCents:    input.AmountCents,
                IsReal:         input.IsReal,
                Currency:       "KES",
                Status:         models.StatusSuccess,
                Metadata:       JSONMap{"ev": input.EV, "metadata": input.Metadata},
                Reference:      fmt.Sprintf("TRD-%s", uuid.New().String()[:8]),
            }
        case "crypto":
            var acc models.CryptoAccount
            if err := db.Where("customer_id = ? AND sharp_id = ? AND sharp_profile_id = ?", customerID, sharpID, profile.ID).First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            if input.IsReal && acc.RealBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient real balance"})
            }
            if !input.IsReal && acc.FakeBalanceCents < input.AmountCents {
                return c.Status(400).JSON(fiber.Map{"error": "Insufficient fake balance"})
            }
            if input.IsReal {
                acc.RealBalanceCents -= input.AmountCents
                profile.RealTradeVolume += input.AmountCents
                profile.RealEV += input.EV
            } else {
                acc.FakeBalanceCents -= input.AmountCents
                profile.FakeTradeVolume += input.AmountCents
                profile.FakeEV += input.EV
            }
            acc.Addresses = input.Metadata
            if err := db.Save(&acc).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update account"})
            }
            if err := db.Save(&profile).Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to update profile"})
            }
            transaction = models.Transaction{
                CryptoAccountID: acc.ID,
                CustomerID:      customerID,
                Type:            models.TransactionTypeTrade,
                AmountCents:     input.AmountCents,
                IsReal:          input.IsReal,
                Currency:        "KES",
                Status:          models.StatusSuccess,
                Metadata:        JSONMap{"ev": input.EV, "metadata": input.Metadata},
                Reference:       fmt.Sprintf("TRD-%s", uuid.New().String()[:8]),
            }
        default:
            return c.Status(400).JSON(fiber.Map{"error": "Invalid account type"})
        }
        transaction.ID = uuid.New()
        if err := db.Create(&transaction).Error; err != nil {
            return c.Status(500).JSON(fiber.Map{"error": "Failed to create transaction"})
        }
        return c.JSON(fiber.Map{
            "message":        "Trade placed successfully",
            "transaction_id": transaction.ID,
            "reference":      transaction.Reference,
        })
    }
}
