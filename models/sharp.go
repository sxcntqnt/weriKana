// models/sharp.go
package models

import (
        "fmt"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Sharp struct {
	gorm.Model
	ID             uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SportsBankID   uuid.UUID      `gorm:"type:uuid;index"`
	StockBankID    uuid.UUID      `gorm:"type:uuid;index"`
	ForexBankID    uuid.UUID      `gorm:"type:uuid;index"`
	CryptoBankID   uuid.UUID      `gorm:"type:uuid;index"`
	Name           string         `gorm:"size:255;not null"`
	AccountNumber  string         `gorm:"size:50;not null;index"`
	MinTradeCents  int64          `gorm:"default:100"`      // 1 KES = 100 cents
	MaxTradeCents  int64          `gorm:"default:10000000"` // 100,000 KES
	CurrentBalance int64          // Total balance (real + fake)
	SportsBank     SportsBank     `gorm:"foreignKey:SportsBankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	StockBank      StockBank      `gorm:"foreignKey:StockBankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	ForexBank      ForexBank      `gorm:"foreignKey:ForexBankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	CryptoBank     CryptoBank     `gorm:"foreignKey:CryptoBankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	SharpAccounts  []SharpAccount `gorm:"foreignKey:SharpID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	SportsAccounts []SportsAccount `gorm:"foreignKey:SharpID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	StockAccounts  []StockAccount  `gorm:"foreignKey:SharpID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	ForexAccounts  []ForexAccount  `gorm:"foreignKey:SharpID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CryptoAccounts []CryptoAccount `gorm:"foreignKey:SharpID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (s *Sharp) BeforeCreate(tx *gorm.DB) error {
	count := 0
	if s.SportsBankID != uuid.Nil {
		count++
	}
	if s.StockBankID != uuid.Nil {
		count++
	}
	if s.ForexBankID != uuid.Nil {
		count++
	}
	if s.CryptoBankID != uuid.Nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("exactly one bank ID must be set")
	}
	return nil
}
