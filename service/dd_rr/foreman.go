// service/dd_rr/foreman.go
package dd_rr

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"weriKana/models"
        "weriKana/service/keystore"
	"weriKana/service/natsAnish"
	"weriKana/service/otp" // ← now uses the new Service
)

type SmartWithdrawRequest struct {
	CustomerID string `json:"customer_id"`
	OTP        string `json:"otp"`
	Amount     int64  `json:"amount"`
	Signature  []byte `json:"signature"`
	IsReal     bool   `json:"is_real"`
}

func balanceSportsAccount(acct models.SportsAccount, isReal bool) int64 {
	if isReal {
		return acct.RealBalanceCents
	}
	return acct.FakeBalanceCents
}
func SmartWithdraw(db *gorm.DB, keyStore keystore.KeyStore, otpSvc *otp.Service) http.HandlerFunc {
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

                customerID, err := uuid.Parse(req.CustomerID)
                if err != nil {
                        http.Error(w, "invalid customer_id", http.StatusBadRequest)
                        return
                }

                // 1. Verify OTP — using the new Service
                if !otpSvc.Verify(customerID, req.OTP) {
                        http.Error(w, "invalid or expired OTP", http.StatusUnauthorized)
                        return
                }

                // Invalidate OTP after successful use (one-time use!)
                otpSvc.Invalidate(customerID)

                // 2. Verify signature
                signaturePayload := fmt.Sprintf("%s:%s:%d", req.CustomerID, req.OTP, req.Amount)
                if !keyStore.Verify(req.CustomerID, signaturePayload, req.Signature) {
                        http.Error(w, "invalid signature", http.StatusUnauthorized)
                        return
                }

                // --- Rest of your logic stays 100% the same ---
                var accounts []models.SportsAccount
                if err := db.Preload("Bookie").
                        Where("customer_id = ? AND is_active = ?", customerID, true).
                        Find(&accounts).Error; err != nil {
                        log.Printf("Failed to load accounts for customer %s: %v", customerID, err)
                        http.Error(w, "failed to load accounts", http.StatusInternalServerError)
                        return
                }
                if len(accounts) == 0 {
                        http.Error(w, "no active sports accounts", http.StatusBadRequest)
                        return
                }

                var totalPot int64
                for _, acct := range accounts {
                        totalPot += balanceSportsAccount(acct, req.IsReal)
                }
                if totalPot < req.Amount {
                        http.Error(w, "insufficient total balance", http.StatusBadRequest)
                        return
                }

                parentRef := uuid.New().String()
                tx := db.Begin()
                var withdrawals []map[string]interface{}

                for _, acct := range accounts {
                        balance := balanceSportsAccount(acct, req.IsReal)
                        if balance == 0 {
                                continue
                        }
                        proportion := float64(balance) / float64(totalPot)
                        amountToWithdraw := int64(float64(req.Amount) * proportion)
                        if amountToWithdraw <= 0 || amountToWithdraw > balance {
                                amountToWithdraw = balance
                        }

                        updateField := "fake_balance_cents"
                        if req.IsReal {
                                updateField = "real_balance_cents"
                        }
                        if err := db.Model(&acct).Update(updateField, gorm.Expr(updateField+" - ?", amountToWithdraw)).Error; err != nil {
                                tx.Rollback()
                                log.Printf("Failed to update balance: %v", err)
                                http.Error(w, "failed to update balance", http.StatusInternalServerError)
                                return
                        }

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
                                        "stage":        "execution_queued",
                                        "proportion":   proportion,
                                        "bookie_name":  acct.Bookie.Name,
                                },
                                IdempotencyKey: uuid.New().String(),
                        }
                        if err := db.Create(&txn).Error; err != nil {
                                tx.Rollback()
                                log.Printf("Failed to create transaction: %v", err)
                                http.Error(w, "failed to create transaction", http.StatusInternalServerError)
                                return
                        }

                        withdrawals = append(withdrawals, map[string]interface{}{
                                "bookie_account_id": acct.ID.String(),
                                "bookie_name":       acct.Bookie.Name,
                                "amount_cents":      float64(amountToWithdraw),
                                "encrypted_key":     acct.EncryptedKey,
                                "otp":               req.OTP,
                                "transaction_id":    txn.ID.String(),
                        })
                }

                if err := tx.Commit().Error; err != nil {
                        log.Printf("Commit failed: %v", err)
                        http.Error(w, "failed to commit", http.StatusInternalServerError)
                        return
                }

                // Publish to NATS
                payload := map[string]interface{}{
                        "parent_ref":    parentRef,
                        "customer_id":   customerID,
                        "total_cents":   float64(req.Amount),
                        "is_real":       req.IsReal,
                        "withdrawals":   withdrawals,
                        "requested_at":  time.Now().UTC(),
                }
                if msg, err := natsAnish.BuildWithdrawalMessage(payload); err == nil {
                        natsAnish.PublishTransaction(db, msg)
                }

                // Success response
                w.WriteHeader(http.StatusAccepted)
                json.NewEncoder(w).Encode(map[string]interface{}{
                        "status":        "smart_withdraw_queued",
                        "parent_ref":    parentRef[:8],
                        "total_cents":   req.Amount,
                        "is_real":       req.IsReal,
                        "bookies":       len(withdrawals),
                        "pot_balance":   totalPot,
                })
        }
}
// service/securewithdrawal/foreman.go — ADD THIS FUNCTION
func ProcessSmartWithdraw(db *gorm.DB, customerID uuid.UUID, req SmartWithdrawRequest) (parentRef string, totalPot int64, withdrawals []map[string]interface{}, err error) {
	// Reuse the exact same logic from your http handler

	var accounts []models.SportsAccount
	if err := db.Preload("Bookie").
		Where("customer_id = ? AND is_active = ?", customerID, true).
		Find(&accounts).Error; err != nil {
		return "", 0, nil, err
	}
	if len(accounts) == 0 {
		return "", 0, nil, fmt.Errorf("no active accounts")
	}

	totalPot = 0
	for _, acct := range accounts {
		totalPot += balanceSportsAccount(acct, req.IsReal)
	}
	if totalPot < req.Amount {
		return "", totalPot, nil, fmt.Errorf("insufficient balance")
	}

	parentRef = uuid.New().String()
	tx := db.Begin()
	withdrawals = []map[string]interface{}{}

	for _, acct := range accounts {
		balance := balanceSportsAccount(acct, req.IsReal)
		if balance == 0 {
			continue
		}
		proportion := float64(balance) / float64(totalPot)
		amountToWithdraw := int64(float64(req.Amount) * proportion)
		if amountToWithdraw > balance {
			amountToWithdraw = balance
		}
		if amountToWithdraw <= 0 {
			continue
		}

		field := "fake_balance_cents"
		if req.IsReal {
			field = "real_balance_cents"
		}
		if err := tx.Model(&acct).Update(field, gorm.Expr(field+" - ?", amountToWithdraw)).Error; err != nil {
			tx.Rollback()
			return "", 0, nil, err
		}

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
				"stage":        "execution_queued",
				"proportion":   proportion,
				"bookie_name":  acct.Bookie.Name,
			},
		}
		if err := tx.Create(&txn).Error; err != nil {
			tx.Rollback()
			return "", 0, nil, err
		}

		withdrawals = append(withdrawals, map[string]interface{}{
			"bookie_account_id": acct.ID.String(),
			"bookie_name":       acct.Bookie.Name,
			"amount_cents":      amountToWithdraw,
			"encrypted_key":     acct.EncryptedKey,
			"transaction_id":    txn.ID.String(),
		})
	}

	if err := tx.Commit().Error; err != nil {
		return "", 0, nil, err
	}

	return parentRef, totalPot, withdrawals, nil
}
