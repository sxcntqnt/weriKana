// api/handlers/login.go
package handlers

import (
	"context"

	"github.com/gofiber/fiber/v2"
	"weriKana/service/keystore"
)

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// KeycloakLogin — uses client secret stored in Vault
func KeycloakLogin(ks *keystore.KeyStore) fiber.Handler {
	const clientID = "werikana-api" // your confidential client in Keycloak

	// Path inside Vault where you store the secret, e.g.:
	// vault kv put secret/data/werikana keycloak_client_secret=super-secret-123
	const vaultSecretPath = "secret/data/werikana"
	const vaultSecretKey  = "keycloak_client_secret"

	return func(c *fiber.Ctx) error {
		var req LoginRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
		}
		if req.Username == "" || req.Password == "" {
			return c.Status(400).JSON(fiber.Map{"error": "username and password required"})
		}

		// === Read client secret from Vault (cached in memory after first read) ===
		secret, err := ks.GetVaultSecret(vaultSecretPath, vaultSecretKey)
		if err != nil {
			// Don't leak Vault errors to the client
			return c.Status(500).JSON(fiber.Map{"error": "Internal authentication error"})
		}

		// === Perform Keycloak login ===
		token, err := ks.Login(context.Background(), clientID, secret, req.Username, req.Password)
		if err != nil {
			return c.Status(401).JSON(fiber.Map{"error": "Invalid username or password"})
		}

		return c.JSON(LoginResponse{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			TokenType:    "bearer",
			ExpiresIn:    token.ExpiresIn,
		})
	}
}
