// models/common.go
package models

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "regexp"
    "database/sql/driver"
    "encoding/json"
	"fmt"
)

// --- Regex ---
// Kenyan phone regex for validation (e.g., +2540/1/7xxxxxxxx or 00/1/7xxxxxxxx; includes 011 landlines).
var kenyanPhoneRegex = regexp.MustCompile(`^\+254[01]\d{8}$|^0[01]\d{8}$`)

// PhoneNumbers is a JSON-serializable slice for multiple phones (primary + fallbacks).
type PhoneNumbers []string

// Value implements Valuer for SQL serialization.
func (p PhoneNumbers) Value() (driver.Value, error) {
        if len(p) == 0 {
                return nil, nil // Null if empty
        }
        b, err := json.Marshal(p)
        return string(b), err
}

// Scan implements Scanner for SQL deserialization.
func (p *PhoneNumbers) Scan(value interface{}) error {
        if value == nil {
                *p = nil
                return nil
        }
        b, ok := value.([]byte)
        if !ok {
                return fmt.Errorf("PhoneNumbers: cannot scan []byte")
        }
        return json.Unmarshal(b, p)
}
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// --- Enums ---
type GenderType string
const (
    GenderMale   GenderType = "male"
    GenderFemale GenderType = "female"
    GenderOther  GenderType = "other"
)

type IdentificationType string
const (
    IDNational   IdentificationType = "national_id"
    IDPassport   IdentificationType = "passport"
    IDAlien      IdentificationType = "alien_card"
)

type BankAccountType string
const (
    BankAccountTypeCurrent BankAccountType = "current"
    BankAccountTypeSavings BankAccountType = "savings"
)

// --- Encryption Key (same for sender/recipient) ---
var encryptionKey = []byte("32-byte-key-for-aes-256-gcm!!!!!") // 32 bytes = AES-256

// --- Encrypt Helper ---
func encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(encryptionKey)
    if err != nil {
        return "", err
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonce := make([]byte, aesGCM.NonceSize())
    if _, err = rand.Read(nonce); err != nil {
        return "", err
    }

    ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// --- Decrypt Helper ---
func decrypt(ciphertextB64 string) (string, error) {
    data, err := base64.StdEncoding.DecodeString(ciphertextB64)
    if err != nil {
        return "", err
    }

    block, err := aes.NewCipher(encryptionKey)
    if err != nil {
        return "", err
    }

    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }

    nonceSize := aesGCM.NonceSize()
    if len(data) < nonceSize {
        return "", errors.New("ciphertext too short")
    }

    nonce, ciphertext := data[:nonceSize], data[nonceSize:]
    plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return "", err
    }

    return string(plaintext), nil
}

// BalanceUpdater lets you add real/fake money in a type-safe way across all account types
type BalanceUpdater interface {
    AddReal(cents int64)
    AddFake(cents int64)
}

// ─────────────────────────────────────────────────────────────────────────────
// SportsAccount
// ─────────────────────────────────────────────────────────────────────────────
func (a *SportsAccount) AddReal(cents int64) {
    a.RealBalanceCents += cents
}

func (a *SportsAccount) AddFake(cents int64) {
    a.FakeBalanceCents += cents
}

// ─────────────────────────────────────────────────────────────────────────────
// StockAccount
// ─────────────────────────────────────────────────────────────────────────────
func (a *StockAccount) AddReal(cents int64) {
    a.RealBalanceCents += cents
}

func (a *StockAccount) AddFake(cents int64) {
    a.FakeBalanceCents += cents
}

// ─────────────────────────────────────────────────────────────────────────────
// ForexAccount
// ─────────────────────────────────────────────────────────────────────────────
func (a *ForexAccount) AddReal(cents int64) {
    a.RealBalanceCents += cents
}

func (a *ForexAccount) AddFake(cents int64) {
    a.FakeBalanceCents += cents
}

// ─────────────────────────────────────────────────────────────────────────────
// CryptoAccount
// ─────────────────────────────────────────────────────────────────────────────
func (a *CryptoAccount) AddReal(cents int64) {
    a.RealBalanceCents += cents
}

func (a *CryptoAccount) AddFake(cents int64) {
    a.FakeBalanceCents += cents
}
