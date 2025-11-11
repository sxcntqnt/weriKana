// api/handlers/smart-withdraw.go
package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"werikana/models"
	"werikana/services/natsclient"
	"werikana/services/otp"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SmartWithdrawRequest — incoming API
type SmartWithdrawRequest struct {
	CustomerID uuid.UUID `json:"customer_id"`
	OTP        string    `json:"otp"`
	Signature  string    `json:"signature"` // HMAC(payload, sender_key)
	Amount     int64     `json:"amount_cents"` // total to withdraw
	IsReal     bool      `json:"is_real"`
}

// SmartWithdraw — proportional debit + send to Execution Engine
func SmartWithdraw(db *gorm.DB, keyStore *KeyStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SmartWithdrawRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Amount <= 0 {
			http.Error(w, "amount must be > 0", http.StatusBadRequest)
			return
		}

		// 1. Verify OTP
		if !otp.Verify(req.CustomerID, req.OTP) {
			http.Error(w, "invalid or expired OTP", http.StatusUnauthorized)
			return
		}

		// 2. Verify signature
		payload := fmt.Sprintf("%s:%s:%d", req.CustomerID, req.OTP, req.Amount)
		if !keyStore.Verify(req.CustomerID, payload, req.Signature) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}

		// 3. Fetch all active BookieAccounts
		var accounts []models.BookieAccount
		if err := db.Preload("Bookie").
			Where("customer_id = ? AND is_active = ?", req.CustomerID, true).
			Find(&accounts).Error; err != nil {
			http.Error(w, "failed to load accounts", http.StatusInternalServerError)
			return
		}

		if len(accounts) == 0 {
			http.Error(w, "no active bookie accounts", http.StatusBadRequest)
			return
		}

		// 4. Calculate pot balance
		var totalPot int64
		for _, acct := range accounts {
			if req.IsReal {
				totalPot += acct.RealBalanceCents
			} else {
				totalPot += acct.FakeBalanceCents
			}
		}

		if totalPot < req.Amount {
			http.Error(w, "insufficient total balance", http.StatusBadRequest)
			return
		}

		// 5. Proportional withdrawal plan
		parentRef := uuid.New().String()
		tx := db.Begin()
		var withdrawals []map[string]any

		for _, acct := range accounts {
			balance := acct.Balance(req.IsReal)
			if balance == 0 {
				continue
			}

			proportion := float64(balance) / float64(totalPot)
			amountToWithdraw := int64(float64(req.Amount) * proportion)
			if amountToWithdraw <= 0 {
				continue
			}

			if balance < amountToWithdraw {
				amountToWithdraw = balance
			}

			// Debit balance
			if req.IsReal {
				db.Model(&acct).Update("real_balance_cents", gorm.Expr("real_balance_cents - ?", amountToWithdraw))
			} else {
				db.Model(&acct).Update("fake_balance_cents", gorm.Expr("fake_balance_cents - ?", amountToWithdraw))
			}

			// Log transaction
			txn := models.Transaction{
				ID:              uuid.New(),
				BookieAccountID: acct.ID,
				CustomerID:      req.CustomerID,
				Type:            models.TransactionTypeWithdraw,
				AmountCents:     -amountToWithdraw,
				IsReal:          req.IsReal,
				Status:          models.StatusPending,
				Reference:       parentRef,
				Metadata: models.JSONMap{
					"stage":         "execution_queued",
					"proportion":    proportion,
					"bookie_name":   acct.Bookie.Name,
				},
			}
			db.Create(&txn)

			// Prepare for Execution Engine
			withdrawals = append(withdrawals, map[string]any{
				"bookie_account_id": acct.ID,
				"bookie_name":       acct.Bookie.Name,
				"amount_cents":      amountToWithdraw,
				"encrypted_key":     acct.EncryptedKey,
				"otp":               req.OTP,
				"transaction_id":    txn.ID,
			})
		}

		tx.Commit()

		// 6. Publish to NATS for Execution Engine
		payload := map[string]any{
			"parent_ref":    parentRef,
			"customer_id":   req.CustomerID,
			"total_cents":   req.Amount,
			"is_real":       req.IsReal,
			"otp":           req.OTP,
			"withdrawals":   withdrawals,
			"requested_at":  time.Now().UTC(),
		}
		data, _ := json.Marshal(payload)
		if err := natsclient.Publish("bets.cashout.withdraw", data); err != nil {
			log.Printf("NATS publish failed: %v", err)
		}

		// 7. Response
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{
			"status":       "smart_withdraw_queued",
			"parent_ref":   parentRef[:8],
			"total_cents":  req.Amount,
			"is_real":      req.IsReal,
			"bookies":      len(withdrawals),
			"pot_balance":  totalPot,
		})
	}
}
