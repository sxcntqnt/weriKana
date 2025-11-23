package handlers

import (
    "github.com/google/uuid"
    "gorm.io/gorm"
    "weriKana/models"
    "github.com/gofiber/fiber/v2" // Import the fiber package for the fiber context
)

// FakeTopup handles the fake top-up of a bookie account's balance
func FakeTopup(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var req struct {
            BookieAccountID uuid.UUID `json:"bookie_account_id"`
            AmountCents     int64     `json:"amount_cents"`
        }

        // Decode the incoming JSON request body
        if err := c.BodyParser(&req); err != nil {
            return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
                "error": "Invalid request body",
            })
        }

        // Find the bookie account in the database
        var acct models.SportsAccount
        if err := db.First(&acct, "id = ?", req.BookieAccountID).Error; err != nil {
            return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
                "error": "Account not found",
            })
        }

        // Update the fake balance for the account
        if err := db.Model(&acct).Update("fake_balance_cents", gorm.Expr("fake_balance_cents + ?", req.AmountCents)).Error; err != nil {
            return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
                "error": "Failed to update balance",
            })
        }

        // Respond with the updated balance
        return c.Status(fiber.StatusOK).JSON(fiber.Map{
            "status":      "fake_credited",
            "new_balance": acct.FakeBalanceCents + req.AmountCents,
        })
    }
}

