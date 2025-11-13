// middleware/auth.go
package middleware

import (
    "github.com/gofiber/fiber/v2"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    CustomerID  string `json:"customer_id"`
    SharpID     string `json:"sharp_id"`
    AccountType string `json:"account_type"`
    jwt.RegisteredClaims
}

func AuthMiddleware(secretKey string) fiber.Handler {
    return func(c *fiber.Ctx) error {
        authHeader := c.Get("Authorization")
        if authHeader == "" || len(authHeader) < 7 || authHeader[:7] != "Bearer " {
            return c.Status(401).JSON(fiber.Map{"error": "Missing or invalid token"})
        }
        tokenString := authHeader[7:]
        token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fiber.NewError(fiber.StatusUnauthorized, "Unexpected signing method")
            }
            return []byte(secretKey), nil
        })
        if err != nil || !token.Valid {
            return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
        }
        claims, ok := token.Claims.(*Claims)
        if !ok {
            return c.Status(401).JSON(fiber.Map{"error": "Invalid token claims"})
        }
        c.Locals("customer_id", claims.CustomerID)
        c.Locals("sharp_id", claims.SharpID)
        c.Locals("account_type", claims.AccountType)
        return c.Next()
    }
}
