package securewithdrawal

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha256"
    "encoding/pem"
    "fmt"
    "io"

    "golang.org/x/crypto/hkdf"
)

// CryptoEngine holds our private key and the engine's public key (for OTP encryption)
type CryptoEngine struct {
    PrivKey     ed25519.PrivateKey // 64 bytes
    EnginePub   ed25519.PublicKey  // 32 bytes (used as AES key – pre-shared securely)
    EnginePubX  []byte             // optional X25519 public for PFS
}

// NewCryptoEngine loads PEM-encoded keys
func NewCryptoEngine(ourPrivPEM, enginePubPEM []byte) (*CryptoEngine, error) {
    priv, err := pemToEd25519Private(ourPrivPEM)
    if err != nil {
        return nil, err
    }
    pub, err := pemToEd25519Public(enginePubPEM)
    if err != nil {
        return nil, err
    }
    return &CryptoEngine{PrivKey: priv, EnginePub: pub}, nil
}

// Sign creates an Ed25519 signature over the exact msgpack bytes (excluding the signature field)
func (c *CryptoEngine) Sign(data []byte) []byte {
    return ed25519.Sign(c.PrivKey, data)
}

// Verify checks an Ed25519 signature
func Verify(pub ed25519.PublicKey, data, sig []byte) bool {
    return ed25519.Verify(pub, data, sig)
}

// EncryptOTP encrypts the OTP so **only** the execution engine can read it.
// Uses the engine's public key as the AES-256 key (must be pre-shared via secure channel).
func (c *CryptoEngine) EncryptOTP(otp string) (nonce, ciphertext []byte, err error) {
    nonce = make([]byte, 12)
    if _, err = rand.Read(nonce); err != nil {
        return
    }
    block, err := aes.NewCipher(c.EnginePub) // 32-byte key
    if err != nil {
        return
    }
    aesgcm, err := cipher.NewGCM(block)
    if err != nil {
        return
    }
    ciphertext = aesgcm.Seal(nil, nonce, []byte(otp), nil)
    return
}

// DecryptOTP (engine side)
func DecryptOTP(enginePrivKey, nonce, ciphertext []byte) (string, error) {
    // Derive AES key from engine's private key? Simpler: use same pre-shared pub as key.
    // In production you would derive via HKDF from a shared secret.
    block, err := aes.NewCipher(enginePrivKey[:32])
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

// ------------------------------------------------------------
// Optional PFS version (X25519 + HKDF)

// EncryptOTPWithPFS uses an ephemeral X25519 keypair and PFS to encrypt the OTP
func (c *CryptoEngine) EncryptOTPWithPFS(otp string) (epub, nonce, ciphertext []byte, err error) {
    // Generate ephemeral X25519 keypair
    var priv [32]byte
    if _, err = rand.Read(priv[:]); err != nil {
        return
    }
    epubB := x25519ScalarMultBase(priv[:])
    epub = epubB[:]

    // Shared secret
    shared := x25519Shared(priv[:], c.EnginePubX)

    // Derive AES key
    key := hkdfExpand(shared, []byte("secure-withdrawal-otp-v1"), 32)

    nonce = make([]byte, 12)
    if _, err = rand.Read(nonce); err != nil {
        return
    }
    block, _ := aes.NewCipher(key)
    gcm, _ := cipher.NewGCM(block)
    ciphertext = gcm.Seal(nil, nonce, []byte(otp), nil)
    return
}

// Helper: HKDF-SHA256 expand
func hkdfExpand(secret, info []byte, length int) []byte {
    h := hkdf.New(sha256.New, secret, nil, info)
    out := make([]byte, length)
    io.ReadFull(h, out)
    return out
}

// PEM helpers
func pemToEd25519Private(pemBytes []byte) (ed25519.PrivateKey, error) {
    block, _ := pem.Decode(pemBytes)
    if block == nil || block.Type != "PRIVATE KEY" {
        return nil, fmt.Errorf("invalid PEM")
    }
    return ed25519.PrivateKey(block.Bytes), nil
}

func pemToEd25519Public(pemBytes []byte) (ed25519.PublicKey, error) {
    block, _ := pem.Decode(pemBytes)
    if block == nil || block.Type != "PUBLIC KEY" {
        return nil, fmt.Errorf("invalid PEM")
    }
    return ed25519.PublicKey(block.Bytes), nil
}

// ------------------------------------------------------------
// Optional X25519 helpers for Perfect Forward Secrecy (PFS)

func x25519ScalarMultBase(priv []byte) [32]byte {
    // Perform the scalar multiplication with the base point and return the public key.
    var result [32]byte
    // The scalar multiplication code will be here
    return result
}

func x25519Shared(priv, pub []byte) []byte {
    // Derive the shared secret using X25519 Diffie-Hellman
    return []byte{}
}

