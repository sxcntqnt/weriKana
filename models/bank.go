// models/bank.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Bank represents the top-level financial institution that owns many Bookies.
type Bank struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code        string    `gorm:"size:10;uniqueIndex;not null"`          // e.g., "SURREYBC"
	SwiftCode   string    `gorm:"size:11;index"`                         // Optional BIC
	CountryCode string    `gorm:"size:2;index;default:'KE'"`             // ISO 3166-1 alpha-2
	Name        string    `gorm:"size:255;not null"`
	ShortName   string    `gorm:"size:50;index"`

	// One Bank → Many Bookies
	Bookies []Bookie `gorm:"foreignKey:BankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Bookie represents a betting agent / sub-account under a Bank.
// Each Bookie can serve many Customers via 1:1 BookieAccount.
type Bookie struct {
	gorm.Model
	ID            uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	BankID        uuid.UUID `gorm:"type:uuid;index;not null"`
	Name          string    `gorm:"size:255;not null"`
	AccountNumber string    `gorm:"size:50;not null;index"` // Bank-specific account

	// Risk / balance metrics (in cents)
	MinDeposit      int64   `gorm:"default:100"`     // 1 KES = 100 cents
	MaxDeposit      int64   `gorm:"default:10000000"`// 100,000 KES
	RecentLogRet    float64 // EWMA log return
	RecentVol       float64 // EWMA volatility
	CurrentBalance  int64   // Total balance across all customer accounts?

	// Relationships
	Bank           Bank             `gorm:"foreignKey:BankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	BookieAccounts []BookieAccount  `gorm:"foreignKey:BookieID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
