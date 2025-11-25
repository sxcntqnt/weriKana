// models/sharp_profile.go
// SharpProfile model — Updated for Sharp merge: FK to SharpID; UUID gen in BeforeCreate.
// UniqueIndex uses SharpID + AssetClass; removed "sharp" from valid classes (not a tradable asset).

package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SharpProfile struct {
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID            uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_asset_class;not null"` // FK to Sharp (replaces CustomerID)
	AssetClass         string     `gorm:"size:20;index;uniqueIndex:idx_sharp_asset_class;not null"` // "sports", "stock", "forex", "crypto"
	RealEV             float64    `gorm:"default:0.0"` // Real money EV (cents)
	FakeEV             float64    `gorm:"default:0.0"` // Fake money EV (cents)
	RealSharpeRatio    float64    `gorm:"default:0.0"` // Risk-adjusted return
	FakeSharpeRatio    float64    `gorm:"default:0.0"`
	RealHitRate        float64    `gorm:"default:0.0"` // % successful trades
	FakeHitRate        float64    `gorm:"default:0.0"`
	RealMaxDrawdown    int64      `gorm:"default:0"` // Largest loss (cents)
	FakeMaxDrawdown    int64      `gorm:"default:0"`
	RealKellyFraction  float64    `gorm:"default:0.0"` // Optimal trade size
	FakeKellyFraction  float64    `gorm:"default:0.0"`
	RealTradeVolume    int64      `gorm:"default:0"` // Total trades (cents)
	FakeTradeVolume    int64      `gorm:"default:0"`
	RiskScore          float64    `gorm:"default:0.0"` // 0-100, higher = riskier
	PreferredMarkets   JSONMap    `gorm:"type:json"` // e.g., {"leagues": ["EPL"], "tickers": ["AAPL"]}
	Sharp              Sharp      `gorm:"foreignKey:SharpID"` // Back-ref to merged Sharp
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate validates AssetClass (only tradable verticals: sports/stock/forex/crypto) & generates UUID.
func (sp *SharpProfile) BeforeCreate(tx *gorm.DB) error {
	if sp.ID == uuid.Nil {
		sp.ID = uuid.New()
	}
	validAssetClasses := map[string]bool{
		"sports": true,
		"stock":  true,
		"forex":  true,
		"crypto": true,
		// "sharp" removed — not a tradable asset class
	}
	if !validAssetClasses[sp.AssetClass] {
		return fmt.Errorf("invalid asset class: %s (must be sports, stock, forex, or crypto)", sp.AssetClass)
	}
	return nil
}

func (SharpProfile) TableName() string {
	return "sharp_profiles"
}
