// api/handlers/smart-mpesa-depo.go
package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
	"weriKana/models"
)

type DepositRequest struct {
	CustomerID  uuid.UUID `json:"customer_id,omitempty"`
	Phone       string    `json:"phone"`
	AmountCents int64     `json:"amount_cents"`
	IsReal      bool      `json:"is_real"`
	DryRun      *bool     `json:"dry_run,omitempty"`
	Beta        float64   `json:"beta,omitempty"`
	ReservePct  float64   `json:"reserve_pct,omitempty"`
}

// 1. SmartDeposit — the quant-powered beast
func SmartDeposit(db *gorm.DB, nc *nats.Conn) fiber.Handler {
	return func(c *fiber.Ctx) error {
		customerID := c.Locals("user_id").(uuid.UUID)

		var req DepositRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
		}
		if req.AmountCents <= 0 {
			return c.Status(400).JSON(fiber.Map{"error": "Amount must be positive"})
		}
		if req.Phone == "" {
			return c.Status(400).JSON(fiber.Map{"error": "Phone number required"})
		}

		var accounts []models.SportsAccount
		if err := db.Preload("Bookie").
			Where("customer_id = ? AND is_active = ?", customerID, true).
			Find(&accounts).Error; err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "Failed to load accounts"})
		}
		if len(accounts) == 0 {
			return c.Status(400).JSON(fiber.Map{"error": "No active bookie accounts"})
		}

		bookies := make([]models.Bookie, len(accounts))
		var totalPot int64
		for i, acct := range accounts {
			balance := acct.RealBalanceCents
			if !req.IsReal {
				balance = acct.FakeBalanceCents
			}
			totalPot += balance

			bookies[i] = models.Bookie{
				ID:             acct.Bookie.ID,
				Name:           acct.Bookie.Name,
				MpesaNumber:    acct.Bookie.MpesaNumber,
				MinDeposit:     acct.Bookie.MinDeposit,
				MaxDeposit:     acct.Bookie.MaxDeposit,
				RecentLogRet:   acct.Bookie.RecentLogRet,
				RecentVol:      acct.Bookie.RecentVol,
				CurrentBalance: balance,
			}
		}

		beta := 0.7
		if req.Beta > 0 && req.Beta <= 1 {
			beta = req.Beta
		}
		reservePct := 0.10
		if req.ReservePct > 0 && req.ReservePct < 0.5 {
			reservePct = req.ReservePct
		}

		allocations := AllocateFunds(totalPot, reservePct, bookies, beta, 10000)
		parentRef := uuid.New().String()
		var results []models.AllocationResult

		for _, alloc := range allocations {
			if alloc.AmountToSend == 0 {
				continue
			}

			idempotency := fmt.Sprintf("smartdep-%s-%s", parentRef[:8], alloc.IdempotencyKey[:8])

			metadata := models.JSONMap{
				"algorithm":   "risk_perf_weighted_v2",
				"beta":        beta,
				"reserve_pct": reservePct,
				"proportion":  alloc.Proportion,
				"idempotency": idempotency,
				"phone":       req.Phone,
				"bookie":      alloc.BookieName,
				"parent_ref":  parentRef,
			}

			tx, err := BaseDeposit(db, customerID, "bookie", alloc.BookieID, alloc.AmountToSend, req.IsReal, metadata, parentRef, alloc.BookieID)
			if err != nil {
				return c.Status(500).JSON(fiber.Map{"error": err.Error()})
			}

			results = append(results, models.AllocationResult{
				BookieID:       alloc.BookieID,
				BookieName:     alloc.BookieName,
				MpesaNumber:    alloc.MpesaNumber,
				AmountToSend:   alloc.AmountToSend,
				Proportion:     alloc.Proportion,
				IsReal:         req.IsReal,
				IdempotencyKey: idempotency,
				TransactionID:  tx.ID,
			})
		}

		isDryRun := req.DryRun != nil && *req.DryRun
		if req.IsReal && !isDryRun && len(results) > 0 {
			payload := map[string]any{
				"parent_ref":   parentRef,
				"phone":        req.Phone,
				"total_cents":  req.AmountCents,
				"allocations":  results,
				"customer_id":  customerID,
				"algorithm":    "risk_perf_weighted_v2",
				"beta":         beta,
				"timestamp":     time.Now().UTC(),
			}
			if data, err := json.Marshal(payload); err == nil {
				nc.Publish("mpesa.stk.sequence", data)
			}
		}

		if !req.IsReal || isDryRun {
			for _, r := range results {
				db.Model(&models.Transaction{}).Where("id = ?", r.TransactionID).
					Update("status", models.StatusSuccess)
			}
		}

		return c.JSON(fiber.Map{
			"status":            "smart_deposit_queued",
			"parent_ref":        parentRef[:8],
			"is_real":           req.IsReal,
			"dry_run":           isDryRun,
			"algorithm":         "risk_perf_weighted_v2",
			"beta":              beta,
			"reserve_pct":       reservePct,
			"total_allocated":   req.AmountCents,
			"allocations_count": len(results),
			"pot_balance_cents": totalPot,
			"allocations":       results,
		})
	}
}

// 2. AccountDeposit — single account deposits (sharp, stock, etc.)
func AccountDeposit(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		customerID := c.Locals("user_id").(uuid.UUID)
		accountType := c.Locals("account_type").(string)

		var req DepositRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
		}
		if req.AmountCents <= 0 {
			return c.Status(400).JSON(fiber.Map{"error": "Amount must be positive"})
		}

		var accountID uuid.UUID
		switch accountType {
		case "sharp":
			var acc models.SharpAccount
			if err := db.Select("id").First(&acc, "customer_id = ?", customerID).Error; err != nil {
				return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
			}
			accountID = acc.ID
		default:
			return c.Status(400).JSON(fiber.Map{"error": "Unsupported account type"})
		}

		ref := "DEP-" + uuid.New().String()[:8]
		tx, err := BaseDeposit(db, customerID, accountType, accountID, req.AmountCents, req.IsReal, models.JSONMap{}, ref, uuid.Nil)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}

		db.Model(tx).Update("status", models.StatusSuccess)

		return c.JSON(fiber.Map{
			"message":        "Deposit successful",
			"transaction_id": tx.ID,
			"amount_cents":   req.AmountCents,
			"is_real":        req.IsReal,
		})
	}
}

// 3. BaseDeposit — shared core logic
func BaseDeposit(
	db *gorm.DB,
	customerID uuid.UUID,
	accountType string,
	accountID uuid.UUID,
	amountCents int64,
	isReal bool,
	metadata models.JSONMap,
	reference string,
	bookieAccountID uuid.UUID,
) (*models.Transaction, error) {

	tx := models.Transaction{
		ID:              uuid.New(),
		CustomerID:      customerID,
		Type:            models.TransactionTypeDeposit,
		AmountCents:     amountCents,
		IsReal:          isReal,
		Currency:        "KES",
		Status:          models.StatusPending,
		Metadata:        metadata,
		Reference:       reference,
		BookieAccountID: bookieAccountID,
	}

	var err error
	switch accountType {
	case "sharp":
		var acc models.SharpAccount
		err = db.First(&acc, "id = ? AND customer_id = ?", accountID, customerID).Error
		if err != nil {
			return nil, fmt.Errorf("sharp account not found")
		}
		if isReal {
			acc.RealBalanceCents += amountCents
		} else {
			acc.FakeBalanceCents += amountCents
		}
		err = db.Save(&acc).Error
		tx.SharpAccountID = acc.ID

	case "sports", "stock", "forex", "crypto":
		// Add more if needed
		return nil, fmt.Errorf("account type %s not implemented in BaseDeposit", accountType)

	case "bookie":
		var acc models.SportsAccount
		err = db.First(&acc, "id = ? AND customer_id = ?", accountID, customerID).Error
		if err != nil {
			return nil, fmt.Errorf("bookie account not found")
		}
		if isReal {
			acc.RealBalanceCents += amountCents
		} else {
			acc.FakeBalanceCents += amountCents
		}
		err = db.Save(&acc).Error
		tx.BookieAccountID = acc.ID
	}

	if err != nil {
		return nil, err
	}
	if err := db.Create(&tx).Error; err != nil {
		return nil, err
	}
	return &tx, nil
}
