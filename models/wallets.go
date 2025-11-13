package models

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SportsWallet struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID   uuid.UUID `gorm:"uniqueIndex:idx_customer_manager"`
	BookieID     uuid.UUID
	RealCents    int64
	BonusCents   int64
	EncryptedKey string `gorm:"type:text"`
}

type StockWallet struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID uuid.UUID `gorm:"uniqueIndex:idx_customer_manager"`
	ManagerID  uuid.UUID
	RealCents  int64
	Broker     string `gorm:"default:'DhowCSD'"`
}

type ForexWallet struct {
	ID         uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID uuid.UUID       `gorm:"uniqueIndex:idx_customer_manager"`
	ManagerID  uuid.UUID
	RealCents  int64
	Leverage   decimal.Decimal `gorm:"type:decimal(5,2)"`
}

type CryptoWallet struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID    uuid.UUID `gorm:"uniqueIndex:idx_customer_manager"`
	ManagerID     uuid.UUID
	FiatCentsKE   int64
	EncryptedSeed string `gorm:"type:text"`
}
