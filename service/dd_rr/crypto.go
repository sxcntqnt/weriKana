// service/dd_rr/crypto.go
package dd_rr

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha512"
    "io"
    "log"
    "fmt"

    "golang.org/x/crypto/hkdf"
    "weriKana/db"
)

// CryptoEngine — now derived entirely from the master key
type CryptoEngine struct {
    SigningKey ed25519.PrivateKey // 64 bytes — signs withdrawal requests
    AesKey     []byte             // 32 bytes — shared AES-256-GCM key (pre-shared with engine)
}

// Global singleton — initialized once at startup
var Engine *CryptoEngine

// InitCryptoEngine — call this ONCE in main.go
func InitCryptoEngine() {
    masterKey := db.LoadOrGenerateMasterKey() // ← your 32-byte root key
    Engine = deriveFromMasterKey(masterKey)
    log.Println("CryptoEngine initialized from master key (Ed25519 + AES-256-GCM)")
}

// deriveFromMasterKey uses HKDF-SHA512 to split the 32-byte master key into two secure 32-byte keys
func deriveFromMasterKey(masterKey []byte) *CryptoEngine {
    if len(masterKey) != 32 {
        log.Fatalf("Master key must be 32 bytes, got %d", len(masterKey))
    }

    hkdf := hkdf.New(sha512.New, masterKey, nil, []byte("weriKana-withdrawal-engine-v1"))

    signingSeed := make([]byte, 32) // Ed25519 seed
    aesKey := make([]byte, 32)      // AES-256 key

    if _, err := io.ReadFull(hkdf, signingSeed); err != nil {
        log.Fatalf("HKDF failed (signing seed): %v", err)
    }
    if _, err := io.ReadFull(hkdf, aesKey); err != nil {
        log.Fatalf("HKDF failed (AES key): %v", err)
    }

    // Ed25519 private key from 32-byte seed
    signingKey := ed25519.NewKeyFromSeed(signingSeed)

    return &CryptoEngine{
        SigningKey: signingKey,
        AesKey:     aesKey,
    }
}

// PublicKey returns the engine's public key (send to trusted clients once)
func (c *CryptoEngine) PublicKey() ed25519.PublicKey {
    return c.SigningKey.Public().(ed25519.PublicKey)
}

// Sign a withdrawal payload (msgpack bytes)
func (c *CryptoEngine) Sign(data []byte) []byte {
    return ed25519.Sign(c.SigningKey, data)
}

// Verify a signature using the engine's public key
func Verify(pub ed25519.PublicKey, data, sig []byte) bool {
    return ed25519.Verify(pub, data, sig)
}

// EncryptOTP — encrypts OTP so only the execution engine can read it
func (c *CryptoEngine) EncryptOTP(otp string) (nonce, ciphertext []byte, err error) {
    nonce = make([]byte, 12)
    if _, err = rand.Read(nonce); err != nil {
        return nil, nil, err
    }

    block, err := aes.NewCipher(c.AesKey)
    if err != nil {
        return nil, nil, err
    }

    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, err
    }

    ciphertext = aesgcm.Seal(nil, nonce, []byte(otp), nil)
    return nonce, ciphertext, nil
}

// DecryptOTP — used by the execution engine (same key!)
func DecryptOTP(nonce, ciphertext []byte) (string, error) {
    if Engine == nil {
        return "", fmt.Errorf("crypto engine not initialized")
    }

    block, err := aes.NewCipher(Engine.AesKey)
    if err != nil {
        return "", err
    }

    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    plain, err := aesgcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plain), nil
}
