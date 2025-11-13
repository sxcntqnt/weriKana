package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "weriKana/models"
)

func GetAccount(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        customerIDStr := c.Locals("customer_id").(string) // From JWT middleware
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
        response := fiber.Map{
            "customer_id":   customerID,
            "sharp_id":      sharpID,
            "account_type":  accountType,
        }
        switch accountType {
        case "sharp":
            var acc models.SharpAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).Preload("Sharp").First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            response["account"] = acc
        case "sports":
            var acc models.SportsAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).Preload("Sharp").First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            response["account"] = acc
        case "stock":
            var acc models.StockAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).Preload("Sharp").First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            response["account"] = acc
        case "forex":
            var acc models.ForexAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).Preload("Sharp").First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            response["account"] = acc
        case "crypto":
            var acc models.CryptoAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).Preload("Sharp").First(&acc).Error; err != nil {
                return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
            }
            response["account"] = acc
        default:
            return c.Status(400).JSON(fiber.Map{"error": "Invalid account type"})
        }
        return c.JSON(response)
    }
}
