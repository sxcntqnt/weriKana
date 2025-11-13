package securewithdrawal

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"weriKana/models"             // Import models package
	"weriKana/service/natsAnish"  // Use natsAnish
	"weriKana/service/otp"        // Import otp package
)

// Define KeyStore interface (replace with your actual implementation)
type KeyStore interface {
	Verify(customerID, payload string, signature []byte) bool
}

// Define SmartWithdrawRequest struct (replace with your actual struct)
type SmartWithdrawRequest struct {
	CustomerID string `json:"customer_id"`
	OTP        string `json:"otp"`
	Amount     int64  `json:"amount"`
	Signature  []byte `json:"signature"`
	IsReal     bool   `json:"is_real"`
}

// Helper function to mimic Balance method for SportsAccount
func balanceSportsAccount(acct models.SportsAccount, isReal bool) int64 {
	if isReal {
		return acct.RealBalanceCents
	}
	return acct.FakeBalanceCents
}

// SmartWithdraw — proportional debit + send to Execution Engine
func SmartWithdraw(db *gorm.DB, keyStore KeyStore) http.HandlerFunc {
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
		// Parse CustomerID to uuid.UUID
		customerID, err := uuid.Parse(req.CustomerID)
		if err != nil {
			http.Error(w, "invalid customer_id", http.StatusBadRequest)
			return
		}
		// 1. Verify OTP
		if !otp.Verify(customerID, req.OTP) {
			http.Error(w, "invalid or expired OTP", http.StatusUnauthorized)
			return
		}
		// 2. Verify signature
		payload := fmt.Sprintf("%s:%s:%d", req.CustomerID, req.OTP, req.Amount)
		if !keyStore.Verify(req.CustomerID, payload, req.Signature) {
			http.Error(w, "invalid signature", http.StatusUnauthorized)
			return
		}
		// 3. Fetch all active SportsAccounts
		var accounts []models.SportsAccount
		if err := db.Preload("Bookie").
			Where("customer_id = ? AND is_active = ?", customerID, true).
			Find(&accounts).Error; err != nil {
			http.Error(w, "failed to load accounts", http.StatusInternalServerError)
			return
		}
		if len(accounts) == 0 {
			http.Error(w, "no active sports accounts", http.StatusBadRequest)
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
			balance := balanceSportsAccount(acct, req.IsReal)
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
				if err := db.Model(&acct).Update("real_balance_cents", gorm.Expr("real_balance_cents - ?", amountToWithdraw)).Error; err != nil {
					tx.Rollback()
					http.Error(w, "failed to update balance", http.StatusInternalServerError)
					return
				}
			} else {
				if err := db.Model(&acct).Update("fake_balance_cents", gorm.Expr("fake_balance_cents - ?", amountToWithdraw)).Error; err != nil {
					tx.Rollback()
					http.Error(w, "failed to update balance", http.StatusInternalServerError)
					return
				}
			}
			// Log transaction
			txn := models.Transaction{
				ID:              uuid.New(),
				SportsAccountID: acct.ID,
				CustomerID:      customerID,
				Type:            models.TransactionTypeWithdraw,
				AmountCents:     -amountToWithdraw,
				IsReal:          req.IsReal,
				Status:          models.StatusPending,
				Reference:       parentRef,
				Metadata: models.JSONMap{
					"stage":       "execution_queued",
					"proportion":  proportion,
					"bookie_name": acct.Bookie.Name,
				},
				IdempotencyKey: uuid.New().String(), // Added for compatibility
			}
			if err := db.Create(&txn).Error; err != nil {
				tx.Rollback()
				http.Error(w, "failed to create transaction", http.StatusInternalServerError)
				return
			}
			// Prepare for Execution Engine
			withdrawals = append(withdrawals, map[string]any{
				"bookie_account_id": acct.ID.String(),
				"bookie_name":       acct.Bookie.Name,
				"amount_cents":      amountToWithdraw,
				"encrypted_key":     acct.EncryptedKey,
				"otp":               req.OTP,
				"transaction_id":    txn.ID.String(),
			})
		}
		if err := tx.Commit().Error; err != nil {
			http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
			return
		}
		// 6. Publish to NATS for Execution Engine
		payload := map[string]any{
			"parent_ref":   parentRef,
			"customer_id":  req.CustomerID,
			"total_cents":  req.Amount,
			"is_real":      req.IsReal,
			"otp":          req.OTP,
			"withdrawals":  withdrawals,
			"requested_at": time.Now().UTC(),
		}
		data, err := json.Marshal(payload)
		if err != nil {
			log.Printf("JSON marshal failed: %v", err)
			http.Error(w, "failed to prepare payload", http.StatusInternalServerError)
			return
		}
		if err := natsAnish.Publish("bets.cashout.withdraw", data); err != nil {
			log.Printf("NATS publish failed: %v", err)
			// Not failing the request, as the transaction is committed
		}
		// 7. Response
		w.WriteHeader(http.StatusAccepted)
		if err := json.NewEncoder(w).Encode(map[string]any{
			"status":      "smart_withdraw_queued",
			"parent_ref":  parentRef[:8],
			"total_cents": req.Amount,
			"is_real":     req.IsReal,
			"bookies":     len(withdrawals),
			"pot_balance": totalPot,
		}); err != nil {
			log.Printf("Response encoding failed: %v", err)
		}
	}
}
