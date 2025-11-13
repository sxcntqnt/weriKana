package securewithdrawal

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack/v5"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Placeholder types (replace with your actual definitions)
type Transaction struct {
	gorm.Model
	ID            uuid.UUID
	Status        string
	Reference     string
	IdempotencyKey string
	Metadata      JSONMap
}

type WithdrawalItem struct {
	BookieAccountID uuid.UUID
	AmountCents     int64
	EncryptedKey    []byte
}

type SecureWithdrawalEnvelope struct {
	TransactionID uuid.UUID
	Items         []WithdrawalItem
	OTPNonce      []byte
	OTPEncrypted  []byte
	Timestamp     int64
	Idempotency   string
	Signature     []byte
}

type JSONMap map[string]interface{}

// Placeholder constants (replace with your actual definitions)
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
)

// Placeholder GenerateOTP function (replace with your actual implementation)
func GenerateOTP() (string, error) {
	return "123456", nil
}

// SendSecureWithdrawalViaNATS is the **single public function** you asked for.
func (t *Transaction) SendSecureWithdrawalViaNATS(
	db *gorm.DB,
	nc *nats.Conn,
	subject string,
	crypto *CryptoEngine,
	items []WithdrawalItem,
) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// 1. Row lock + idempotency guard
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(t, t.ID).Error; err != nil {
			return err
		}
		if t.Status == StatusPending || t.Status == StatusProcessing {
			return nil // already sent
		}
		// 2. OTP + encryption
		otp, err := GenerateOTP()
		if err != nil {
			return err
		}
		nonce, ciphertext, err := crypto.EncryptOTP(otp)
		if err != nil {
			return err
		}
		// 3. Envelope
		idempotency := uuid.New().String()
		envelope := SecureWithdrawalEnvelope{
			TransactionID: t.ID,
			Items:         items,
			OTPNonce:      nonce,
			OTPEncrypted:  ciphertext,
			Timestamp:     time.Now().Unix(),
			Idempotency:   idempotency,
		}
		// 4. Sign (msgpack without signature field)
		tmp, _ := msgpack.Marshal(envelope)
		signable := tmp[:len(tmp)-len(envelope.Signature)] // strip placeholder
		envelope.Signature = crypto.Sign(signable)
		// 5. Final msgpack payload
		payload, err := msgpack.Marshal(envelope)
		if err != nil {
			return err
		}
		// 6. Publish via JetStream (dedup + ack)
		js, err := nc.JetStream()
		if err != nil {
			return err
		}
		ack, err := js.Publish(subject, payload,
			nats.MsgId(idempotency),
			nats.AckWait(30*time.Second),
		)
		if err != nil {
			return fmt.Errorf("nats publish: %w", err)
		}
		// 7. Persist audit trail
		hash := sha256.Sum256(payload)
		t.IdempotencyKey = idempotency
		t.Status = StatusPending
		t.Reference = fmt.Sprintf("%s@%d", ack.Stream, ack.Sequence)
		t.Metadata = JSONMap{
			"nats_stream":    ack.Stream,
			"nats_seq":       ack.Sequence,
			"otp_nonce":      base64.StdEncoding.EncodeToString(nonce),
			"otp_encrypted":  base64.StdEncoding.EncodeToString(ciphertext),
			"signature":      base64.StdEncoding.EncodeToString(envelope.Signature),
			"msgpack_sha256": fmt.Sprintf("%x", hash),
			"sent_at":        time.Now().Format(time.RFC3339),
			"otp_plain":      otp, // optional – remove in prod if you don't want it stored
		}
		return tx.Save(t).Error
	})
}
