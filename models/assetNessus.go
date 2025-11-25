// models/asset_nexus.go
// AssetNexus model — Links Sharps to banks/managers/accounts for assets.
// Updated for Sharp merge: FK to SharpID (replaces CustomerID); UUID gen in BeforeCreate.

package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AssetNexus struct {
	gorm.Model
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SportsBankID     uuid.UUID  `gorm:"type:uuid;index"`
	StockBankID      uuid.UUID  `gorm:"type:uuid;index"`
	ForexBankID      uuid.UUID  `gorm:"type:uuid;index"`
	CryptoBankID     uuid.UUID  `gorm:"type:uuid;index"`
	SharpID          uuid.UUID  `gorm:"type:uuid;unique;not null"` // FK to Sharp (replaces CustomerID)
	Name             string     `gorm:"default:Customer Asset Nexus"`
	LaunchedAt       time.Time  `gorm:"autoCreateTime"`
	SportsManagerID  uuid.UUID  `gorm:"type:uuid"`
	StockManagerID   uuid.UUID  `gorm:"type:uuid"`
	ForexManagerID   uuid.UUID  `gorm:"type:uuid"`
	CryptoManagerID  uuid.UUID  `gorm:"type:uuid"`
	SportsManager    SportsManager `gorm:"foreignKey:SportsManagerID"`
	StockManager     StockManager `gorm:"foreignKey:StockManagerID"`
	ForexManager     ForexManager `gorm:"foreignKey:ForexManagerID"`
	CryptoManager    CryptoManager `gorm:"foreignKey:CryptoManagerID"`
	SportsAccounts   []SportsAccount `gorm:"foreignKey:ManagerID"`
	StockAccounts    []StockAccount `gorm:"foreignKey:ManagerID"`
	ForexAccounts    []ForexAccount `gorm:"foreignKey:ManagerID"`
	CryptoAccounts   []CryptoAccount `gorm:"foreignKey:ManagerID"`
	SportsBank       SportsBank `gorm:"foreignKey:SportsBankID"`
	StockBank        StockBank `gorm:"foreignKey:StockBankID"`
	ForexBank        ForexBank `gorm:"foreignKey:ForexBankID"`
	CryptoBank       CryptoBank `gorm:"foreignKey:CryptoBankID"`
	Sharp            Sharp     `gorm:"foreignKey:SharpID"` // Back-ref to merged Sharp
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate GORM hook → gen UUID; ensure only one bank ID; gen ManagerIDs if nil; validate SharpID.
func (an *AssetNexus) BeforeCreate(tx *gorm.DB) error {
	if an.ID == uuid.Nil {
		an.ID = uuid.New()
	}
	if an.SharpID == uuid.Nil {
		return fmt.Errorf("sharp_id required")
	}
	count := 0
	if an.SportsBankID != uuid.Nil {
		count++
	}
	if an.StockBankID != uuid.Nil {
		count++
	}
	if an.ForexBankID != uuid.Nil {
		count++
	}
	if an.CryptoBankID != uuid.Nil {
		count++
	}
	if count > 1 {
		return fmt.Errorf("only one bank ID can be set")
	}
	if an.SportsManagerID == uuid.Nil {
		an.SportsManagerID = uuid.New()
	}
	if an.StockManagerID == uuid.Nil {
		an.StockManagerID = uuid.New()
	}
	if an.ForexManagerID == uuid.Nil {
		an.ForexManagerID = uuid.New()
	}
	if an.CryptoManagerID == uuid.Nil {
		an.CryptoManagerID = uuid.New()
	}
	return nil
}
