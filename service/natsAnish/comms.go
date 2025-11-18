package natsAnish

import (
	"log"
	"os"
	"time"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
)

// NATS Connection Management
var NC *nats.Conn

func InitNATS() {
	url := os.Getenv("NATS_URL")
	if url == "" {
		url = nats.DefaultURL
	}

	var err error
	NC, err = nats.Connect(url)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}

	log.Println("✅ Connected to NATS:", url)
}

func CloseNATS() {
	if NC != nil {
		NC.Close()
	}
}

// Message Structures
type UUID uuid.UUID

func (u UUID) MarshalMsgpack() ([]byte, error) {
	return uuid.UUID(u).MarshalBinary()
}

func (u *UUID) UnmarshalMsgpack(data []byte) error {
	return (*uuid.UUID)(u).UnmarshalBinary(data)
}

type TransactionLeg struct {
	AccountID     uuid.UUID `msgpack:"account_id"`
	ProviderName  string    `msgpack:"provider_name"`
	AmountCents   int64     `msgpack:"amount_cents"`
	EncryptedKey  string    `msgpack:"encrypted_key"`
	OTP           string    `msgpack:"otp"`
	TransactionID uuid.UUID `msgpack:"transaction_id"`
}

type ExecutionMessage struct {
	Type           string          `msgpack:"type"`
	ParentRef      string          `msgpack:"parent_ref"`
	CustomerID     uuid.UUID       `msgpack:"customer_id"`
	TotalCents     int64           `msgpack:"total_cents"`
	IsReal         bool            `msgpack:"is_real"`
	Transactions   []TransactionLeg `msgpack:"transactions"`
	RequestedAt    time.Time       `msgpack:"requested_at"`
	IdempotencyKey string          `msgpack:"idempotency_key"`
}

func EncodeMsg(msg *ExecutionMessage) ([]byte, error) {
	return msgpack.Marshal(msg)
}

func DecodeMsg(data []byte) (*ExecutionMessage, error) {
	var msg ExecutionMessage
	if err := msgpack.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func init() {
	msgpack.RegisterExt(0, (*UUID)(nil))
}

func Publish(subject string, data []byte) error {
    if NC == nil {
        return fmt.Errorf("NATS connection not initialized")
    }
    return NC.Publish(subject, data)
}
