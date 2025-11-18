package natsAnish

import (
	"fmt"
	"log"
        "time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PublishTransaction publishes a transaction message to the appropriate NATS subject
func PublishTransaction(db *gorm.DB, msg *ExecutionMessage) error {
	data, err := EncodeMsg(msg)
	if err != nil {
		return err
	}

	subject := ""
	switch msg.Type {
	case "withdrawal":
		subject = "bets.cashout.withdraw"
	case "deposit":
		subject = "bets.cashin.deposit"
	default:
		return fmt.Errorf("unsupported transaction type: %s", msg.Type)
	}

	log.Printf("📤 Publishing %s to %s", msg.Type, subject)
	return NC.Publish(subject, data)
}

// BuildWithdrawalMessage constructs an ExecutionMessage for withdrawals
func BuildWithdrawalMessage(payload map[string]interface{}) (*ExecutionMessage, error) {
	msg := &ExecutionMessage{
		Type:           "withdrawal",
		ParentRef:      payload["parent_ref"].(string),
		CustomerID:     payload["customer_id"].(uuid.UUID),
		TotalCents:     int64(payload["total_cents"].(float64)),
		IsReal:         payload["is_real"].(bool),
		RequestedAt:    payload["requested_at"].(time.Time),
		IdempotencyKey: uuid.New().String(),
	}

	legs := payload["withdrawals"].([]interface{})
	for _, legData := range legs {
		ld := legData.(map[string]interface{})
		msg.Transactions = append(msg.Transactions, TransactionLeg{
			AccountID:     uuid.MustParse(ld["bookie_account_id"].(string)),
			ProviderName:  ld["bookie_name"].(string),
			AmountCents:   int64(ld["amount_cents"].(float64)),
			EncryptedKey:  ld["encrypted_key"].(string),
			OTP:           ld["otp"].(string),
			TransactionID: uuid.MustParse(ld["transaction_id"].(string)),
		})
	}
	return msg, nil
}
