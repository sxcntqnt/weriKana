package keystore

import (
    "fmt"
    "sync"
    "github.com/google/uuid"
    "crypto/hmac"
    "crypto/sha256"
)

type KeyStore struct {
    keys map[uuid.UUID][]byte // customer_id → 32-byte HMAC key
    mu   sync.RWMutex
}

// hmacSHA256 computes the HMAC-SHA256 of a given payload using a given key.
func hmacSHA256(payload string, key []byte) string {
    mac := hmac.New(sha256.New, key)
    mac.Write([]byte(payload))
    return fmt.Sprintf("%x", mac.Sum(nil)) // Return the result as a hexadecimal string
}

func (ks *KeyStore) Verify(customerID uuid.UUID, payload, signature string) bool {
    ks.mu.RLock()
    key := ks.keys[customerID]
    ks.mu.RUnlock()

    expected := hmacSHA256(payload, key)
    return hmac.Equal([]byte(signature), []byte(expected))
}

