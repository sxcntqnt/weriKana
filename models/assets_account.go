// models/assets_account.go
// Account models for assets (Sports, Stock, Forex, Crypto) — Updated for Sharp merge.
// Added EWMA metrics (LogReturn, Volatility, SharpeRatio, LastUpdatedPerf) to all for uniform tracking.

package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type SportsAccount struct {
	gorm.Model
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID          uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie;not null"` // FK to Sharp (replaces CustomerID)
	BookieID         uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie;not null"`
	ManagerID        uuid.UUID  `gorm:"type:uuid;index;not null"`
	MpesaNumber      string     `gorm:"size:20"`
	RealBalanceCents int64      `gorm:"default:0"`
	FakeBalanceCents int64      `gorm:"default:0"`
	BonusCents       int64      `gorm:"default:0"`
	Currency         string     `gorm:"size:3;default:KES"`
	IsActive         bool       `gorm:"default:true"`
	EncryptedKey     string     `gorm:"type:text"`
	// EWMA Performance Metrics (tracked across all assets)
	EWMALogReturn    float64    `gorm:"default:0"` // EWMA of log returns
	EWMAVolatility   float64    `gorm:"default:0"` // EWMA volatility (annualized)
	SharpeRatio      float64    `gorm:"default:0"` // Precomputed Sharpe ratio
	LastUpdatedPerf  time.Time  `gorm:"default:now()"` // Last perf calc timestamp
	BetHistory       JSONMap    `gorm:"type:json"` // JSONMap impl above
	Bookie           Bookie     `gorm:"foreignKey:BookieID"`
	Sharp            Sharp      `gorm:"foreignKey:SharpID"` // Back-ref to merged Sharp
	Transactions     []Transaction `gorm:"foreignKey:SportsAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate generates UUID if nil (cross-DB compatible).
func (sa *SportsAccount) BeforeCreate(tx *gorm.DB) error {
	if sa.ID == uuid.Nil {
		sa.ID = uuid.New()
	}
	return nil
}

func (SportsAccount) TableName() string {
	return "sports_accounts"
}

func CreateSportsAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_sharp_bookie
		ON sports_accounts (sharp_id, bookie_id)
		WHERE deleted_at IS NULL;
	`).Error
}

type StockAccount struct {
	gorm.Model
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID          uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie;not null"` // FK to Sharp (replaces CustomerID)
	BookieID         uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie;not null"`
	ManagerID        uuid.UUID  `gorm:"type:uuid;index;not null"` // Links to StockManager
	MpesaNumber      string     `gorm:"size:20"`
	RealBalanceCents int64      `gorm:"default:0"`
	FakeBalanceCents int64      `gorm:"default:0"`
	Currency         string     `gorm:"size:3;default:KES"`
	IsActive         bool       `gorm:"default:true"`
	Broker           string     `gorm:"default:DhowCSD"`
	Portfolio        JSONMap    `gorm:"type:json"` // e.g., {"holdings": [{"ticker": "AAPL", "shares": 10}]}
	EncryptedKey     string     `gorm:"type:text"`
	// EWMA Performance Metrics (added for uniformity)
	EWMALogReturn    float64    `gorm:"default:0"` // EWMA of log returns
	EWMAVolatility   float64    `gorm:"default:0"` // EWMA volatility (annualized)
	SharpeRatio      float64    `gorm:"default:0"` // Precomputed Sharpe ratio
	LastUpdatedPerf  time.Time  `gorm:"default:now()"` // Last perf calc timestamp
	// Relationships
	Bookie           Bookie     `gorm:"foreignKey:BookieID"`
	Sharp            Sharp      `gorm:"foreignKey:SharpID"` // Back-ref
	Transactions     []Transaction `gorm:"foreignKey:StockAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate generates UUID if nil.
func (sa *StockAccount) BeforeCreate(tx *gorm.DB) error {
	if sa.ID == uuid.Nil {
		sa.ID = uuid.New()
	}
	return nil
}

func (StockAccount) TableName() string {
	return "stock_accounts"
}

func CreateStockAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_sharp_bookie
		ON stock_accounts (sharp_id, bookie_id)
		WHERE deleted_at IS NULL;
	`).Error
}

type ForexAccount struct {
	gorm.Model
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID          uuid.UUID       `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie;not null"` // FK to Sharp (replaces CustomerID)
	BookieID         uuid.UUID       `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie;not null"`
	ManagerID        uuid.UUID       `gorm:"type:uuid;index;not null"` // Links to ForexManager
	MpesaNumber      string          `gorm:"size:20"`
	RealBalanceCents int64           `gorm:"default:0"`
	FakeBalanceCents int64           `gorm:"default:0"`
	Currency         string          `gorm:"size:3;default:KES"`
	IsActive         bool            `gorm:"default:true"`
	Leverage         decimal.Decimal `gorm:"type:decimal(5,2);default:1.00"`
	OpenPositions    JSONMap         `gorm:"type:json"` // e.g., {"trades": [{"pair": "USD/KES", "size": 1000}]}
	EncryptedKey     string          `gorm:"type:text"`
	// EWMA Performance Metrics (added for uniformity)
	EWMALogReturn    float64         `gorm:"default:0"` // EWMA of log returns
	EWMAVolatility   float64         `gorm:"default:0"` // EWMA volatility (annualized)
	SharpeRatio      float64         `gorm:"default:0"` // Precomputed Sharpe ratio
	LastUpdatedPerf  time.Time       `gorm:"default:now()"` // Last perf calc timestamp
	// Relationships
	Bookie           Bookie          `gorm:"foreignKey:BookieID"`
	Sharp            Sharp           `gorm:"foreignKey:SharpID"` // Back-ref
	Transactions     []Transaction   `gorm:"foreignKey:ForexAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate generates UUID if nil.
func (fa *ForexAccount) BeforeCreate(tx *gorm.DB) error {
	if fa.ID == uuid.Nil {
		fa.ID = uuid.New()
	}
	return nil
}

func (ForexAccount) TableName() string {
	return "forex_accounts"
}

func CreateForexAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_sharp_bookie
		ON forex_accounts (sharp_id, bookie_id)
		WHERE deleted_at IS NULL;
	`).Error
}

type CryptoAccount struct {
	gorm.Model
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	SharpID          uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie;not null"` // FK to Sharp (replaces CustomerID)
	BookieID         uuid.UUID  `gorm:"type:uuid;index;uniqueIndex:idx_sharp_bookie;not null"`
	ManagerID        uuid.UUID  `gorm:"type:uuid;index;not null"` // Links to CryptoManager
	MpesaNumber      string     `gorm:"size:20"`
	RealBalanceCents int64      `gorm:"default:0"`
	FakeBalanceCents int64      `gorm:"default:0"`
	FiatCentsKE      int64      `gorm:"default:0"`
	Currency         string     `gorm:"size:3;default:KES"`
	IsActive         bool       `gorm:"default:true"`
	EncryptedSeed    string     `gorm:"type:text"` // AES-GCM encrypted wallet seed
	Addresses        JSONMap    `gorm:"type:json"` // e.g., {"coins": {"BTC": "bc1...", "ETH": "0x..."}}
	EncryptedKey     string     `gorm:"type:text"`
	// EWMA Performance Metrics (added for uniformity)
	EWMALogReturn    float64    `gorm:"default:0"` // EWMA of log returns
	EWMAVolatility   float64    `gorm:"default:0"` // EWMA volatility (annualized)
	SharpeRatio      float64    `gorm:"default:0"` // Precomputed Sharpe ratio
	LastUpdatedPerf  time.Time  `gorm:"default:now()"` // Last perf calc timestamp
	// Relationships
	Bookie           Bookie     `gorm:"foreignKey:BookieID"`
	Sharp            Sharp      `gorm:"foreignKey:SharpID"` // Back-ref
	Transactions     []Transaction `gorm:"foreignKey:CryptoAccountID"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// BeforeCreate generates UUID if nil.
func (ca *CryptoAccount) BeforeCreate(tx *gorm.DB) error {
	if ca.ID == uuid.Nil {
		ca.ID = uuid.New()
	}
	return nil
}

func (CryptoAccount) TableName() string {
	return "crypto_accounts"
}

func CreateCryptoAccountConstraints(db *gorm.DB) error {
	return db.Exec(`
		CREATE UNIQUE INDEX IF NOT EXISTS idx_sharp_bookie
		ON crypto_accounts (sharp_id, bookie_id)
		WHERE deleted_at IS NULL;
	`).Error
}
