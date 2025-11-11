// services/mpesa_depo_sequence.go
package services

import (
	"encoding/json"
	"log"
	"time"

	"werikana/models"
	"werikana/services/mpesa"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

type StkSequencePayload struct {
	ParentRef    string            `json:"parent_ref"`
	Phone        string            `json:"phone"`
	TotalCents   int64             `json:"total_cents"`
	Allocations  []AllocationResult `json:"allocations"`
	CustomerID   uuid.UUID         `json:"customer_id"`
}

func StartStkSequenceConsumer(db *gorm.DB, nc *nats.Conn) {
	nc.Subscribe("mpesa.stk.sequence", func(m *nats.Msg) {
		var payload StkSequencePayload
		if err := json.Unmarshal(m.Data, &payload); err != nil {
			log.Printf("invalid payload: %v", err)
			return
		}

		// Sequential STK Push
		for i, a := range payload.Allocations {
			time.Sleep(2 * time.Second) // avoid rate limits

			resp, err := mpesa.SendSTKPush(a.MpesaNumber, a.AmountToSend, a.IdempotencyKey)
			if err != nil {
				updateTxStatus(db, a.TransactionID, models.StatusFailed, err.Error())
				continue
			}

			updateTxStatus(db, a.TransactionID, models.StatusInitiated, resp.CheckoutRequestID)
		}

		m.Ack()
	})
}

func updateTxStatus(db *gorm.DB, txID uuid.UUID, status models.TransactionStatus, meta string) {
	db.Model(&models.Transaction{}).Where("id = ?", txID).
		Updates(map[string]any{
			"status": status,
			"metadata": models.JSONMap{
				"third_party_ref": meta,
				"updated_at":      time.Now(),
			},
		})
}
