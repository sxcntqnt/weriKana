// api/handlers/asset_nexus.go
package handlers

import (
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "weriKana/models"
)

// GetAssetNexus retrieves the customer's AssetNexus with managers and account counts
func GetAssetNexus(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        customerIDStr := c.Locals("customer_id").(string)
        customerID, err := uuid.Parse(customerIDStr)
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid customer_id"})
        }
        var nexus models.AssetNexus
        if err := db.Where("customer_id = ?", customerID).
            Preload("SportsManager").
            Preload("StockManager").
            Preload("ForexManager").
            Preload("CryptoManager").
            Preload("SportsAccounts").
            Preload("StockAccounts").
            Preload("ForexAccounts").
            Preload("CryptoAccounts").
            First(&nexus).Error; err != nil {
            return c.Status(404).JSON(fiber.Map{"error": "AssetNexus not found"})
        }
        response := fiber.Map{
            "customer_id": customerID,
            "asset_nexus": fiber.Map{
                "id":   nexus.ID,
                "name": nexus.Name,
                "managers": fiber.Map{
                    "sports": fiber.Map{
                        "brand":             nexus.SportsManager.Brand,
                        "supported_leagues": nexus.SportsManager.SupportedLeagues,
                    },
                    "stock": fiber.Map{
                        "brand":             nexus.StockManager.Brand,
                        "supported_markets": nexus.StockManager.SupportedMarkets,
                    },
                    "forex": fiber.Map{
                        "brand":           nexus.ForexManager.Brand,
                        "supported_pairs": nexus.ForexManager.SupportedPairs,
                    },
                    "crypto": fiber.Map{
                        "brand":            nexus.CryptoManager.Brand,
                        "supported_coins":  nexus.CryptoManager.SupportedCoins,
                    },
                },
                "accounts": fiber.Map{
                    "sports": len(nexus.SportsAccounts),
                    "stock":  len(nexus.StockAccounts),
                    "forex":  len(nexus.ForexAccounts),
                    "crypto": len(nexus.CryptoAccounts),
                },
            },
        }
        return c.Status(200).JSON(response)
    }
}
