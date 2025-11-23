// api/handlers/login.go
// Final Version — 2025 Elite Edition
// Includes: httpOnly refresh token + secure cookie settings
package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"weriKana/service/keystore"
)

type LoginRequest struct {
	Username string `json:"username"` // phone number
	Password string `json:"password"` // OTP
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// KeycloakLogin — THE GOLD STANDARD (2025)
func KeycloakLogin(ks *keystore.KeyStore) fiber.Handler {
	const clientID = "werikana-api"
	const vaultSecretPath = "secret/data/werikana"
	const vaultSecretKey = "keycloak_client_secret"

	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if req.Username == "" || req.Password == "" {
			return c.Status(400).JSON(fiber.Map{"error": "username and password required"})
		}

		// 1. Get client secret from Vault (cached forever in RAM)
		clientSecret, err := ks.GetVaultSecret(vaultSecretPath, vaultSecretKey)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Authentication service unavailable"})
		}

		// 2. Exchange OTP for real Keycloak tokens
		token, err := ks.Keycloak.Login(
			context.Background(),
			clientID,
			clientSecret,
			ks.Realm,
			req.Username,
			req.Password,
		)
		if err != nil {
			// Don't leak details — Keycloak already logged it
			return c.Status(401).JSON(fiber.Map{"error": "Invalid credentials or OTP expired"})
		}

		// 3. Set refresh token as httpOnly, Secure, Strict cookie
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    token.RefreshToken,
			Path:     "/",                                 // Available everywhere (or "/auth/refresh" if you prefer)
			Expires:  time.Now().Add(30 * 24 * time.Hour), // 30 days — standard
			Secure:   true,                                // HTTPS only
			HTTPOnly: true,                                // JS cannot touch it
			SameSite: "strict",                            // Best anti-CSRF protection
		})

		// 4. Return only the short-lived access token
		return c.JSON(LoginResponse{
			AccessToken: token.AccessToken,
			TokenType:   "bearer",
			ExpiresIn:   token.ExpiresIn,
		})
	}
}
