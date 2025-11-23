// api/handlers/refresh.go
package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"weriKana/service/keystore"
)

func RefreshToken(ks *keystore.KeyStore) fiber.Handler {
	const clientID = "werikana-api"
	const vaultSecretPath = "secret/data/werikana"
	const vaultSecretKey = "keycloak_client_secret"

	return func(c *fiber.Ctx) error {
		oldRefreshToken := c.Cookies("refresh_token")
		if oldRefreshToken == "" {
			return c.Status(401).JSON(fiber.Map{"error": "No refresh token"})
		}

		clientSecret, err := ks.GetVaultSecret(vaultSecretPath, vaultSecretKey)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Service unavailable"})
		}

		newToken, err := ks.Keycloak.RefreshToken(
			context.Background(),
			oldRefreshToken,
			clientID,
			clientSecret,
			ks.Realm,
		)
		if err != nil {
			// Invalid/expired refresh token → force re-login
			c.ClearCookie("refresh_token")
			return c.Status(401).JSON(fiber.Map{"error": "Session expired"})
		}

		// Renew the cookie
		c.Cookie(&fiber.Cookie{
			Name:     "refresh_token",
			Value:    newToken.RefreshToken,
			Path:     "/",
			Expires:  time.Now().Add(30 * 24 * time.Hour),
			Secure:   true,
			HTTPOnly: true,
			SameSite: "strict",
		})

		return c.JSON(LoginResponse{
			AccessToken: newToken.AccessToken,
			TokenType:   "bearer",
			ExpiresIn:   newToken.ExpiresIn,
		})
	}
}
