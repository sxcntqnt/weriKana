package models

import (
	"database/sql/driver"
	"encoding/json"
        "github.com/google/uuid"
        "fmt"
)

// JSONMap is a convenience type for GORM JSON columns
type JSONMap map[string]interface{}

// Value implements Valuer for SQL serialization.
func (j JSONMap) Value() (driver.Value, error) {
        if len(j) == 0 {
                return nil, nil
        }
        b, err := json.Marshal(j)
        return string(b), err
}

// Scan implements Scanner for SQL deserialization.
func (j *JSONMap) Scan(value interface{}) error {
        if value == nil {
                *j = nil
                return nil
        }
        b, ok := value.([]byte)
        if !ok {
                return fmt.Errorf("JSONMap: cannot scan []byte")
        }
        return json.Unmarshal(b, j)
}

type AllocationResult struct {
    BookieID       uuid.UUID
    BookieName     string
    MpesaNumber    string
    AmountToSend   int64 // cents
    Proportion     float64
    IsReal         bool
    IdempotencyKey string
    TransactionID  uuid.UUID
}

// ------------------------------------------------------------
// WithdrawalItem and SecureWithdrawalEnvelope for NATS
type WithdrawalItem struct {
	BookieID    uint64 `msgpack:"b" json:"bookie_id"`
	AmountCents int64  `msgpack:"a" json:"amount_cents"`
}

// SecureWithdrawalEnvelope – msgpack-encoded payload sent over NATS
type SecureWithdrawalEnvelope struct {
	_             struct{}         `msgpack:",omitempty"` // force field order
	TransactionID uint64           `msgpack:"t"`
	Items         []WithdrawalItem `msgpack:"i"`
	OTPNonce      []byte           `msgpack:"n"` // 12-byte nonce
	OTPEncrypted  []byte           `msgpack:"o"` // AES-GCM ciphertext
	Timestamp     int64            `msgpack:"s"`
	Idempotency   string           `msgpack:"d"`
	Signature     []byte           `msgpack:"S"` // Ed25519 over all fields except this one
}

// Define the possible transaction statuses
type TransactionStatus string
const (
    StatusPending   TransactionStatus = "pending"
    StatusInitiated TransactionStatus = "initiated"
    StatusCompleted TransactionStatus = "completed"
    StatusFailed    TransactionStatus = "failed"
    StatusSuccess   TransactionStatus = "success"  // Added StatusSuccess
)
