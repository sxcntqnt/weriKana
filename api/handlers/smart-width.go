package handlers

import (
    "fmt"
    "github.com/gofiber/fiber/v2"
    "github.com/google/uuid"
    "weriKana/models"
    "weriKana/service/dd_rr"
    "gorm.io/gorm"
    "log"
)

func SmartWithdraw(db *gorm.DB, keyStore *KeyStore, crypto *securewithdrawal.CryptoEngine, nc *nats.Conn) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var req SmartWithdrawRequest
        if err := c.BodyParser(&req); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
        }

        // Step 1: Validate the withdrawal amount
        if req.Amount <= 0 {
            return c.Status(400).JSON(fiber.Map{"error": "amount must be > 0"})
        }

        // Step 2: Verify OTP
        if !otp.Verify(req.CustomerID, req.OTP) {
            return c.Status(401).JSON(fiber.Map{"error": "invalid or expired OTP"})
        }

        // Step 3: Verify signature using keyStore
        payload := fmt.Sprintf("%s:%s:%d", req.CustomerID, req.OTP, req.Amount)
        if !keyStore.Verify(req.CustomerID, payload, req.Signature) {
            return c.Status(401).JSON(fiber.Map{"error": "invalid signature"})
        }

        // Step 4: Delegate allocation logic to `securewithdrawal` service
        allocs, totalPot, err := securewithdrawal.CalculateAllocations(db, req.CustomerID, req.Amount, req.IsReal)
        if err != nil {
            return c.Status(500).JSON(fiber.Map{"error": err.Error()})
        }

        if len(allocs) == 0 {
            return c.Status(400).JSON(fiber.Map{"error": "no active bookie accounts or insufficient balance"})
        }

        // Step 5: Prepare withdrawal items and transaction
        items := make([]securewithdrawal.WithdrawalItem, len(allocs))
        for i, a := range allocs {
            items[i] = securewithdrawal.WithdrawalItem{
                BookieID:    a.BookieID,
                AmountCents: a.AmountToSend,
            }
        }

        // Step 6: Send secure withdrawal via NATS
        transaction := &securewithdrawal.Transaction{ID: uuid.New()}
        if err := transaction.SendSecureWithdrawalViaNATS(db, nc, "withdraw.secure", crypto, items); err != nil {
            log.Printf("Failed to send secure withdrawal to NATS: %v", err)
            return c.Status(500).JSON(fiber.Map{"error": "failed to send secure withdrawal"})
        }

        // Step 7: Return a successful response
        return c.JSON(fiber.Map{
            "status":       "smart_withdraw_initiated",
            "parent_ref":   transaction.ID.String()[:8], // Shortened for readability
            "total_cents":  req.Amount,
            "is_real":      req.IsReal,
            "bookies":      len(allocs),
            "pot_balance":  totalPot,
        })
    }
}


func ProcessWithdrawal(db *gorm.DB, nc *nats.Conn, request map[string]interface{}) error {
    // Extract necessary fields from the NATS message (request)
    customerID := request["customer_id"].(string)
    amount := int64(request["total_cents"].(float64))
    isReal := request["is_real"].(bool)
    otp := request["otp"].(string)
    signature := []byte(request["signature"].(string))

    // Create SmartWithdrawRequest to pass to business logic in securewithdrawal
    smartWithdrawRequest := dd_rr.SmartWithdrawRequest{
        CustomerID: customerID,
        Amount:     amount,
        IsReal:     isReal,
        OTP:        otp,
        Signature:  signature,
    }

    // Call SmartWithdraw logic from dd_rr/foreman.go to verify and process withdrawal
    if err := dd_rr.SmartWithdraw(db, smartWithdrawRequest); err != nil {
        return fmt.Errorf("failed to process smart withdrawal: %w", err)
    }

    // Send withdrawal information to NATS via natsAnish
    err := natsAnish.PublishWithdrawalToNATS(db, nc, smartWithdrawRequest)
    if err != nil {
        return fmt.Errorf("failed to send withdrawal to NATS: %w", err)
    }

    return nil
}
