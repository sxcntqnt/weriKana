// models/bookie_account.go
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)
package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type BookieAccount struct {
	gorm.Model
	ID              uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	BookieID        uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_bookie_customer;not null"`
	CustomerID      uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_bookie_customer;not null"`
        MpesaNumber     string    `gorm:"size:20"`
	// Dual Balances (in cents)
	RealBalanceCents int64  `gorm:"default:0"`
	FakeBalanceCents int64  `gorm:"default:0"`
	Currency         string `gorm:"size:3;default:'KES'"`
	IsActive         bool   `gorm:"default:true"`

	// Relationships
	Bookie       Bookie       `gorm:"foreignKey:BookieID"`
	Customer     Customer     `gorm:"foreignKey:CustomerID"`
	Transactions []Transaction `gorm:"foreignKey:BookieAccountID"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
        EncryptedKey string `gorm:"column:encrypted_key;type:text"` // AES-GCM encrypted
}

// Enforce 1:1 via unique composite index
func (BookieAccount) TableName() string {
	return "bookie_accounts"
}

// Migration hook (call from initdb.go)
func CreateBookieAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_bookie_customer 
		ON bookie_accounts (bookie_id, customer_id) 
		WHERE deleted_at IS NULL;
	`).Error
}
// Helper: Get balance based on mode
func (ba *BookieAccount) Balance(isReal bool) int64 {
	if isReal {
		return ba.RealBalanceCents
	}
	return ba.FakeBalanceCents
}
// DecryptKey returns decrypted key (e.g., session cookie, API token)
func (ba *BookieAccount) DecryptKey() (string, error) {
    return decrypt(ba.EncryptedKey)
