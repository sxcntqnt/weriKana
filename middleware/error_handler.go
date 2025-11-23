// internal/middleware/error_handler.go
// Custom ErrorHandler for Fiber — Graceful, non-crashing error handling.

package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

// ErrorHandler handles errors in Fiber without crashing the app.
// Logs the error with context (if available) and sends a JSON response.
// Returns the error to allow Fiber's default behavior if needed.
func ErrorHandler(c *fiber.Ctx, err error) error {
    // Log the error with request context
    fields := logrus.Fields{
        "method":     c.Method(),
        "path":       c.Path(),
        "ip":         c.IP(),
        "user_agent": c.Get("User-Agent"),
    }
    if userID := c.Locals("user_id"); userID != nil {
        fields["user_id"] = userID
    }
    logrus.WithError(err).WithFields(fields).Error("API error")

    // Default response
    status := fiber.StatusInternalServerError
    code := "INTERNAL_ERROR"
    message := "An unexpected error occurred. Please try again later."

    // Handle Fiber errors
    if fiberErr, ok := err.(*fiber.Error); ok {
        status = fiberErr.Code
        code = "VALIDATION_ERROR"
        message = fiberErr.Message
        if status == fiber.StatusUnauthorized {
            code = "UNAUTHORIZED"
            message = "Authentication required."
        }
    }

    c.Status(status).JSON(fiber.Map{
        "success": false,
        "error": fiber.Map{
            "code":    code,
            "message": message,
        },
    })

    return nil
}
	
