// models/sharp.go
// Merged Customer & Sharp model — Single entity with separate real/fake balance tracking.
// Real/Fake: Distinct fields + ledger entities for history.

package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)



type Sharp struct {
	gorm.Model
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey"` // UUID gen in BeforeCreate
	// Merged from Customer
	Name             string         `gorm:"size:255;not null" json:"name"`
	Email            string         `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PhoneNumbers     PhoneNumbers   `gorm:"type:json;not null" json:"phone_numbers"` // Multiple: primary + fallbacks (at least 1)
	PreferredMpesa   string         `gorm:"size:20" json:"preferred_mpesa"` // Optional fallback MPESA (separate for payouts)
	// Merged from Sharp
	SportsBankID     uuid.UUID      `gorm:"type:uuid;index"`
	StockBankID      uuid.UUID      `gorm:"type:uuid;index"`
	ForexBankID      uuid.UUID      `gorm:"type:uuid;index"`
	CryptoBankID     uuid.UUID      `gorm:"type:uuid;index"`
	AccountNumber    string         `gorm:"size:50;not null;index"`
	MinTradeCents    int64          `gorm:"default:100"` // 1 KES = 100 cents
	MaxTradeCents    int64          `gorm:"default:10000000"` // 100,000 KES
	// Separate Real & Fake Balances (aggregated from accounts/ledgers)
	RealBalanceCents int64          `gorm:"default:0" json:"real_balance_cents"` // Actual funds
	FakeBalanceCents int64          `gorm:"default:0" json:"fake_balance_cents"` // Bonus/simulated
	// Merged Relations (all FK to Sharp.ID)
	SportsAccounts   []SportsAccount `gorm:"foreignKey:SharpID" json:"-"`
	StockAccounts    []StockAccount  `gorm:"foreignKey:SharpID" json:"-"`
	ForexAccounts    []ForexAccount  `gorm:"foreignKey:SharpID" json:"-"`
	CryptoAccounts   []CryptoAccount `gorm:"foreignKey:SharpID" json:"-"`
	SharpAccounts    []SharpAccount  `gorm:"foreignKey:SharpID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`
	// Real/Fake Ledger Relations (separate entities for history)
	RealLedgerEntries []RealLedgerEntry `gorm:"foreignKey:SharpID" json:"-"`
	FakeLedgerEntries []FakeLedgerEntry `gorm:"foreignKey:SharpID" json:"-"`
	// Bank Relations (from Sharp)
	SportsBank       SportsBank      `gorm:"foreignKey:SportsBankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	StockBank        StockBank       `gorm:"foreignKey:StockBankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	ForexBank        ForexBank       `gorm:"foreignKey:ForexBankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	CryptoBank       CryptoBank      `gorm:"foreignKey:CryptoBankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT" json:"-"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	DeletedAt        gorm.DeletedAt  `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate GORM hook → gen UUID; validate email; ensure at least one phone; normalize phones; ensure exactly one bank ID.
func (s *Sharp) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Email == "" {
		return fmt.Errorf("email is required")
	}
	if len(s.PhoneNumbers) == 0 {
		return fmt.Errorf("at least one phone number is required for fallback")
	}
	// Validate & normalize each phone
	for i, phone := range s.PhoneNumbers {
		if !kenyanPhoneRegex.MatchString(phone) {
			return fmt.Errorf("invalid Kenyan phone number at index %d: %s", i, phone)
		}
		// Normalize: always store as +254...
		if len(phone) == 10 && phone[0] == '0' {
			s.PhoneNumbers[i] = "+254" + phone[1:]
		}
	}
	// Normalize PreferredMpesa if set
	if s.PreferredMpesa != "" && !kenyanPhoneRegex.MatchString(s.PreferredMpesa) {
		return fmt.Errorf("invalid preferred MPESA number")
	}
	if s.PreferredMpesa != "" && len(s.PreferredMpesa) == 10 && s.PreferredMpesa[0] == '0' {
		s.PreferredMpesa = "+254" + s.PreferredMpesa[1:]
	}
	// Sharp-specific: Exactly one bank ID
	count := 0
	if s.SportsBankID != uuid.Nil {
		count++
	}
	if s.StockBankID != uuid.Nil {
		count++
	}
	if s.ForexBankID != uuid.Nil {
		count++
	}
	if s.CryptoBankID != uuid.Nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("exactly one bank ID must be set")
	}
	return nil
}

// AfterSave hook → Recalculate aggregated balances from ledgers/accounts.
func (s *Sharp) AfterSave(tx *gorm.DB) error {
	// Sum real/fake from ledgers (primary source)
	var realSum, fakeSum int64
	if err := tx.Model(&RealLedgerEntry{}).Where("sharp_id = ?", s.ID).Select("SUM(amount_cents)").Scan(&realSum).Error; err != nil {
		return fmt.Errorf("real balance aggregation failed: %w", err)
	}
	if err := tx.Model(&FakeLedgerEntry{}).Where("sharp_id = ?", s.ID).Select("SUM(amount_cents)").Scan(&fakeSum).Error; err != nil {
		return fmt.Errorf("fake balance aggregation failed: %w", err)
	}
	s.RealBalanceCents = realSum
	s.FakeBalanceCents = fakeSum
	return tx.Save(s).Error
}
