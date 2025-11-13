// models/sender.go
package models

import (
    "errors"
    "gorm.io/gorm"
    "github.com/google/uuid"
    "time"
)

type SenderType string
const (
    SenderTypePerson   SenderType = "person"
    SenderTypeBusiness SenderType = "business"
)

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
    IdentificationExpiry time.Time
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

// GORM Hooks
func (s *Sender) BeforeCreate(tx *gorm.DB) error {
    if s.CustomerID == uuid.Nil {
        return errors.New("customer_id required")
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

