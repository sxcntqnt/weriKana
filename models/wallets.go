// models/wallets.go
// Wallet models for assets (Sports, Stock, Forex, Crypto) — Updated for Sharp merge.
// Added FakeBalanceCents to all for bonus tracking; non-negative validation in BeforeSave.
// FK: SharpID; back-ref to Sharp; UUID gen in BeforeCreate.

package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type SportsWallet struct {
	gorm.Model
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID          uuid.UUID  `gorm:"type:uuid;uniqueIndex:idx_sharp_bookie;not null"` // FK to Sharp (replaces CustomerID)
	BookieID         uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie"`
	RealCents        int64      `gorm:"default:0"`
	FakeBalanceCents int64      `gorm:"default:0;check:fake_cents >= 0"` // NEW: Fake/bonus cents (non-negative)
	BonusCents       int64      `gorm:"default:0"` // Subset of FakeCents (promos)
	EncryptedKey     string     `gorm:"type:text"`
	Sharp            Sharp      `gorm:"foreignKey:SharpID"` // Back-ref
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeSave ensures FakeBalanceCents non-negative (bonus can't be overdrawn).
func (sw *SportsWallet) BeforeSave(tx *gorm.DB) error {
	if sw.FakeBalanceCents < 0 {
		return fmt.Errorf("fake_balance_cents cannot be negative")
	}
	return nil
}

// BeforeCreate generates UUID if nil (cross-DB compatible).
func (sw *SportsWallet) BeforeCreate(tx *gorm.DB) error {
	if sw.ID == uuid.Nil {
		sw.ID = uuid.New()
	}
	return nil
}

type StockWallet struct {
	gorm.Model
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID          uuid.UUID  `gorm:"type:uuid;uniqueIndex:idx_sharp_manager;not null"` // FK to Sharp (replaces CustomerID)
	ManagerID        uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_manager"`
	RealCents        int64      `gorm:"default:0"`
	FakeBalanceCents int64      `gorm:"default:0;check:fake_cents >= 0"` // NEW: Fake/bonus cents (non-negative)
	Broker           string     `gorm:"default:DhowCSD"`
	Sharp            Sharp      `gorm:"foreignKey:SharpID"` // Back-ref
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeSave ensures FakeBalanceCents non-negative.
func (sw *StockWallet) BeforeSave(tx *gorm.DB) error {
	if sw.FakeBalanceCents < 0 {
		return fmt.Errorf("fake_balance_cents cannot be negative")
	}
	return nil
}

// BeforeCreate generates UUID if nil.
func (sw *StockWallet) BeforeCreate(tx *gorm.DB) error {
	if sw.ID == uuid.Nil {
		sw.ID = uuid.New()
	}
	return nil
}

type ForexWallet struct {
	gorm.Model
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID          uuid.UUID       `gorm:"type:uuid;uniqueIndex:idx_sharp_manager;not null"` // FK to Sharp (replaces CustomerID)
	ManagerID        uuid.UUID       `gorm:"type:uuid;index;uniqueIndex:idx_sharp_manager"`
	RealCents        int64           `gorm:"default:0"`
	FakeBalanceCents int64           `gorm:"default:0;check:fake_cents >= 0"` // NEW: Fake/bonus cents (non-negative)
	Leverage         decimal.Decimal `gorm:"type:decimal(5,2);default:1.00"`
	Sharp            Sharp           `gorm:"foreignKey:SharpID"` // Back-ref
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeSave ensures FakeBalanceCents non-negative.
func (fw *ForexWallet) BeforeSave(tx *gorm.DB) error {
	if fw.FakeBalanceCents < 0 {
		return fmt.Errorf("fake_balance_cents cannot be negative")
	}
	return nil
}

// BeforeCreate generates UUID if nil.
func (fw *ForexWallet) BeforeCreate(tx *gorm.DB) error {
	if fw.ID == uuid.Nil {
		fw.ID = uuid.New()
	}
	return nil
}

type CryptoWallet struct {
	gorm.Model
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID          uuid.UUID  `gorm:"type:uuid;uniqueIndex:idx_sharp_manager;not null"` // FK to Sharp (replaces CustomerID)
	ManagerID        uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_manager"`
	RealCents        int64      `gorm:"default:0"`
	FakeBalanceCents int64      `gorm:"default:0;check:fake_cents >= 0"` // NEW: Fake/bonus cents (non-negative)
	FiatCentsKE      int64      `gorm:"default:0"`
	EncryptedSeed    string     `gorm:"type:text"`
	Sharp            Sharp      `gorm:"foreignKey:SharpID"` // Back-ref
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeSave ensures FakeBalanceCents non-negative.
func (cw *CryptoWallet) BeforeSave(tx *gorm.DB) error {
	if cw.FakeBalanceCents < 0 {
		return fmt.Errorf("fake_balance_cents cannot be negative")
	}
	return nil
}

// BeforeCreate generates UUID if nil.
func (cw *CryptoWallet) BeforeCreate(tx *gorm.DB) error {
	if cw.ID == uuid.Nil {
		cw.ID = uuid.New()
	}
	return nil
}
