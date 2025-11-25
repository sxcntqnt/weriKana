// models/ledger.go
// Separate entities for Real & Fake balance ledgers — Track deltas with reason/ref.

package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type RealLedgerEntry struct {
	gorm.Model
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	SharpID      uuid.UUID  `gorm:"type:uuid;index"` // FK to Sharp
	AmountCents  int64      `gorm:"not null"` // Positive deposit, negative withdrawal
	Reason       string     `gorm:"size:255;not null"` // e.g., "deposit", "trade_win"
	Reference    string     `gorm:"size:100"` // Transaction ref
	Status       string     `gorm:"size:50;default:completed"` // completed, pending, failed
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Sharp        Sharp      `gorm:"foreignKey:SharpID"` // Back-ref
}

type FakeLedgerEntry struct {
	gorm.Model
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey"`
	SharpID      uuid.UUID  `gorm:"type:uuid;index"` // FK to Sharp
	AmountCents  int64      `gorm:"not null"` // Positive bonus, negative usage
	Reason       string     `gorm:"size:255;not null"` // e.g., "promo_bonus", "fake_trade_loss"
	Reference    string     `gorm:"size:100"` // Promo code or tx ref
	Status       string     `gorm:"size:50;default:completed"` // completed, pending, failed
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Sharp        Sharp      `gorm:"foreignKey:SharpID"` // Back-ref
}

// BeforeCreate for UUID gen
func (le *RealLedgerEntry) BeforeCreate(tx *gorm.DB) error {
	if le.ID == uuid.Nil {
		le.ID = uuid.New()
	}
	return nil
}

func (le *FakeLedgerEntry) BeforeCreate(tx *gorm.DB) error {
	if le.ID == uuid.Nil {
		le.ID = uuid.New()
	}
	return nil
}
