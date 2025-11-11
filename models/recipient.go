// models/recipient.go
package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Enums ---
type BankAccountType string
const (
	BankAccountTypeCurrent BankAccountType = "current"
	BankAccountTypeSavings BankAccountType = "savings"
)

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

// --- Regex ---
var kenyanPhoneRegex = regexp.MustCompile(`^(\+254|0)7[0-9]{8}$`)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// --- Global Encryption Key (in prod: load from Vault/KMS) ---
var encryptionKey = []byte("32-byte-key-for-aes-256-gcm!!!!!") // 32 bytes = AES-256

// --- Recipient ---
type Recipient struct {
	gorm.Model
	ID                   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CustomerID           uuid.UUID `gorm:"type:uuid;index"` // links to Customer
	FirstName            string    `gorm:"size:100;not null"`
	LastName             string    `gorm:"size:100;not null"`
	BirthDate            time.Time
	Gender               GenderType
	IdentificationType   IdentificationType
	IdentificationNumber string `gorm:"size:50"`
	IdentificationExpiry  time.Time
	CountryCode          string `gorm:"size:2;default:'KE'"`
	Street               string `gorm:"size:255"`
	PostalCode           string `gorm:"size:20"`
	City                 string `gorm:"size:100"`

	// Encrypted Fields
	EmailEnc          string `gorm:"column:email_enc;size:500"`
	PhoneNumberEnc    string `gorm:"column:phone_enc;size:500"`
	BankNameEnc       string `gorm:"column:bank_name_enc;size:500"`
	BankCodeEnc       string `gorm:"column:bank_code_enc;size:500"`
	BankAccountEnc    string `gorm:"column:bank_account_enc;size:500"`
	SortCodeEnc       string `gorm:"column:sort_code_enc;size:500"`
	IBANEnc           string `gorm:"column:iban_enc;size:500"`
	BICEnc            string `gorm:"column:bic_enc;size:500"`

	// Plaintext (non-sensitive)
	BankAccountType BankAccountType `gorm:"size:20"`
	TransferReasonCode string       `gorm:"size:50"`
	ExternalID         string       `gorm:"size:100;uniqueIndex"`
	MobileProvider     string       `gorm:"size:50"`

	// Relationships
	Customer Customer `gorm:"foreignKey:CustomerID"`
}

// Table name
func (Recipient) TableName() string {
	return "recipients"
}

// --- Transient Fields (not stored) ---
func (r *Recipient) GetEmail() (string, error)    { return decrypt(r.EmailEnc) }
func (r *Recipient) GetPhone() (string, error)    { return decrypt(r.PhoneNumberEnc) }
func (r *Recipient) GetBankAccount() (string, error) { return decrypt(r.BankAccountEnc) }

// --- Setters with Encryption ---
func (r *Recipient) SetEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email")
	}
	enc, err := encrypt(email)
	if err != nil {
		return err
	}
	r.EmailEnc = enc
	return nil
}

func (r *Recipient) SetPhone(phone string) error {
	if !kenyanPhoneRegex.MatchString(phone) {
		return errors.New("invalid Kenyan phone")
	}
	if phone[0] == '0' {
		phone = "+254" + phone[1:]
	}
	enc, err := encrypt(phone)
	if err != nil {
		return err
	}
	r.PhoneNumberEnc = enc
	return nil
}

func (r *Recipient) SetBankAccount(acc string) error {
	enc, err := encrypt(acc)
	if err != nil {
		return err
	}
	r.BankAccountEnc = enc
	return nil
}

// --- GORM Hooks ---
func (r *Recipient) BeforeCreate(tx *gorm.DB) error {
	if r.CustomerID == uuid.Nil {
		return errors.New("customer_id required")
	}
	return nil
}

// --- Encryption Helpers ---
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
