// services/otp/otp.go
package otp

import (
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	cache = make(map[uuid.UUID]string)
	mu    sync.RWMutex
)

func Send(customerID uuid.UUID) string {
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))

	mu.Lock()
	cache[customerID] = otp
	mu.Unlock()

	go func() {
		time.Sleep(5 * time.Minute)
		mu.Lock()
		delete(cache, customerID)
		mu.Unlock()
	}()

	// TODO: Send via SMS (NATS)
	natsclient.Publish("sms.send", []byte(fmt.Sprintf(`{"to":"user_phone","msg":"BankRoll OTP: %s"}`, otp)))
	return otp
}

func Verify(customerID uuid.UUID, otp string) bool {
	mu.RLock()
	stored := cache[customerID]
	mu.RUnlock()
	return stored == otp
}
