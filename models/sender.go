// models/sender.go
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
type SenderType string
const (
	SenderTypePerson   SenderType = "person"
	SenderTypeBusiness SenderType = "business"
)

type GenderType string
const (
	GenderTypeMale   GenderType = "male"
	GenderTypeFemale GenderType = "female"
	GenderTypeOther  GenderType = "other"
)

type IdentificationType string
const (
	IdentificationTypePassport      IdentificationType = "passport"
	IdentificationTypeNationalId    IdentificationType = "national_id"
	IdentificationTypeDrivingLicense IdentificationType = "driving_license"
	IdentificationTypeOther        IdentificationType = "other"
)

// --- Regex ---
var kenyanPhoneRegex = regexp.MustCompile(`^(\+254|0)7[0-9]{8}$`)
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// --- Encryption Key (same as recipient.go) ---
var encryptionKey = []byte("32-byte-key-for-aes-256-gcm!!!!!") // 32 bytes = AES-256

// --- Sender ---
type Sender struct {
	gorm.Model
	ID                   uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	CustomerID           uuid.UUID `gorm:"type:uuid;uniqueIndex;not null"` // 1:1 with Customer
	Type                 SenderType
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
	EmailEnc       string `gorm:"column:email_enc;size:500"`
	PhoneNumberEnc string `gorm:"column:phone_enc;size:500"`
	IPAddressEnc   string `gorm:"column:ip_address_enc;size:500"`

	// Plaintext
	ExternalID string `gorm:"size:100;uniqueIndex"`

	// Relationships
	Customer Customer `gorm:"foreignKey:CustomerID;constraint:OnDelete:RESTRICT"`
}

// Table name
func (Sender) TableName() string {
	return "senders"
}

// --- Transient Getters ---
func (s *Sender) GetEmail() (string, error)     { return decrypt(s.EmailEnc) }
func (s *Sender) GetPhone() (string, error)     { return decrypt(s.PhoneNumberEnc) }
func (s *Sender) GetIPAddress() (string, error) { return decrypt(s.IPAddressEnc) }

// --- Setters with Encryption & Validation ---
func (s *Sender) SetEmail(email string) error {
	if !emailRegex.MatchString(email) {
		return errors.New("invalid email")
	}
	enc, err := encrypt(email)
	if err != nil {
		return err
	}
	s.EmailEnc = enc
	return nil
}

func (s *Sender) SetPhone(phone string) error {
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
	s.PhoneNumberEnc = enc
	return nil
}

func (s *Sender) SetIPAddress(ip string) error {
	enc, err := encrypt(ip)
	if err != nil {
		return err
	}
	s.IPAddressEnc = enc
	return nil
}

// --- GORM Hooks ---
func (s *Sender) BeforeCreate(tx *gorm.DB) error {
	if s.CustomerID == uuid.Nil {
		return errors.New("customer_id required")
	}
	return nil
}

// --- Reuse Encryption from recipient.go ---
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
