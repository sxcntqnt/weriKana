// api/handlers/natsConsumer.go
package handlers

import (
	"encoding/json"
	"log"
        "fmt"
 	"time"

	"github.com/nats-io/nats.go"
	"weriKana/models"
	"weriKana/service/mpesa"
	"gorm.io/gorm"
)

// WithdrawalMessage — adjust fields to match what you publish
type WithdrawalMessage struct {
	TransactionID   string `json:"transaction_id"`
	PhoneNumber     string `json:"phone_number"`
	AmountCents     int64  `json:"amount_cents"`
	IdempotencyKey  string `json:"idempotency_key,omitempty"`
	CustomerID      string `json:"customer_id,omitempty"`
}

var nc *nats.Conn

// StartWithdrawalConsumer — call this from main.go instead of ListenForWithdrawals
func StartWithdrawalConsumer(db *gorm.DB, natsConn *nats.Conn) {
	nc = natsConn

	_, err := nc.Subscribe("bets.cashout.withdraw", func(msg *nats.Msg) {
		go processWithdrawal(db, msg) // async so we ack fast
		msg.Ack()
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to withdrawals: %v", err)
	}

	log.Println("Withdrawal consumer started on subject: bets.cashout.withdraw")
}

// processWithdrawal — now uses your real SendB2C function
func processWithdrawal(db *gorm.DB, msg *nats.Msg) {
	var wm WithdrawalMessage
	if err := json.Unmarshal(msg.Data, &wm); err != nil {
		log.Printf("Invalid withdrawal message: %v", err)
		return
	}

	log.Printf("Processing withdrawal for %s → %s KES (idempotency: %s)",
		wm.PhoneNumber, fmt.Sprintf("%.2f", float64(wm.AmountCents)/100), wm.IdempotencyKey)

	// Use your real B2C function
	resp, err := mpesa.SendB2C(wm.PhoneNumber, wm.AmountCents, wm.IdempotencyKey)
	if err != nil {
		log.Printf("M-Pesa B2C failed for %s: %v", wm.TransactionID, err)
		// TODO: publish to DLQ or retry queue
		return
	}

	log.Printf("B2C initiated successfully! ConversationID: %s, Originator: %s",
		resp.ConversationID, resp.OriginatorConvID)

	// Optional: update transaction status in DB
	var tx models.Transaction
	if err := db.Where("id = ? OR metadata->>'third_party_ref' = ?", wm.TransactionID, wm.IdempotencyKey).First(&tx).Error; err != nil {
		log.Printf("Transaction not found for withdrawal: %v", err)
		return
	}

	// Mark as pending M-Pesa processing
	db.Model(&tx).Updates(map[string]any{
		"status": models.StatusPending,
		"metadata": models.JSONMap{
			"mpesa_conversation_id":     resp.ConversationID,
			"mpesa_originator_conv_id":  resp.OriginatorConvID,
			"withdrawal_initiated_at":   time.Now(),
		},
	})

	log.Printf("Withdrawal queued successfully for tx %s", tx.ID)
}
