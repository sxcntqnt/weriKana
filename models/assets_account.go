package models

import (
	"time"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type SportsAccount struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID       uuid.UUID      `gorm:"type:uuid;index;uniqueIndex:idx_customer_bookie;not null"`
	BookieID         uuid.UUID      `gorm:"type:uuid;index;uniqueIndex:idx_customer_bookie;not null"`
	ManagerID        uuid.UUID      `gorm:"type:uuid;index;not null"` // Links to SportsManager
	MpesaNumber      string         `gorm:"size:20"`
	RealBalanceCents int64          `gorm:"default:0"`
	FakeBalanceCents int64          `gorm:"default:0"`
	BonusCents       int64          `gorm:"default:0"`
	Currency         string         `gorm:"size:3;default:'KES'"`
	IsActive         bool           `gorm:"default:true"`
	EncryptedKey     string         `gorm:"type:text"` // AES-GCM encrypted session key
	BetHistory       JSONMap        `gorm:"type:jsonb"` // e.g., {"bets": [{"match": "EPL", "amount": 1000}]}
	// Relationships
	Bookie           Bookie         `gorm:"foreignKey:BookieID"`
	Customer         Customer       `gorm:"foreignKey:CustomerID"`
	Transactions     []Transaction  `gorm:"foreignKey:SportsAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (SportsAccount) TableName() string {
	return "sports_accounts"
}

func CreateSportsAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_bookie
		ON sports_accounts (customer_id, bookie_id)
		WHERE deleted_at IS NULL;
	`).Error
}

type StockAccount struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID       uuid.UUID      `gorm:"type:uuid;index;uniqueIndex:idx_customer_bookie;not null"`
	BookieID         uuid.UUID      `gorm:"type:uuid;index;uniqueIndex:idx_customer_bookie;not null"`
	ManagerID        uuid.UUID      `gorm:"type:uuid;index;not null"` // Links to StockManager
	MpesaNumber      string         `gorm:"size:20"`
	RealBalanceCents int64          `gorm:"default:0"`
	FakeBalanceCents int64          `gorm:"default:0"`
	Currency         string         `gorm:"size:3;default:'KES'"`
	IsActive         bool           `gorm:"default:true"`
	Broker           string         `gorm:"default:'DhowCSD'"`
	Portfolio        JSONMap        `gorm:"type:jsonb"` // e.g., {"holdings": [{"ticker": "AAPL", "shares": 10}]}
	EncryptedKey     string         `gorm:"type:text"`
	// Relationships
	Bookie           Bookie         `gorm:"foreignKey:BookieID"`
	Customer         Customer       `gorm:"foreignKey:CustomerID"`
	Transactions     []Transaction  `gorm:"foreignKey:StockAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (StockAccount) TableName() string {
	return "stock_accounts"
}

func CreateStockAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_bookie
		ON stock_accounts (customer_id, bookie_id)
		WHERE deleted_at IS NULL;
	`).Error
}

type ForexAccount struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID       uuid.UUID       `gorm:"type:uuid;index;uniqueIndex:idx_customer_bookie;not null"`
	BookieID         uuid.UUID       `gorm:"type:uuid;index;uniqueIndex:idx_customer_bookie;not null"`
	ManagerID        uuid.UUID       `gorm:"type:uuid;index;not null"` // Links to ForexManager
	MpesaNumber      string          `gorm:"size:20"`
	RealBalanceCents int64           `gorm:"default:0"`
	FakeBalanceCents int64           `gorm:"default:0"`
	Currency         string          `gorm:"size:3;default:'KES'"`
	IsActive         bool            `gorm:"default:true"`
	Leverage         decimal.Decimal `gorm:"type:decimal(5,2);default:1.00"`
	OpenPositions    JSONMap         `gorm:"type:jsonb"` // e.g., {"trades": [{"pair": "USD/KES", "size": 1000}]}
	EncryptedKey     string          `gorm:"type:text"`
	// Relationships
	Bookie           Bookie         `gorm:"foreignKey:BookieID"`
	Customer         Customer       `gorm:"foreignKey:CustomerID"`
	Transactions     []Transaction  `gorm:"foreignKey:ForexAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt  `gorm:"index"`
}

func (ForexAccount) TableName() string {
	return "forex_accounts"
}

func CreateForexAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_bookie
		ON forex_accounts (customer_id, bookie_id)
		WHERE deleted_at IS NULL;
	`).Error
}

type CryptoAccount struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	CustomerID       uuid.UUID      `gorm:"type:uuid;index;uniqueIndex:idx_customer_bookie;not null"`
	BookieID         uuid.UUID      `gorm:"type:uuid;index;uniqueIndex:idx_customer_bookie;not null"`
	ManagerID        uuid.UUID      `gorm:"type:uuid;index;not null"` // Links to CryptoManager
	MpesaNumber      string         `gorm:"size:20"`
	RealBalanceCents int64          `gorm:"default:0"`
	FakeBalanceCents int64          `gorm:"default:0"`
	FiatCentsKE      int64          `gorm:"default:0"`
	Currency         string         `gorm:"size:3;default:'KES'"`
	IsActive         bool           `gorm:"default:true"`
	EncryptedSeed    string         `gorm:"type:text"` // AES-GCM encrypted wallet seed
	Addresses        JSONMap        `gorm:"type:jsonb"` // e.g., {"coins": {"BTC": "bc1...", "ETH": "0x..."}}
	EncryptedKey     string         `gorm:"type:text"`
	// Relationships
	Bookie           Bookie         `gorm:"foreignKey:BookieID"`
	Customer         Customer       `gorm:"foreignKey:CustomerID"`
	Transactions     []Transaction  `gorm:"foreignKey:CryptoAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (CryptoAccount) TableName() string {
	return "crypto_accounts"
}

func CreateCryptoAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_customer_bookie
		ON crypto_accounts (customer_id, bookie_id)
		WHERE deleted_at IS NULL;
	`).Error
}
