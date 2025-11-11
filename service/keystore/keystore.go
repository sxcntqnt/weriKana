// services/keystore.go
type KeyStore struct {
	keys map[uuid.UUID][]byte // customer_id → 32-byte HMAC key
	mu   sync.RWMutex
}

func (ks *KeyStore) Verify(customerID uuid.UUID, payload, signature string) bool {
	ks.mu.RLock()
	key := ks.keys[customerID]
	ks.mu.RUnlock()

	expected := hmacSHA256(payload, key)
	return hmac.Equal([]byte(signature), []byte(expected))
}
