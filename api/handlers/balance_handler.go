// handlers/balance_handler.go
package handlers

import (
	"errors"
	"time" // ← FIXED: missing import

	"weriKana/internal/bank"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BalanceHandler exposes balance endpoints
type BalanceHandler struct {
	engine *bank.BalanceEngine
}

func NewBalanceHandler(engine *bank.BalanceEngine) *BalanceHandler {
	return &BalanceHandler{engine: engine}
}

// GetMyBalance — Customer endpoint: GET /api/v1/balance/me
func (h *BalanceHandler) GetMyBalance(c *fiber.Ctx) error {
	customerID, ok := c.Locals("customer_id").(uuid.UUID)
	if !ok || customerID == uuid.Nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "unauthorized: missing customer context",
		})
	}

	bal, err := h.engine.GetCustomerBalance(c.Context(), customerID)
	if err != nil {
		// FIXED: proper error inspection instead of fragile string comparison
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"error":   "customer not found",
			})
		}

		// Any other error = internal
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to retrieve balance",
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    bal,
	})
}

// AdminGetAllBalances — Admin only: GET /api/v1/admin/balances?limit=100&cursor=...
func (h *BalanceHandler) AdminGetAllBalances(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 100)
	if limit < 1 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	cursor := c.Query("cursor")

	summaries, nextCursor, err := h.engine.GetAllBalancesPaginated(c.Context(), limit, cursor)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "failed to load customer balances",
		})
	}

	resp := fiber.Map{
		"success":  true,
		"data":     summaries,
		"count":    len(summaries),
		"has_more": nextCursor != "",
	}

	if nextCursor != "" {
		resp["next_cursor"] = nextCursor
	}

	return c.JSON(resp)
}

// GetBalanceEngineMetrics — Monitoring endpoint (Prometheus/Datadog ready)
func (h *BalanceHandler) GetBalanceEngineMetrics(c *fiber.Ctx) error {
	metrics := h.engine.Metrics()

	return c.JSON(fiber.Map{
		"success":   true,
		"metrics":   metrics,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}
