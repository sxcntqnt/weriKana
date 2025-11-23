// models/transaction_log.go
package models

import (
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// TransactionLog is an immutable audit trail of every balance change
type TransactionLog struct {
    ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    CustomerID  uuid.UUID      `gorm:"type:uuid;index;not null"`
    AccountType string         `gorm:"size:20;index;not null"` // "sports", "stock", "forex", "crypto", "sharp"
    AccountID   uuid.UUID      `gorm:"type:uuid;index;not null"`

    // Balance deltas (positive = credit, negative = debit)
    RealCents  int64 `gorm:"default:0"`
    FakeCents  int64 `gorm:"default:0"`
    BonusCents int64 `gorm:"default:0"`

    // Optional: track fiat in crypto wallet separately
    FiatCentsKE int64 `gorm:"default:0"`

    Reference string `gorm:"size:100;index"` // e.g. MPESA receipt, trade ID, withdrawal ID
    Reason    string `gorm:"size:255"`       // human description: "deposit", "fake_topup", "bet_win", "trade_profit", etc.
    Status    string `gorm:"size:20;default:'COMPLETED'"`

    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

// Optional: make sure table name is clean
func (TransactionLog) TableName() string {
    return "transaction_logs"
}

// Optional: auto-set timestamp
func (tl *TransactionLog) BeforeCreate(tx *gorm.DB) error {
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
