// models/sharp_account.go
// SharpAccount model — Updated for Sharp merge: FK to SharpID (replaces CustomerID); UUID gen in BeforeCreate.
// Constraints use SharpID + SharpProfileID.

package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SharpAccount struct {
	gorm.Model
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID         uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_profile;not null"` // FK to Sharp (replaces CustomerID)
	SharpProfileID  uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_profile;not null"` // Link to SharpProfile (AssetClass = "sharp")
	MpesaNumber     string     `gorm:"size:20"`
	RealBalanceCents int64     `gorm:"default:0"`
	FakeBalanceCents int64     `gorm:"default:0"`
	Currency        string     `gorm:"size:3;default:KES"`
	IsActive        bool       `gorm:"default:true"`
	EncryptedKey    string     `gorm:"column:encrypted_key;type:text"`
	Sharp           Sharp      `gorm:"foreignKey:SharpID"`
	SharpProfile    SharpProfile `gorm:"foreignKey:SharpProfileID"`
	Transactions    []Transaction `gorm:"foreignKey:SharpAccountID"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate generates UUID if nil (cross-DB compatible).
func (sa *SharpAccount) BeforeCreate(tx *gorm.DB) error {
	if sa.ID == uuid.Nil {
		sa.ID = uuid.New()
	}
	return nil
}

func (SharpAccount) TableName() string {
	return "sharp_accounts"
}

func CreateSharpAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_sharp_profile
		ON sharp_accounts (sharp_id, sharp_profile_id)
		WHERE deleted_at IS NULL;
	`).Error
}
