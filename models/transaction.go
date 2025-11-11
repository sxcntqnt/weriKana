// models/transaction.go
package models

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- Transaction Types ---
type TransactionType string
const (
	TransactionTypeDeposit   TransactionType = "deposit"
	TransactionTypeWithdraw  TransactionType = "withdraw"
	TransactionTypeTransfer  TransactionType = "transfer"
	TransactionTypeFee       TransactionType = "fee"
)

// --- Status ---
type TransactionStatus string
const (
	StatusPending   TransactionStatus = "pending"
	StatusSuccess   TransactionStatus = "success"
	StatusFailed    TransactionStatus = "failed"
	StatusReversed  TransactionStatus = "reversed"
)

// --- Transaction ---
type Transaction struct {
	gorm.Model
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	BookieAccountID  uuid.UUID `gorm:"type:uuid;index;not null"`
	CustomerID       uuid.UUID `gorm:"type:uuid;index;not null"`
	SenderID         uuid.UUID `gorm:"type:uuid;index"`
	Reference        string    `gorm:"size:50;uniqueIndex"` // MPESA Receipt / OTP Ref
	Type             TransactionType
	AmountCents      int64     `gorm:"type:bigint;not null"` // Always in cents
        IsReal           bool      `gorm:"default:false"`
	Currency         string    `gorm:"size:3;default:'KES'"`
	Status           TransactionStatus `gorm:"size:20;default:'pending'"`
	Metadata         JSONMap   `gorm:"type:jsonb"` // Flexible: allocations, handshake, OTP
	ExternalID       string    `gorm:"size:100;index"`
	ExpiresAt        sql.NullTime
	InvalidAt        sql.NullTime

	// Relationships
	BookieAccount BookieAccount `gorm:"foreignKey:BookieAccountID"`
	Customer      Customer      `gorm:"foreignKey:CustomerID"`
	Sender        Sender        `gorm:"foreignKey:SenderID"`
}

// JSONMap for flexible metadata
type JSONMap map[string]any

func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal JSONMap")
	}
	return json.Unmarshal(bytes, j)
}

func (j JSONMap) Value() (interface{}, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

// Human-readable reference
func (t *Transaction) GetReferenceForHumans() string {
	return fmt.Sprintf("TXN-%s", t.Reference[:8])
}
