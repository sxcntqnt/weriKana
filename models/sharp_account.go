// models/sharp_account.go
package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SharpAccount struct {
	gorm.Model
	ID               uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	SharpID          uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_sharp_customer;not null"`
	CustomerID       uuid.UUID `gorm:"type:uuid;index;uniqueIndex:idx_sharp_customer;not null"`
	SharpProfileID   uuid.UUID `gorm:"type:uuid;index;not null"` // Link to SharpProfile (AssetClass = "sharp")
	MpesaNumber      string    `gorm:"size:20"`
	RealBalanceCents int64     `gorm:"default:0"`
	FakeBalanceCents int64     `gorm:"default:0"`
	Currency         string    `gorm:"size:3;default:'KES'"`
	IsActive         bool      `gorm:"default:true"`
	EncryptedKey     string    `gorm:"column:encrypted_key;type:text"`
	Sharp            Sharp         `gorm:"foreignKey:SharpID"`
	Customer         Customer      `gorm:"foreignKey:CustomerID"`
	SharpProfile     SharpProfile  `gorm:"foreignKey:SharpProfileID"`
	Transactions     []Transaction `gorm:"foreignKey:SharpAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (SharpAccount) TableName() string {
	return "sharp_accounts"
}

func CreateSharpAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_sharp_customer
		ON sharp_accounts (sharp_id, customer_id)
		WHERE deleted_at IS NULL;
	`).Error
}
