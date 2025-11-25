// models/sender.go
// Sender model — Updated for Sharp merge: FK to SharpID (replaces CustomerID); UUID gen in BeforeCreate.
// 1:1 with Sharp for sender details (person/business verification).

package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SenderType string

const (
	SenderTypePerson   SenderType = "person"
	SenderTypeBusiness SenderType = "business"
)

type Sender struct {
	gorm.Model
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID             uuid.UUID  `gorm:"type:uuid;uniqueIndex;not null"` // 1:1 with Sharp (replaces CustomerID)
	Type                SenderType
	FirstName           string `gorm:"size:100;not null"`
	LastName            string `gorm:"size:100;not null"`
	BirthDate           time.Time
	Gender              GenderType
	IdentificationType  IdentificationType
	IdentificationNumber string `gorm:"size:50"`
	IdentificationExpiry time.Time
	CountryCode         string `gorm:"size:2;default:KE"`
	Street              string `gorm:"size:255"`
	PostalCode          string `gorm:"size:20"`
	City                string `gorm:"size:100"`
	// Encrypted Fields
	EmailEnc            string `gorm:"column:email_enc;size:500"`
	PhoneNumberEnc      string `gorm:"column:phone_enc;size:500"`
	IPAddressEnc        string `gorm:"column:ip_address_enc;size:500"`
	// Plaintext
	ExternalID          string `gorm:"size:100;uniqueIndex"`
	// Relationships
	Sharp               Sharp  `gorm:"foreignKey:SharpID;constraint:OnDelete:RESTRICT"`
}

// Table name
func (Sender) TableName() string {
	return "senders"
}

// BeforeCreate GORM hook → validate SharpID & generate UUID.
func (s *Sender) BeforeCreate(tx *gorm.DB) error {
	if s.SharpID == uuid.Nil {
		return errors.New("sharp_id required")
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// Setters with Encryption & Validation
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
