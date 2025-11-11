// api/handlers/smart-depo.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"werikana/models"
	"werikana/services/natsclient"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AllocationResult — one deposit leg
type AllocationResult struct {
	BookieID        uuid.UUID
	BookieName      string
	MpesaNumber     string
	AmountToSend    int64 // cents
	Proportion      float64
	IsReal          bool
	IdempotencyKey  string
	TransactionID   uuid.UUID
}

// SmartDepositRequest — incoming API
type SmartDepositRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Phone      string    `json:"phone"` // +254...
	Amount     int64     `json:"amount_cents"`
	IsReal     bool      `json:"is_real"`     // true = MPESA, false = fake
	DryRun     *bool     `json:"dry_run,omitempty"`     // optional

}

// SmartDeposit — proportional funding across bookie accounts
func SmartDeposit(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SmartDepositRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 {
			http.Error(w, "amount must be > 0", http.StatusBadRequest)
			return
		}

		// 1. Fetch all active BookieAccounts for this customer
		var accounts []models.BookieAccount
		if err := db.
			Preload("Bookie").
			Where("customer_id = ? AND is_active = ?", req.CustomerID, true).
			Find(&accounts).Error; err != nil {
			http.Error(w, "failed to load accounts", http.StatusInternalServerError)
			return
		}

		if len(accounts) == 0 {
			http.Error(w, "no active bookie accounts", http.StatusBadRequest)
			return
		}

		// 2. Calculate total pot balance (real or fake)
		var totalPot int64
		for _, acct := range accounts {
			if req.IsReal {
				totalPot += acct.RealBalanceCents
			} else {
				totalPot += acct.FakeBalanceCents
			}
		}

		if totalPot == 0 && !req.DryRun {
			http.Error(w, "total pot balance is zero", http.StatusBadRequest)
			return
		}

		// 3. Build allocation plan
		var allocs []AllocationResult
		parentRef := uuid.New().String()

		for _, acct := range accounts {
			var balance int64
			if req.IsReal {
				balance = acct.RealBalanceCents
			} else {
				balance = acct.FakeBalanceCents
			}

			proportion := float64(balance) / float64(totalPot)
			if totalPot == 0 {
				proportion = 1.0 / float64(len(accounts)) // equal split if zero
			}

			amountToSend := int64(float64(req.Amount) * proportion)
			if amountToSend <= 0 {
				continue
			}

			idempotency := fmt.Sprintf("%s-%d-%d", acct.Bookie.Name, time.Now().UnixNano(), amountToSend)

			// Record pending transaction
			tx := models.Transaction{
				ID:              uuid.New(),
				BookieAccountID: acct.ID,
				CustomerID:      req.CustomerID,
				SenderID:        uuid.Nil, // system
				Type:            models.TransactionTypeDeposit,
				AmountCents:     amountToSend,
				IsReal:          req.IsReal,
				Status:          models.StatusPending,
				Reference:       parentRef,
				Metadata: models.JSONMap{
					"proportion":     proportion,
					"idempotency":    idempotency,
					"phone":          req.Phone,
					"bookie":         acct.Bookie.Name,
					"dry_run":        req.DryRun,
				},
			}
			if err := db.Create(&tx).Error; err != nil {
				http.Error(w, "failed to record tx", http.StatusInternalServerError)
				return
			}

			allocs = append(allocs, AllocationResult{
				BookieID:       acct.Bookie.ID,
				BookieName:     acct.Bookie.Name,
				MpesaNumber:    acct.Bookie.MpesaNumber,
				AmountToSend:   amountToSend,
				Proportion:     proportion,
				IsReal:         req.IsReal,
				IdempotencyKey: idempotency,
				TransactionID:  tx.ID,
			})
		}

		// 4. If not dry-run & real → publish to NATS for sequential STK
		if req.IsReal && !req.DryRun && len(allocs) > 0 {
			payload := map[string]any{
				"parent_ref":   parentRef,
				"phone":        req.Phone,
				"total_cents":  req.Amount,
				"allocations":  allocs,
				"customer_id":  req.CustomerID,
				"timestamp":    time.Now().UTC(),
			}
			data, _ := json.Marshal(payload)
			if err := natsclient.Publish("mpesa.stk.sequence", data); err != nil {
				log.Printf("NATS publish failed: %v", err)
				// Don't fail request — async retry
			}
		}

		// 5. If fake or dry-run → credit instantly
		if !req.IsReal || req.DryRun {
			tx := db.Begin()
			for _, a := range allocs {
				var acct models.BookieAccount
				db.First(&acct, "id = ?", a.BookieID)

				if req.IsReal {
					db.Model(&acct).Update("real_balance_cents", gorm.Expr("real_balance_cents + ?", a.AmountToSend))
				} else {
					db.Model(&acct).Update("fake_balance_cents", gorm.Expr("fake_balance_cents + ?", a.AmountToSend))
				}

				// Update status
				db.Model(&models.Transaction{}).Where("id = ?", a.TransactionID).
					Updates(map[string]any{
						"status": models.StatusSuccess,
						"metadata": models.JSONMap{
							"final_status": "instant_credited",
							"dry_run":      req.DryRun,
						},
					})
			}
			tx.Commit()
		}

		// 6. Response
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":          "smart_deposit_initiated",
			"parent_ref":      parentRef[:8],
			"is_real":         req.IsReal,
			"dry_run":         req.DryRun,
			"total_allocated": req.Amount,
			"allocations":     len(allocs),
			"pot_balance":     totalPot,
		})
	}
}
