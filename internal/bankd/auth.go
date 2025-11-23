// internal/bankd/auth.go
package bankd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gliderlabs/ssh"
)

type contextKey string

const (
	partnerIDKey contextKey = "partnerID"
	sessionKey   contextKey = "session"
)

// GitAuthorizedKeys reads authorized_keys files from a git-style repo layout.
type GitAuthorizedKeys struct {
	repoPath string
}

// NewGitAuthorizedKeys creates a new GitAuthorizedKeys backed by repoPath.
func NewGitAuthorizedKeys(repoPath string) *GitAuthorizedKeys {
	return &GitAuthorizedKeys{repoPath: repoPath}
}

// GetAuthorizedKeys returns parsed public keys; if a per-user file exists it uses that.
func (g *GitAuthorizedKeys) GetAuthorizedKeys(user string) ([]ssh.PublicKey, error) {
	keysPath := filepath.Join(g.repoPath, "authorized_keys")
	if user != "" {
		userPath := filepath.Join(g.repoPath, "users", user, "authorized_keys")
		if _, err := os.Stat(userPath); err == nil {
			keysPath = userPath
		}
	}
	return g.parseAuthorizedKeys(keysPath)
}

// parseAuthorizedKeys parses an OpenSSH authorized_keys file and returns slice of keys.
func (g *GitAuthorizedKeys) parseAuthorizedKeys(path string) ([]ssh.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var keys []ssh.PublicKey
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line)); err == nil {
			keys = append(keys, pubKey)
		}
	}
	return keys, nil
}

// Authenticate checks whether the provided public key exists in the repo.
// It returns a partnerID (string) and a bool indicating whether the key matched.
func (g *GitAuthorizedKeys) Authenticate(pub ssh.PublicKey, user string) (string, bool) {
	keys, err := g.GetAuthorizedKeys(user)
	if err != nil {
		log.Printf("GitAuthorizedKeys: error reading keys: %v", err)
		return "", false
	}
	for _, ak := range keys {
		if ssh.KeysEqual(pub, ak) {
			partnerID := DerivePartnerIDFromKey(pub, user)
			return partnerID, true
		}
	}
	return "", false
}

// PublicKeyAuthHandler returns a function suitable for ssh.PublicKeyAuth that uses the provided GitAuthorizedKeys.
func PublicKeyAuthHandler(g *GitAuthorizedKeys) func(ctx ssh.Context, key ssh.PublicKey) bool {
	return func(ctx ssh.Context, key ssh.PublicKey) bool {
		partnerID, ok := g.Authenticate(key, ctx.User())
		if !ok {
			return false
		}
		// store partner ID for later use by the session handler
		ctx.SetValue(partnerIDKey, partnerID)
		return true
	}
}

// PasswordAuthHandler is a PasswordAuth function suitable for ssh.PasswordAuth(...).
// It tries to read SECRET (or PARTNER_SECRET) from the session environ (SetEnv).
// For this to work, the session must be stored on the context with key "session" early
// in your session handler (see instructions in the repo).
func PasswordAuthHandler(ctx ssh.Context, password string) bool {
	// Attempt to get the session previously stored on the connection context.
	// Your session handler should call: s.Context().SetValue("session", s)
	sessVal := ctx.Value(sessionKey)
	if sessVal != nil {
		if sess, ok := sessVal.(ssh.Session); ok {
			for _, e := range sess.Environ() {
				// each entry is "KEY=value"
				if strings.HasPrefix(e, "SECRET=") {
					secret := strings.TrimPrefix(e, "SECRET=")
					if ValidateSecret(secret) {
						ctx.SetValue(partnerIDKey, DerivePartnerIDFromSecret(secret))
						return true
					}
				}
				if strings.HasPrefix(e, "PARTNER_SECRET=") {
					secret := strings.TrimPrefix(e, "PARTNER_SECRET=")
					if ValidateSecret(secret) {
						ctx.SetValue(partnerIDKey, DerivePartnerIDFromSecret(secret))
						return true
					}
				}
			}
		}
	}
	// Fallback: treat the password supplied as the secret.
	if ValidateSecret(password) {
		ctx.SetValue(partnerIDKey, DerivePartnerIDFromSecret(password))
		return true
	}
	return false
}

// ValidateSecret performs simple validation of the secret.
// Replace with your real validation (database, HMAC verification, etc.).
func ValidateSecret(secret string) bool {
	return secret != "" && len(secret) > 8
}

// DerivePartnerIDFromSecret deterministically derives a partner ID from a secret.
func DerivePartnerIDFromSecret(secret string) string {
	return fmt.Sprintf("partner_%s", shortHex(simpleHash(secret)))
}

// DerivePartnerIDFromKey derives a partner id from the key and username (simple example).
func DerivePartnerIDFromKey(key ssh.PublicKey, user string) string {
	// Use user and short hash of key bytes so derived ID is stable and human-readable.
	return fmt.Sprintf("partner_%s_%s", user, shortHex(simpleHash(string(key.Marshal()))))
}

// simpleHash returns a SHA-256 hash (as bytes) of the given input string.
func simpleHash(s string) []byte {
	h := sha256.Sum256([]byte(s))
	return h[:] // full 32 bytes
}

// shortHex returns the first 8 hex chars of the hash for compact IDs.
func shortHex(b []byte) string {
	hexs := hex.EncodeToString(b)
	if len(hexs) > 8 {
		return hexs[:8]
	}
	return hexs
}
