// models/common.go
package models

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "errors"
    "regexp"
)

// --- Regex ---
var kenyanPhoneRegex = regexp.MustCompile(`^(\+254|0)7[0-9]{8}$`)
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

