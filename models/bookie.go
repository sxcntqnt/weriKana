package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Bookie struct {
	ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name      string         `gorm:"size:255;not null"` // e.g., "Bet365", "SportPesa"
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Bookie) TableName() string {
	return "bookies"
}
