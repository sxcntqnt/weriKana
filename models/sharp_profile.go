// models/sharp_profile.go
package models

import (
	"fmt"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SharpProfile struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID         uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_customer_asset_class;not null"`
	AssetClass         string    `gorm:"size:20;index;uniqueIndex:idx_customer_asset_class;not null"` // "sharp", "sports", "stock", "forex", "crypto"
	RealEV             float64   `gorm:"default:0.0"` // Real money EV (cents)
	FakeEV             float64   `gorm:"default:0.0"` // Fake money EV (cents)
	RealSharpeRatio    float64   `gorm:"default:0.0"` // Risk-adjusted return
	FakeSharpeRatio    float64   `gorm:"default:0.0"`
	RealHitRate        float64   `gorm:"default:0.0"` // % successful trades
	FakeHitRate        float64   `gorm:"default:0.0"`
	RealMaxDrawdown    int64     `gorm:"default:0"`   // Largest loss (cents)
	FakeMaxDrawdown    int64     `gorm:"default:0"`
	RealKellyFraction  float64   `gorm:"default:0.0"` // Optimal trade size
	FakeKellyFraction  float64   `gorm:"default:0.0"`
	RealTradeVolume    int64     `gorm:"default:0"`   // Total trades (cents)
	FakeTradeVolume    int64     `gorm:"default:0"`
	RiskScore          float64   `gorm:"default:0.0"` // 0-100, higher = riskier
	PreferredMarkets   JSONMap   `gorm:"type:jsonb"`  // e.g., {"leagues": ["EPL"], "tickers": ["AAPL"]}
	Customer           Customer  `gorm:"foreignKey:CustomerID"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

func (SharpProfile) TableName() string {
	return "sharp_profiles"
}

func (sp *SharpProfile) BeforeCreate(tx *gorm.DB) error {
	validAssetClasses := map[string]bool{
		"sharp":  true,
		"sports": true,
		"stock":  true,
		"forex":  true,
		"crypto": true,
	}
	if !validAssetClasses[sp.AssetClass] {
		return fmt.Errorf("invalid asset class: %s", sp.AssetClass)
	}
	return nil
}
