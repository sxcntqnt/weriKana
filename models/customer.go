// models/customer.go
package models

import (
	"regexp"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Kenyan phone regex: +2547xxxxxxxx or 07xxxxxxxx
var kenyanPhoneRegex = regexp.MustCompile(`^(\+254|0)7[0-9]{8}$`)

type Customer struct {
	gorm.Model
	ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name           string    `gorm:"size:255;not null" json:"name"`
	Email          string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Phone          string    `gorm:"size:20;uniqueIndex;not null" json:"phone"` // e.g. +254712345678
	PreferredMpesa string    `gorm:"size:20" json:"preferred_mpesa"`           // fallback payout number

	// One Customer → Many BookieAccounts (across different bookies)
	BookieAccounts []BookieAccount `gorm:"foreignKey:CustomerID" json:"-"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate GORM hook → validate phone & normalize
func (c *Customer) BeforeCreate(tx *gorm.DB) error {
	if c.Email == "" {
		return gorm.ErrInvalidData
	}

	if !kenyanPhoneRegex.MatchString(c.Phone) {
		return gorm.ErrInvalidData.Wrap("invalid Kenyan phone number")
	}

	// Normalize: always store as +254...
	if len(c.Phone) == 10 && c.Phone[0] == '0' {
		c.Phone = "+254" + c.Phone[1:]
	}

	// Optional: normalize PreferredMpesa same way
	if c.PreferredMpesa != "" && !kenyanPhoneRegex.MatchString(c.PreferredMpesa) {
		return gorm.ErrInvalidData.Wrap("invalid preferred MPESA number")
	}
	if c.PreferredMpesa != "" && c.PreferredMpesa[0] == '0' {
		c.PreferredMpesa = "+254" + c.PreferredMpesa[1:]
	}

	return nil
}
