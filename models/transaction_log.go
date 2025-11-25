// models/transaction_log.go
// TransactionLog model — Immutable audit trail for all balance changes.
// Updated for Sharp merge: FK to SharpID (replaces CustomerID); supports SharpAccount vertical.
// Logs deltas from Transaction model (real/fake/bonus/fiat across all accounts).

package models

import (
	"time"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionLog struct {
	gorm.Model
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID         uuid.UUID  `gorm:"type:uuid;index;not null"` // FK to Sharp (replaces CustomerID)
	AccountType     string     `gorm:"size:20;index;not null"` // "sharp", "sports", "stock", "forex", "crypto"
	AccountID       uuid.UUID  `gorm:"type:uuid;index;not null"` // ID of SharpAccount/SportsAccount/etc.
	// Balance deltas (positive = credit, negative = debit)
	RealCents       int64      `gorm:"default:0"` // From Transaction.AmountCents if IsReal=true
	FakeCents       int64      `gorm:"default:0"` // From Transaction.AmountCents if IsReal=false
	BonusCents      int64      `gorm:"default:0"` // Bonus/promos
	FiatCentsKE     int64      `gorm:"default:0"` // Crypto fiat (KES)
	Reference       string     `gorm:"size:100;index"` // Maps to Transaction.Reference
	Reason          string     `gorm:"size:255"` // From Transaction.Type + Metadata (e.g., "deposit", "trade_profit")
	Status          string     `gorm:"size:20;default:COMPLETED"` // From Transaction.Status
	// Optional: Link back to source Transaction
	TransactionID   uuid.UUID  `gorm:"type:uuid;index"` // FK to Transaction.ID (if 1:1 logging)
	Transaction     Transaction `gorm:"foreignKey:TransactionID"` // Back-ref (optional preload)
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate GORM hook → gen UUID; set defaults; validate SharpID & AccountType.
func (tl *TransactionLog) BeforeCreate(tx *gorm.DB) error {
	if tl.ID == uuid.Nil {
		tl.ID = uuid.New()
	}
	if tl.SharpID == uuid.Nil {
		return fmt.Errorf("sharp_id required")
	}
	validAccountTypes := map[string]bool{
		"sharp":   true,
		"sports":  true,
		"stock":   true,
		"forex":   true,
		"crypto":  true,
	}
	if !validAccountTypes[tl.AccountType] {
		return fmt.Errorf("invalid account type: %s", tl.AccountType)
	}
	if tl.CreatedAt.IsZero() {
		tl.CreatedAt = time.Now().UTC()
	}
	if tl.UpdatedAt.IsZero() {
		tl.UpdatedAt = time.Now().UTC()
	}
	if tl.Status == "" {
		tl.Status = "COMPLETED"
	}
	return nil
}

func (TransactionLog) TableName() string {
	return "transaction_logs"
}
