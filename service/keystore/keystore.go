// service/keystore/keystore.go
package keystore

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hashicorp/vault/api"
)

// Add a tiny cache so we don't hit Vault on every login
var secretCache struct {
        sync.RWMutex
        values map[string]string
}

func init() {
        secretCache.values = make(map[string]string)
}

type KeyStore struct {
	vaultClient *api.Client
	vaultPath   string

	Keycloak    *gocloak.GoCloak   // exported
	KeycloakURL string             // exported
	Realm       string             // exported

	jwtParser    *jwt.Parser
	cachedCerts  *gocloak.CertResponse
	lastRefresh  time.Time
	refreshMutex sync.RWMutex
	mu           sync.RWMutex
}

func NewKeyStore(vaultAddr, vaultToken, vaultPath, keycloakURL, keycloakRealm string) (*KeyStore, error) {
	config := api.DefaultConfig()
	if config == nil {
		return nil, fmt.Errorf("failed to create vault config")
	}
	config.Address = vaultAddr

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}
	client.SetToken(vaultToken)

	kc := gocloak.NewClient(keycloakURL)

	return &KeyStore{
		vaultClient: client,
		vaultPath:   vaultPath,
		Keycloak:    kc,           // exported
		KeycloakURL: keycloakURL,  // exported
		Realm:       keycloakRealm, // exported
		jwtParser:   jwt.NewParser(jwt.WithValidMethods([]string{"RS256"})),
	}, nil
}

func (ks *KeyStore) fetchOrRefreshCerts(ctx context.Context) (*gocloak.CertResponse, error) {
	ks.refreshMutex.RLock()
	if ks.cachedCerts != nil && time.Since(ks.lastRefresh) < 10*time.Minute {
		defer ks.refreshMutex.RUnlock()
		return ks.cachedCerts, nil
	}
	ks.refreshMutex.RUnlock()

	ks.refreshMutex.Lock()
	defer ks.refreshMutex.Unlock()

	if ks.cachedCerts != nil && time.Since(ks.lastRefresh) < 10*time.Minute {
		return ks.cachedCerts, nil
	}

	certs, err := ks.Keycloak.GetCerts(ctx, ks.Realm) // exported fields
	if err != nil {
		if ks.cachedCerts != nil {
			return ks.cachedCerts, nil
		}
		return nil, fmt.Errorf("failed to fetch Keycloak certs: %w", err)
	}

	ks.cachedCerts = certs
	ks.lastRefresh = time.Now()
	return certs, nil
}

func jwkToRSA(jwk gocloak.CertResponseKey) (*rsa.PublicKey, error) {
	if jwk.Kty == nil || *jwk.Kty != "RSA" {
		return nil, fmt.Errorf("not an RSA key")
	}
	if jwk.N == nil || jwk.E == nil {
		return nil, fmt.Errorf("missing n or e")
	}

	nBytes, _ := base64.RawURLEncoding.DecodeString(*jwk.N)
	eBytes, _ := base64.RawURLEncoding.DecodeString(*jwk.E)

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}

func (ks *KeyStore) VerifyKeycloakUserOffline(accessToken string) (bool, *jwt.Token, error) {
	ctx := context.Background()
	certs, err := ks.fetchOrRefreshCerts(ctx)
	if err != nil {
		return false, nil, err
	}

	token, err := ks.jwtParser.Parse(accessToken, func(t *jwt.Token) (any, error) {
		kidRaw, ok := t.Header["kid"]
		if !ok {
			return nil, fmt.Errorf("kid header missing")
		}
		kid := kidRaw.(string)

		if certs.Keys == nil {
			return nil, fmt.Errorf("no keys in JWKS")
		}
		for _, key := range *certs.Keys {
			if key.Kid != nil && *key.Kid == kid {
				return jwkToRSA(key)
			}
		}
		return nil, fmt.Errorf("no matching key for kid=%s", kid)
	})

	if err != nil {
		return false, nil, fmt.Errorf("invalid token: %w", err)
	}
	if !token.Valid {
		return false, token, nil
	}

	// Verify issuer
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		expected := strings.TrimSuffix(ks.KeycloakURL, "/") + "/realms/" + ks.Realm
		if iss, _ := claims["iss"].(string); iss != expected {
			return false, token, fmt.Errorf("wrong issuer: %s", iss)
		}
	}

	return true, token, nil
}

func (ks *KeyStore) IsTokenValid(token string) (bool, error) {
	ok, _, err := ks.VerifyKeycloakUserOffline(token)
	return ok, err
}

// Optional helper for Direct Access Grant login
func (ks *KeyStore) Login(ctx context.Context, clientID, clientSecret, username, password string) (*gocloak.JWT, error) {
	return ks.Keycloak.Login(ctx, clientID, clientSecret, ks.Realm, username, password)
}

// GetVaultSecret reads a secret from Vault with in-memory caching
func (ks *KeyStore) GetVaultSecret(path, key string) (string, error) {
	cacheKey := path + "#" + key

	// Fast path: return cached value
	secretCache.RLock()
	if val, ok := secretCache.values[cacheKey]; ok {
		secretCache.RUnlock()
		return val, nil
	}
	secretCache.RUnlock()

	// Slow path: read from Vault
	secretCache.Lock()
	defer secretCache.Unlock()

	// Double-check in case another goroutine already fetched it
	if val, ok := secretCache.values[cacheKey]; ok {
		return val, nil
	}

	secret, err := ks.vaultClient.Logical().Read(path)
	if err != nil {
		return "", err
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("secret not found at %s", path)
	}

	// Vault v2 KV: data is under "data" → "data"
	dataMap, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid secret format at %s", path)
	}

	value, ok := dataMap[key].(string)
	if !ok || value == "" {
		return "", fmt.Errorf("key %s not found or empty in %s", key, path)
	}

	// Cache it forever (or until restart — good enough for most services)
	secretCache.values[cacheKey] = value
	return value, nil
}
