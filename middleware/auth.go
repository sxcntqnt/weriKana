// middleware/auth.go
package middleware

import (
	"strings"
        "encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"weriKana/service/keystore" // ← your package path
)

type Claims struct {
	PreferredUsername string   `json:"preferred_username,omitempty"`
	Email             string   `json:"email,omitempty"`
	Name              string   `json:"name,omitempty"`
	Roles             []string `json:"realm_access.roles,omitempty"`
	Groups            []string `json:"groups,omitempty"`
	jwt.RegisteredClaims
}

// KeycloakAuth returns a Fiber middleware that validates Keycloak JWTs offline
func KeycloakAuth(ks *keystore.KeyStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		auth := c.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).
				JSON(fiber.Map{"error": "Missing or invalid Authorization header"})
		}

		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		valid, token, err := ks.VerifyKeycloakUserOffline(tokenStr)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).
				JSON(fiber.Map{"error": "Token validation failed", "details": err.Error()})
		}
		if !valid {
			return c.Status(fiber.StatusUnauthorized).
				JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": "Failed to parse claims"})
		}

		// Optional: unmarshal into structured Claims if you want strong typing
		var kc Claims
		if err := claimsAsStruct(claims, &kc); err != nil {
			// fall back to raw map if needed
			c.Locals("user", claims)
		} else {
			c.Locals("user", kc)
		}

		// Common useful fields
		c.Locals("user_id", claims["sub"])
		c.Locals("username", claims["preferred_username"])
		c.Locals("email", claims["email"])
		c.Locals("roles", getRoles(claims))

		// Pass the raw token and claims downstream if needed
		c.Locals("jwt_token", token)
		c.Locals("jwt_claims", claims)

		return c.Next()
	}
}

// Helper: extract roles (supports both realm_access.roles and resource_access)
func getRoles(claims jwt.MapClaims) []string {
	var roles []string

	if realm, ok := claims["realm_access"].(map[string]any); ok {
		if r, ok := realm["roles"].([]any); ok {
			for _, role := range r {
				if s, ok := role.(string); ok {
					roles = append(roles, s)
				}
			}
		}
	}

	// Add client-specific roles if needed (e.g. resource_access.my-client.roles)
	// if resource, ok := claims["resource_access"].(map[string]any); ok { ... }

	return roles
}

// Optional: safely convert map claims → struct (ignores unknown fields)
func claimsAsStruct(src jwt.MapClaims, dest any) error {
	// Simple but effective: marshal → unmarshal
	data, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}
