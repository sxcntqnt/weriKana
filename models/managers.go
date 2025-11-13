// models/managers.go
package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SportsManager struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Brand           string    `gorm:"size:255;default:'SharpsBet'"`
	SupportedLeagues JSONMap   `gorm:"type:jsonb;default:'{\"leagues\":[]}'"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

type StockManager struct {
	ID               uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Brand            string    `gorm:"size:255;default:'SharpsTrade'"`
	SupportedMarkets JSONMap   `gorm:"type:jsonb;default:'{\"tickers\":[]}'"`
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

type ForexManager struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Brand          string    `gorm:"size:255;default:'SharpsFX'"`
	SupportedPairs JSONMap   `gorm:"type:jsonb;default:'{\"pairs\":[]}'"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

type CryptoManager struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Brand           string    `gorm:"size:255;default:'SharpsCrypto'"`
	SupportedCoins  JSONMap   `gorm:"type:jsonb;default:'{\"coins\":[]}'"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}
