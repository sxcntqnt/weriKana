// service/otp/otp.go
package otp

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"weriKana/service/natsAnish"
)

type Service struct {
	cache map[uuid.UUID]string
	mu    sync.RWMutex
}

func New() *Service {
	return &Service{
		cache: make(map[uuid.UUID]string),
	}
}

func generateOTP() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := (uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}

// Send generates OTP, caches it for 5 min, and publishes via NATS
func (s *Service) Send(customerID uuid.UUID) string {
	otp, err := generateOTP()
	if err != nil {
		fmt.Printf("OTP generation failed: %v\n", err)
		return ""
	}

	s.mu.Lock()
	s.cache[customerID] = otp
	s.mu.Unlock()

	// Auto-expire after 5 minutes
	go func() {
		time.Sleep(5 * time.Minute)
		s.mu.Lock()
		delete(s.cache, customerID)
		s.mu.Unlock()
	}()

	// Send via NATS → your SMS microservice
	payload := fmt.Sprintf(`{"customer_id":"%s","msg":"BankRoll OTP: %s - Valid for 5 mins"}`, customerID, otp)
	if err := natsAnish.Publish("sms.send", []byte(payload)); err != nil {
		fmt.Printf("Failed to publish OTP to NATS: %v\n", err)
	}

	return otp // only in dev! remove in prod
}

// Verify checks if OTP matches
func (s *Service) Verify(customerID uuid.UUID, code string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stored, exists := s.cache[customerID]
	return exists && stored == code
}

// Invalidate removes OTP after successful use (one-time OTP)
func (s *Service) Invalidate(customerID uuid.UUID) {
	s.mu.Lock()
	delete(s.cache, customerID)
	s.mu.Unlock()
}
