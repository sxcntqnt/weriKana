// api/handlers/deposit.go
package handlers

import (
    "encoding/json"
    "fmt"
    "log"
    "time"
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "github.com/nats-io/nats.go"
    "gorm.io/gorm"
    "weriKana/models"
)

// JSONMap is a map for JSON data
type JSONMap map[string]interface{}

// AllocationResult represents one deposit leg for smart allocation
type AllocationResult struct {
    BookieID       uuid.UUID
    BookieName     string
    MpesaNumber    string
    AmountToSend   int64 // cents
    Proportion     float64
    IsReal         bool
    IdempotencyKey string
    TransactionID  uuid.UUID
}

// DepositRequest is the incoming API request
type DepositRequest struct {
    CustomerID  uuid.UUID `json:"customer_id"`
    Phone       string    `json:"phone"`           // +254..., required for smart deposits
    AmountCents int64     `json:"amount_cents"`
    IsReal      bool      `json:"is_real"`        // true = MPESA for smart, real balance for single
    DryRun      *bool     `json:"dry_run,omitempty"` // optional, for smart deposits
    AssetData   JSONMap   `json:"asset_data"`     // metadata for single-account deposits
    BookieID    uuid.UUID `json:"bookie_id,omitempty"` // optional, for bookie account deposits
}

// BaseDeposit updates an account's balance and creates a Transaction
func BaseDeposit(
    db *gorm.DB,
    customerID uuid.UUID,
    accountType string,
    accountID uuid.UUID,
    amountCents int64,
    isReal bool,
    metadata JSONMap,
    reference string,
    bookieAccountID uuid.UUID, // uuid.Nil for non-bookie accounts
) (*models.Transaction, error) {
    tx := models.Transaction{
        ID:              uuid.New(),
        CustomerID:      customerID,
        Type:            models.TransactionTypeDeposit,
        AmountCents:     amountCents,
        IsReal:          isReal,
        Currency:        "KES",
        Status:          models.StatusSuccess,
        Metadata:        metadata,
        Reference:       reference,
        BookieAccountID: bookieAccountID,
    }

    switch accountType {
    case "sharp":
        var acc models.SharpAccount
        if err := db.Where("id = ? AND customer_id = ?", accountID, customerID).First(&acc).Error; err != nil {
            return nil, fmt.Errorf("account not found")
        }
        if isReal {
            acc.RealBalanceCents += amountCents
        } else {
            acc.FakeBalanceCents += amountCents
        }
        if err := db.Save(&acc).Error; err != nil {
            return nil, fmt.Errorf("failed to update account")
        }
        tx.SharpAccountID = acc.ID
    case "sports":
        var acc models.SportsAccount
        if err := db.Where("id = ? AND customer_id = ?", accountID, customerID).First(&acc).Error; err != nil {
            return nil, fmt.Errorf("account not found")
        }
        if isReal {
            acc.RealBalanceCents += amountCents
        } else {
            acc.FakeBalanceCents += amountCents
        }
        acc.TradeHistory = metadata
        if err := db.Save(&acc).Error; err != nil {
            return nil, fmt.Errorf("failed to update account")
        }
        tx.SportsAccountID = acc.ID
    case "stock":
        var acc models.StockAccount
        if err := db.Where("id = ? AND customer_id = ?", accountID, customerID).First(&acc).Error; err != nil {
            return nil, fmt.Errorf("account not found")
        }
        if isReal {
            acc.RealBalanceCents += amountCents
        } else {
            acc.FakeBalanceCents += amountCents
        }
        acc.Portfolio = metadata
        if err := db.Save(&acc).Error; err != nil {
            return nil, fmt.Errorf("failed to update account")
        }
        tx.StockAccountID = acc.ID
    case "forex":
        var acc models.ForexAccount
        if err := db.Where("id = ? AND customer_id = ?", accountID, customerID).First(&acc).Error; err != nil {
            return nil, fmt.Errorf("account not found")
        }
        if isReal {
            acc.RealBalanceCents += amountCents
        } else {
            acc.FakeBalanceCents += amountCents
        }
        acc.OpenPositions = metadata
        if err := db.Save(&acc).Error; err != nil {
            return nil, fmt.Errorf("failed to update account")
        }
        tx.ForexAccountID = acc.ID
    case "crypto":
        var acc models.CryptoAccount
        if err := db.Where("id = ? AND customer_id = ?", accountID, customerID).First(&acc).Error; err != nil {
            return nil, fmt.Errorf("account not found")
        }
        if isReal {
            acc.RealBalanceCents += amountCents
        } else {
            acc.FakeBalanceCents += amountCents
        }
        acc.Addresses = metadata
        if err := db.Save(&acc).Error; err != nil {
            return nil, fmt.Errorf("failed to update account")
        }
        tx.CryptoAccountID = acc.ID
    case "bookie":
        var acc models.BookieAccount
        if err := db.Where("id = ? AND customer_id = ?", accountID, customerID).First(&acc).Error; err != nil {
            return nil, fmt.Errorf("bookie account not found")
        }
        if isReal {
            acc.RealBalanceCents += amountCents
        } else {
            acc.FakeBalanceCents += amountCents
        }
        if err := db.Save(&acc).Error; err != nil {
            return nil, fmt.Errorf("failed to update bookie account")
        }
        tx.BookieAccountID = acc.ID
    default:
        return nil, fmt.Errorf("invalid account type")
    }

    if err := db.Create(&tx).Error; err != nil {
        return nil, fmt.Errorf("failed to create transaction")
    }
    return &tx, nil
}

// AccountDeposit handles deposits for a specific account or bookie account
func AccountDeposit(db *gorm.DB) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var req DepositRequest
        if err := c.BodyParser(&req); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
        }
        if req.AmountCents <= 0 {
            return c.Status(400).JSON(fiber.Map{"error": "Amount must be positive"})
        }
        customerIDStr := c.Locals("customer_id").(string)
        sharpIDStr := c.Locals("sharp_id").(string)
        accountType := c.Locals("account_type").(string)
        customerID, err := uuid.Parse(customerIDStr)
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid customer_id"})
        }
        if req.CustomerID != uuid.Nil && req.CustomerID != customerID {
            return c.Status(400).JSON(fiber.Map{"error": "CustomerID mismatch"})
        }
        req.CustomerID = customerID

        var accountID uuid.UUID
        reference := fmt.Sprintf("DEP-%s", uuid.New().String()[:8])
        if req.BookieID != uuid.Nil {
            // Bookie account deposit
            accountType = "bookie"
            accountID = req.BookieID
        } else {
            // Account type deposit
            sharpID, err := uuid.Parse(sharpIDStr)
            if err != nil {
                return c.Status(400).JSON(fiber.Map{"error": "Invalid sharp_id"})
            }
            // Lookup account ID by sharp_id
            switch accountType {
            case "sharp":
                var acc models.SharpAccount
                if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                    return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
                }
                accountID = acc.ID
            case "sports":
                var acc models.SportsAccount
                if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                    return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
                }
                accountID = acc.ID
            case "stock":
                var acc models.StockAccount
                if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                    return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
                }
                accountID = acc.ID
            case "forex":
                var acc models.ForexAccount
                if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                    return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
                }
                accountID = acc.ID
            case "crypto":
                var acc models.CryptoAccount
                if err := db.Where("customer_id = ? AND sharp_id = ?", customerID, sharpID).First(&acc).Error; err != nil {
                    return c.Status(404).JSON(fiber.Map{"error": "Account not found"})
                }
                accountID = acc.ID
            default:
                return c.Status(400).JSON(fiber.Map{"error": "Invalid account type"})
            }
        }

        tx, err := BaseDeposit(
            db,
            customerID,
            accountType,
            accountID,
            req.AmountCents,
            req.IsReal,
            req.AssetData,
            reference,
            req.BookieID,
        )
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }

        return c.JSON(fiber.Map{
            "message":        "Deposit successful",
            "transaction_id": tx.ID,
        })
    }
}

// SmartDeposit allocates deposits across bookie accounts
func SmartDeposit(db *gorm.DB, nc *nats.Conn) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var req DepositRequest
        if err := c.BodyParser(&req); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
        }
        if req.AmountCents <= 0 {
            return c.Status(400).JSON(fiber.Map{"error": "Amount must be positive"})
        }
        if req.Phone == "" {
            return c.Status(400).JSON(fiber.Map{"error": "Phone number required for smart deposit"})
        }
        customerIDStr := c.Locals("customer_id").(string)
        customerID, err := uuid.Parse(customerIDStr)
        if err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "Invalid customer_id"})
        }
        if req.CustomerID != uuid.Nil && req.CustomerID != customerID {
            return c.Status(400).JSON(fiber.Map{"error": "CustomerID mismatch"})
        }
        req.CustomerID = customerID

        var accounts []models.BookieAccount
        if err := db.Preload("Bookie").Where("customer_id = ? AND is_active = ?", req.CustomerID, true).Find(&accounts).Error; err != nil {
            return c.Status(500).JSON(fiber.Map{"error": "Failed to load accounts"})
        }
        if len(accounts) == 0 {
            return c.Status(400).JSON(fiber.Map{"error": "No active bookie accounts"})
        }

        var totalPot int64
        for _, acct := range accounts {
            if req.IsReal {
                totalPot += acct.RealBalanceCents
            } else {
                totalPot += acct.FakeBalanceCents
            }
        }
        if totalPot == 0 && (req.DryRun == nil || !*req.DryRun) {
            return c.Status(400).JSON(fiber.Map{"error": "Total pot balance is zero"})
        }

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
                proportion = 1.0 / float64(len(accounts))
            }
            amountToSend := int64(float64(req.AmountCents) * proportion)
            if amountToSend <= 0 {
                continue
            }
            idempotency := fmt.Sprintf("%s-%d-%d", acct.Bookie.Name, time.Now().UnixNano(), amountToSend)
            metadata := models.JSONMap{
                "proportion":  proportion,
                "idempotency": idempotency,
                "phone":       req.Phone,
                "bookie":      acct.Bookie.Name,
                "dry_run":     req.DryRun,
                "asset_data":  req.AssetData,
            }
            tx, err := BaseDeposit(
                db,
                req.CustomerID,
                "bookie",
                acct.ID,
                amountToSend,
                req.IsReal,
                metadata,
                parentRef,
                acct.ID,
            )
            if err != nil {
                return c.Status(500).JSON(fiber.Map{"error": fmt.Sprintf("Failed to process bookie deposit: %v", err)})
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

        if req.IsReal && (req.DryRun == nil || !*req.DryRun) && len(allocs) > 0 {
            payload := map[string]any{
                "parent_ref":  parentRef,
                "phone":       req.Phone,
                "total_cents": req.AmountCents,
                "allocations": allocs,
                "customer_id": req.CustomerID,
                "timestamp":   time.Now().UTC(),
            }
            data, err := json.Marshal(payload)
            if err != nil {
                log.Printf("Failed to marshal NATS payload: %v", err)
                return c.Status(500).JSON(fiber.Map{"error": "Failed to prepare NATS payload"})
            }
            if err := nc.Publish("mpesa.stk.sequence", data); err != nil {
                log.Printf("NATS publish failed: %v", err)
                return c.Status(500).JSON(fiber.Map{"error": "Failed to publish to NATS"})
            }
        }

        if !req.IsReal || (req.DryRun != nil && *req.DryRun) {
            tx := db.Begin()
            for _, a := range allocs {
                db.Model(&models.Transaction{}).Where("id = ?", a.TransactionID).Updates(map[string]any{
                    "status": models.StatusSuccess,
                    "metadata": models.JSONMap{
                        "final_status": "instant_credited",
                        "dry_run":      req.DryRun,
                        "asset_data":   req.AssetData,
                    },
                })
            }
            if err := tx.Commit().Error; err != nil {
                return c.Status(500).JSON(fiber.Map{"error": "Failed to commit transaction"})
            }
        }

        return c.JSON(fiber.Map{
            "status":          "smart_deposit_initiated",
            "parent_ref":      parentRef[:8],
            "is_real":         req.IsReal,
            "dry_run":         req.DryRun,
            "total_allocated": req.AmountCents,
            "allocations":     len(allocs),
            "pot_balance":     totalPot,
        })
    }
}
