package models

import (
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
)
type Bookie struct {
    ID        uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Name      string         `gorm:"size:255;not null"`
    MpesaNumber    string
    MinDeposit     int64 // KES in cents (or smallest unit you use)
    MaxDeposit     int64
    RecentLogRet   float64 // EWMA of log returns
    RecentVol      float64 // EWMA volatility
    CurrentBalance int64
    CreatedAt time.Time
    UpdatedAt time.Time
    DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (Bookie) TableName() string {
	return "bookies"
}
