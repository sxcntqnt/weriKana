package models

import (
    "github.com/google/uuid"
    "gorm.io/gorm"
)

// Bank represents a financial institution that can support multiple bookies.
type Bank struct {
    gorm.Model
    ID          uuid.UUID  `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    Code        string     `gorm:"size:10;uniqueIndex"` // e.g., bank code like "ABSA"
    SwiftCode   string     `gorm:"size:11;index"`       // BIC/SWIFT for international transfers
    CountryCode string     `gorm:"size:2;index"`        // ISO 3166-1 alpha-2, e.g., "KE"
    Name        string     `gorm:"size:255;not null"`
    ShortName   string     `gorm:"size:50;index"`
    
    // Relationship: One bank supports many bookies (has-many)
    Bookies     []Bookie   `gorm:"foreignKey:BankID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}

// Bookie represents a recipient/agent (e.g., betting agent) tied to a bank account.
// This is an updated version integrating with Bank; adjust as needed for your existing Bookie.
type Bookie struct {
    gorm.Model
    ID             uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
    Name           string    `gorm:"size:255;not null"`
    BankID         uuid.UUID `gorm:"type:uuid;index"` // Foreign key to Bank
    AccountNumber  string    `gorm:"size:50;not null;index"` // Bookie's specific account in the bank
    MinDeposit     int64     // KES in smallest unit (e.g., cents)
    MaxDeposit     int64
    RecentLogRet   float64   // EWMA of log returns
    RecentVol      float64   // EWMA volatility
    CurrentBalance int64

    // Relationship: Belongs to one bank
    Bank           Bank      `gorm:"foreignKey:BankID"`
}
