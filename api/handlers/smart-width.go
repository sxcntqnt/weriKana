// api/handlers/smart-width.go
package handlers

import (
	"bytes"
	"encoding/json"
        "io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"weriKana/service/dd_rr"
	"weriKana/service/keystore"
	"weriKana/service/otp"
)

func SmartWithdraw(db *gorm.DB, keyStore *keystore.KeyStore, nc any) fiber.Handler {
	// This reuses your EXACT dd_rr handler — perfect reuse, zero duplication
	realHandler := dd_rr.SmartWithdraw(db, *keyStore, otp.GetService())

	return func(c *fiber.Ctx) error {
		customerID := c.Locals("user_id").(uuid.UUID)

		var req struct {
			OTP       string `json:"otp"`
			Amount    int64  `json:"amount"`
			Signature []byte `json:"signature"`
			IsReal    bool   `json:"is_real"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "invalid json"})
		}

		if req.Amount <= 0 {
			return c.Status(400).JSON(fiber.Map{"error": "amount must be > 0"})
		}

		// Build request exactly as your dd_rr handler expects
		ddrrReq := dd_rr.SmartWithdrawRequest{
			CustomerID: customerID.String(),
			OTP:        req.OTP,
			Amount:     req.Amount,
			Signature:  req.Signature,
			IsReal:     req.IsReal,
		}

		body, _ := json.Marshal(ddrrReq)

		// Call your real handler via httptest — clean, fast, perfect
		rec := httptest.NewRecorder()
		httpReq := httptest.NewRequest("POST", "/smart-withdraw", bytes.NewReader(body))
		httpReq.Header.Set("Content-Type", "application/json")

		realHandler.ServeHTTP(rec, httpReq)
		resp := rec.Result()
		defer resp.Body.Close()

		// Forward error responses directly
		if resp.StatusCode != http.StatusAccepted {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return c.Status(resp.StatusCode).Send(bodyBytes)
		}

		// Parse success response
		var result struct {
			Status     string `json:"status"`
			ParentRef  string `json:"parent_ref"`
			TotalCents int64  `json:"total_cents"`
			IsReal     bool   `json:"is_real"`
			Bookies    int    `json:"bookies"`
			PotBalance int64  `json:"pot_balance"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return c.Status(500).JSON(fiber.Map{"error": "internal error"})
		}

		return c.JSON(fiber.Map{
			"status":         "smart_withdraw_initiated",
			"parent_ref":     result.ParentRef,
			"total_cents":    result.TotalCents,
			"is_real":        result.IsReal,
			"bookies":        result.Bookies,
			"pot_balance":    result.PotBalance,
			"timestamp":      time.Now().UTC(),
		})
	}
}
