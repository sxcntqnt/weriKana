// api/handlers/login.go
package handlers

import (
    "time"
    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
    "github.com/google/uuid"
    "gorm.io/gorm"
    "weriKana/middleware"
    "weriKana/models"
)

// Login authenticates users and issues a JWT
func Login(db *gorm.DB, secretKey string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var creds struct {
            CustomerID  string `json:"customer_id"`
            SharpID     string `json:"sharp_id"`
            AccountType string `json:"account_type"`
            Password    string `json:"password"`
        }
        if err := c.BodyParser(&creds); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid input"})
        }
        if creds.CustomerID == "" || creds.SharpID == "" || creds.AccountType == "" || creds.Password != "secret" {
            return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials"})
        }
        validTypes := map[string]bool{"sharp": true, "sports": true, "stock": true, "forex": true, "crypto": true}
        if !validTypes[creds.AccountType] {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid account type"})
        }
        customerID, err := uuid.Parse(creds.CustomerID)
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid customer_id"})
        }
        sharpID, err := uuid.Parse(creds.SharpID)
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid sharp_id"})
        }
        switch creds.AccountType {
        case "sharp":
            var acc models.SharpAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                return c.Status(401).JSON(fiber.Map{"error": "Account not found"})
            }
        case "sports":
            var acc models.SportsAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                return c.Status(401).JSON(fiber.Map{"error": "Account not found"})
            }
        case "stock":
            var acc models.StockAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                return c.Status(401).JSON(fiber.Map{"error": "Account not found"})
            }
        case "forex":
            var acc models.ForexAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                return c.Status(401).JSON(fiber.Map{"error": "Account not found"})
            }
        case "crypto":
            var acc models.CryptoAccount
            if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                return c.Status(401).JSON(fiber.Map{"error": "Account not found"})
            }
        }
        expirationTime := time.Now().Add(1 * time.Hour)
        claims := &middleware.Claims{
            CustomerID:  creds.CustomerID,
            SharpID:     creds.SharpID,
            AccountType: creds.AccountType,
            RegisteredClaims: jwt.RegisteredClaims{
                ExpiresAt: jwt.NewNumericDate(expirationTime),
            },
        }
        token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
        tokenString, err := token.SignedString([]byte(secretKey))
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": "Could not create token"})
        }
        return c.JSON(fiber.Map{
            "access_token": tokenString,
            "token_type":   "bearer",
        })
    }
}
