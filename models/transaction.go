// models/transaction.go
// Transaction model — Updated for Sharp merge: FK to SharpID (replaces CustomerID); UUID gen in BeforeCreate.
// Links to specific accounts under Sharp; Sender now via SharpID chain.

package models

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypeDeposit TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdraw"
	TransactionTypeTrade   TransactionType = "trade"
)

type Transaction struct {
	gorm.Model
	ID                 uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID            uuid.UUID  `gorm:"type:uuid;index;not null"` // FK to Sharp (replaces CustomerID)
	SharpAccountID     uuid.UUID  `gorm:"type:uuid;index"` // Optional: for Sharp-specific txns
	SportsAccountID    uuid.UUID  `gorm:"type:uuid;index"`
	StockAccountID     uuid.UUID  `gorm:"type:uuid;index"`
	ForexAccountID     uuid.UUID  `gorm:"type:uuid;index"`
	CryptoAccountID    uuid.UUID  `gorm:"type:uuid;index"`
	SenderID           uuid.UUID  `gorm:"type:uuid;index"`
	BookieAccountID    uuid.UUID  `gorm:"type:uuid"`
	Reference          string     `gorm:"size:50;uniqueIndex"`
	Type               TransactionType `gorm:"not null"`
	AmountCents        int64      `gorm:"type:bigint;not null"`
	IsReal             bool       `gorm:"not null"`
	Currency           string     `gorm:"size:3;default:KES"`
	Status             TransactionStatus `gorm:"size:20;default:pending"`
	Metadata           JSONMap    `gorm:"type:json"` // e.g., {"ev": 100, "market": "EPL"}
	ExternalID         string     `gorm:"size:100;index"`
	ExpiresAt          sql.NullTime `gorm:"type:timestamp"`
	InvalidAt          sql.NullTime `gorm:"type:timestamp"`
	IdempotencyKey     string     `gorm:"uniqueIndex"` // Added from models.go
	Sharp              Sharp      `gorm:"foreignKey:SharpID"` // Back-ref to merged Sharp
	SharpAccount       SharpAccount `gorm:"foreignKey:SharpAccountID"`
	SportsAccount      SportsAccount `gorm:"foreignKey:SportsAccountID"`
	StockAccount       StockAccount `gorm:"foreignKey:StockAccountID"`
	ForexAccount       ForexAccount `gorm:"foreignKey:ForexAccountID"`
	CryptoAccount      CryptoAccount `gorm:"foreignKey:CryptoAccountID"`
	Sender             Sender     `gorm:"foreignKey:SenderID"`
}

// BeforeCreate GORM hook → gen UUID; validate exactly one account ID; ensure SharpID & positive amount.
func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.SharpID == uuid.Nil {
		return fmt.Errorf("sharp_id required")
	}
	count := 0
	if t.SharpAccountID != uuid.Nil {
		count++
	}
	if t.SportsAccountID != uuid.Nil {
		count++
	}
	if t.StockAccountID != uuid.Nil {
		count++
	}
	if t.ForexAccountID != uuid.Nil {
		count++
	}
	if t.CryptoAccountID != uuid.Nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("exactly one account ID must be set")
	}
	if t.AmountCents <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	return nil
}
