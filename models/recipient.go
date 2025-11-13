// models/recipient.go
package models

import (
    "errors"
    "gorm.io/gorm"
    "github.com/google/uuid"
    "time"
)

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
    IdentificationExpiry time.Time
    CountryCode          string `gorm:"size:2;default:'KE'"`
    Street               string `gorm:"size:255"`
    PostalCode           string `gorm:"size:20"`
    City                 string `gorm:"size:100"`

    // Bank Account Details
    BankAccountType      BankAccountType
    BankAccountNumber    string    `gorm:"size:100"`
    BankName             string    `gorm:"size:255"`

    // Encrypted Fields
    EmailEnc       string `gorm:"column:email_enc;size:500"`
    PhoneNumberEnc string `gorm:"column:phone_enc;size:500"`

    // Plaintext
    ExternalID string `gorm:"size:100;uniqueIndex"`

    // Relationships
    Customer Customer `gorm:"foreignKey:CustomerID;constraint:OnDelete:RESTRICT"`
}

// Setters with Encryption & Validation
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

