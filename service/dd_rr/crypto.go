// service/dd_rr/crypto.go
package dd_rr

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/ed25519"
    "crypto/rand"
    "crypto/sha512"
    "encoding/base64"
    "fmt"
    "io"
    "log"

    "golang.org/x/crypto/hkdf"
    
    // Use your clean wrapper instead of reaching into masterkey directly
    "weriKana/utils"
)

type CryptoEngine struct {
    SigningKey ed25519.PrivateKey // 64 bytes — signs withdrawal requests
    AesKey     []byte              // 32 bytes — AES-256-GCM key shared with engine
}

var Engine *CryptoEngine

// InitCryptoEngine — call once in main.go (after masterkey.Init())
func InitCryptoEngine() {
    // This is now the permanent, Vault-backed, deterministic master key
    masterKey := utils.AESMasterKey()

    Engine = deriveFromMasterKey(masterKey)
    
    log.Println("DD/RR CryptoEngine initialized successfully")
    log.Printf("  Engine Ed25519 public key: %s", base64.StdEncoding.EncodeToString(Engine.PublicKey()))
    log.Printf("  SecureBus identity public key: %s", utils.IdentityPublicB64())
}

// deriveFromMasterKey — unchanged, perfect as-is
func deriveFromMasterKey(masterKey []byte) *CryptoEngine {
    hkdf := hkdf.New(sha512.New, masterKey, nil, []byte("weriKana-withdrawal-engine-v1"))

    signingSeed := make([]byte, 32)
    aesKey := make([]byte, 32)

    if _, err := io.ReadFull(hkdf, signingSeed); err != nil {
        log.Fatalf("HKDF failed (signing seed): %v", err)
    }
    if _, err := io.ReadFull(hkdf, aesKey); err != nil {
        log.Fatalf("HKDF failed (AES key): %v", err)
    }

    return &CryptoEngine{
        SigningKey: ed25519.NewKeyFromSeed(signingSeed),
        AesKey:     aesKey,
    }
}

func (c *CryptoEngine) PublicKey() ed25519.PublicKey {
    return c.SigningKey.Public().(ed25519.PublicKey)
}

func (c *CryptoEngine) Sign(data []byte) []byte {
    return ed25519.Sign(c.SigningKey, data)
}

func Verify(pub ed25519.PublicKey, data, sig []byte) bool {
    return ed25519.Verify(pub, data, sig)
}

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

func DecryptOTP(nonce, ciphertext []byte) (string, error) {
    if Engine == nil {
        return "", fmt.Errorf("dd_rr crypto engine not initialized")
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
        return "", fmt.Errorf("invalid or tampered OTP: %w", err)
    }
    return string(plain), nil
}
