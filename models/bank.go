// models/bank.go
package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SportsBank struct {
	gorm.Model
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code        string         `gorm:"size:10;uniqueIndex;not null"` // e.g., "SPORTSBC"
	SwiftCode   string         `gorm:"size:11;index"`
	CountryCode string         `gorm:"size:2;index;default:'KE'"`
	Name        string         `gorm:"size:255;not null;default:'Sharps Sports Bank'"`
	ShortName   string         `gorm:"size:50;index;default:'Sharps Sports'"`
	Sharps      []Sharp        `gorm:"foreignKey:SportsBankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AssetNexuses []AssetNexus  `gorm:"foreignKey:SportsBankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// models/stock_bank.go
type StockBank struct {
	gorm.Model
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code        string         `gorm:"size:10;uniqueIndex;not null"` // e.g., "STOCKBC"
	SwiftCode   string         `gorm:"size:11;index"`
	CountryCode string         `gorm:"size:2;index;default:'KE'"`
	Name        string         `gorm:"size:255;not null;default:'Sharps Stock Bank'"`
	ShortName   string         `gorm:"size:50;index;default:'Sharps Stock'"`
	Sharps      []Sharp        `gorm:"foreignKey:StockBankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AssetNexuses []AssetNexus  `gorm:"foreignKey:StockBankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// models/forex_bank.go
type ForexBank struct {
	gorm.Model
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code        string         `gorm:"size:10;uniqueIndex;not null"` // e.g., "FOREXBC"
	SwiftCode   string         `gorm:"size:11;index"`
	CountryCode string         `gorm:"size:2;index;default:'KE'"`
	Name        string         `gorm:"size:255;not null;default:'Sharps Forex Bank'"`
	ShortName   string         `gorm:"size:50;index;default:'Sharps Forex'"`
	Sharps      []Sharp        `gorm:"foreignKey:ForexBankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AssetNexuses []AssetNexus  `gorm:"foreignKey:ForexBankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// models/crypto_bank.go
type CryptoBank struct {
	gorm.Model
	ID          uuid.UUID      `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Code        string         `gorm:"size:10;uniqueIndex;not null"` // e.g., "CRYPTOBC"
	SwiftCode   string         `gorm:"size:11;index"`
	CountryCode string         `gorm:"size:2;index;default:'KE'"`
	Name        string         `gorm:"size:255;not null;default:'Sharps Crypto Bank'"`
	ShortName   string         `gorm:"size:50;index;default:'Sharps Crypto'"`
	Sharps      []Sharp        `gorm:"foreignKey:CryptoBankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AssetNexuses []AssetNexus  `gorm:"foreignKey:CryptoBankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}
