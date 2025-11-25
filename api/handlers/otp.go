// api/handlers/otp.go
package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"weriKana/models"
	"weriKana/service/otp"
	"gorm.io/gorm"
)

// RequestWithdrawOTP — generates and sends OTP via NATS → SMS
func RequestWithdrawOTP(db *gorm.DB, otpSvc *otp.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			CustomerID uuid.UUID `json:"customer_id"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid JSON",
			})
		}

		if req.CustomerID == uuid.Nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "customer_id is required",
			})
		}

		// Verify customer exists
		var customer models.Sharp
		if err := db.Select("id").First(&customer, "id = ?", req.CustomerID).Error; err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Customer not found",
			})
		}

		// Generate and send OTP
		otpCode := otpSvc.Send(customer.ID)

		if otpCode == "" {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "Failed to generate OTP",
			})
		}

		return c.JSON(fiber.Map{
			"status": "otp_sent",
			"hint":   "Check your phone for SMS",
			// Remove this in production!
			// "debug_otp": otpCode,
		})
	}
}

// VerifyOTP — checks if provided OTP is correct
func VerifyOTP(otpSvc *otp.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req struct {
			CustomerID uuid.UUID `json:"customer_id"`
			OTP        string    `json:"otp"`
		}

		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid JSON"})
		}

		if req.CustomerID == uuid.Nil || req.OTP == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id and otp required"})
		}

		if otpSvc.Verify(req.CustomerID, req.OTP) {
			otpSvc.Invalidate(req.CustomerID) // one-time use
			return c.JSON(fiber.Map{"valid": true})
		}

		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"valid": false,
			"error": "Invalid or expired OTP",
		})
	}
}
