package models

import (
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypeDeposit  TransactionType = "deposit"
	TransactionTypeWithdraw TransactionType = "withdraw"
	TransactionTypeTrade    TransactionType = "trade"
)

type TransactionStatus string

const (
	StatusPending TransactionStatus = "pending"
	StatusSuccess TransactionStatus = "success"
	StatusFailed  TransactionStatus = "failed"
)

type Transaction struct {
	gorm.Model
	ID              uuid.UUID         `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SharpAccountID  uuid.UUID         `gorm:"type:uuid;index"`
	SportsAccountID uuid.UUID         `gorm:"type:uuid;index"`
	StockAccountID  uuid.UUID         `gorm:"type:uuid;index"`
	ForexAccountID  uuid.UUID         `gorm:"type:uuid;index"`
	CryptoAccountID uuid.UUID         `gorm:"type:uuid;index"`
	CustomerID      uuid.UUID         `gorm:"type:uuid;index;not null"`
	SenderID        uuid.UUID         `gorm:"type:uuid;index"`
	Reference       string            `gorm:"size:50;uniqueIndex"`
	Type            TransactionType   `gorm:"not null"`
	AmountCents     int64             `gorm:"type:bigint;not null"`
	IsReal          bool              `gorm:"not null"`
	Currency        string            `gorm:"size:3;default:'KES'"`
	Status          TransactionStatus `gorm:"size:20;default:'pending'"`
	Metadata        JSONMap           `gorm:"type:jsonb"` // e.g., {"ev": 100, "market": "EPL"}
	ExternalID      string            `gorm:"size:100;index"`
	ExpiresAt       sql.NullTime      `gorm:"type:timestamp"`
	InvalidAt       sql.NullTime      `gorm:"type:timestamp"`
	IdempotencyKey  string            `gorm:"uniqueIndex"` // Added from models.go
	SharpAccount    SharpAccount      `gorm:"foreignKey:SharpAccountID"`
	SportsAccount   SportsAccount     `gorm:"foreignKey:SportsAccountID"`
	StockAccount    StockAccount      `gorm:"foreignKey:StockAccountID"`
	ForexAccount    ForexAccount      `gorm:"foreignKey:ForexAccountID"`
	CryptoAccount   CryptoAccount     `gorm:"foreignKey:CryptoAccountID"`
	Customer        Customer          `gorm:"foreignKey:CustomerID"`
	Sender          Sender            `gorm:"foreignKey:SenderID"`
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
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
