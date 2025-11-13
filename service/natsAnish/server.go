package natsAnish
import (
	"encoding/json"
	"log"
	"os"
	"time"

	"weriKana/models"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
	"gorm.io/gorm"
)

// WithdrawalLeg — one bookie to withdraw from
type WithdrawalLeg struct {
	BookieAccountID uuid.UUID `msgpack:"bookie_account_id"`
	BookieName      string    `msgpack:"bookie_name"`
	AmountCents     int64     `msgpack:"amount_cents"`
	EncryptedKey    string    `msgpack:"encrypted_key"`
	OTP             string    `msgpack:"otp"`
	TransactionID   uuid.UUID `msgpack:"transaction_id"`
}

// ExecutionMessage — full payload for Execution Engine
type ExecutionMessage struct {
	ParentRef     string          `msgpack:"parent_ref"`
	CustomerID    uuid.UUID       `msgpack:"customer_id"`
	TotalCents    int64           `msgpack:"total_cents"`
	IsReal        bool            `msgpack:"is_real"`
	Withdrawals   []WithdrawalLeg `msgpack:"withdrawals"`
	RequestedAt   time.Time       `msgpack:"requested_at"`
	IdempotencyKey string         `msgpack:"idempotency_key"`
}

// PublishWithdrawal — called by SmartWithdraw handler
func PublishWithdrawal(db *gorm.DB, payload map[string]any) error {
	// Reconstruct typed message
	var msg ExecutionMessage
	msg.ParentRef = payload["parent_ref"].(string)
	msg.CustomerID = payload["customer_id"].(uuid.UUID)
	msg.TotalCents = int64(payload["total_cents"].(float64))
	msg.IsReal = payload["is_real"].(bool)
	msg.RequestedAt = payload["requested_at"].(time.Time)
	msg.IdempotencyKey = uuid.New().String()

	withdrawals := payload["withdrawals"].([]any)
	for _, w := range withdrawals {
		wd := w.(map[string]any)
		leg := WithdrawalLeg{
			BookieAccountID: uuid.MustParse(wd["bookie_account_id"].(string)),
			BookieName:      wd["bookie_name"].(string),
			AmountCents:     int64(wd["amount_cents"].(float64)),
			EncryptedKey:    wd["encrypted_key"].(string),
			OTP:             wd["otp"].(string),
			TransactionID:   uuid.MustParse(wd["transaction_id"].(string)),
		}
		msg.Withdrawals = append(msg.Withdrawals, leg)
	}

	// Encode as msgpack
	data, err := msgpack.Marshal(&msg)
	if err != nil {
		return err
	}

	// Publish
	return natsclient.Publish("bets.cashout.withdraw", data)
}
func StartExecutionConsumer(db *gorm.DB, nc *nats.Conn) {
    nc.Subscribe("bets.cashout.withdraw", func(m *nats.Msg) {
        var payload map[string]any
        json.Unmarshal(m.Data, &payload)

        withdrawals := payload["withdrawals"].([]any)
        for _, w := range withdrawals {
            wd := w.(map[string]any)
            bookieAcctID := wd["bookie_account_id"].(string)
            amount := wd["amount_cents"].(float64)
            encKey := wd["encrypted_key"].(string)
            otp := wd["otp"].(string)
            txID := wd["transaction_id"].(string)

            // Decrypt key
            key, _ := decrypt(encKey)

            // Call Execution Engine (web scraping)
            go func() {
                result := scrapeAndWithdraw(bookieAcctID, key, otp, int64(amount))
                if result.Success {
                    updateTx(db, txID, models.StatusSuccess, result.Receipt)
                } else {
                    updateTx(db, txID, models.StatusFailed, result.Error)
                }
            }()
        }

        m.Ack()
    })
}
type ScrapingResult struct {
        Success bool
        Receipt string
        Error   string
}

// services/nats_execution.go
func StartExecutionEngine(db *gorm.DB, nc *nats.Conn) {
        nc.Subscribe("bets.cashout.withdraw", func(m *nats.Msg) {
                var payload map[string]any
                json.Unmarshal(m.Data, &payload)

                withdrawals := payload["withdrawals"].([]any)
                for _, w := range withdrawals {
                        wd := w.(map[string]any)
                        encKey := wd["encrypted_key"].(string)
                        otp := wd["otp"].(string)
                        amount := int64(wd["amount_cents"].(float64))
                        txID := wd["transaction_id"].(string)

                        key, _ := decrypt(encKey)

                        go func() {
                                result := executeScraping(key, otp, amount)
                                if result.Success {
                                        updateTx(db, txID, models.StatusSuccess, result.Receipt)
                                } else {
                                        updateTx(db, txID, models.StatusFailed, result.Error)
                                }
                        }()
                }
                m.Ack()
        })
}

