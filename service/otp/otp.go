package otp

import (
    "fmt"
    "crypto/rand"
    "sync"
    "time"

    "github.com/google/uuid"
    "weriKana/service/natsAnish" // Import the natsAnish package
)

var (
    cache = make(map[uuid.UUID]string)
    mu    sync.RWMutex
)

// GenerateOTP returns a 6-digit string (000000–999999)
func GenerateOTP() (string, error) {
        b := make([]byte, 3)
        if _, err := rand.Read(b); err != nil {
                return "", err
        }
        n := (uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])) % 1_000_000
        return fmt.Sprintf("%06d", n), nil
}

func Send(customerID uuid.UUID) string {
    otp := fmt.Sprintf("%06d", GenerateOTP())

    mu.Lock()
    cache[customerID] = otp
    mu.Unlock()

    go func() {
        time.Sleep(5 * time.Minute)
        mu.Lock()
        delete(cache, customerID)
        mu.Unlock()
    }()

    // Use the Publish function from natsAnish
    err := natsAnish.Publish("sms.send", []byte(fmt.Sprintf(`{"to":"user_phone","msg":"BankRoll OTP: %s"}`, otp)))
    if err != nil {
        fmt.Println("Error publishing to NATS:", err)
    }

    return otp
}

func Verify(customerID uuid.UUID, otp string) bool {
    mu.RLock()
    stored := cache[customerID]
    mu.RUnlock()
    return stored == otp
}

