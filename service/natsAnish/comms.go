package natsAnish

import (
    "fmt"
    "log"
    "os"

    "github.com/nats-io/nats.go"
    "github.com/vmihailenco/msgpack/v5" // Import msgpack package for encoding/decoding
    "github.com/google/uuid"            // Import uuid package
)

var NC *nats.Conn

// UUID wrapper type to implement Marshaler/Unmarshaler interfaces
type UUID uuid.UUID

// MarshalMsgpack implements msgpack.Marshaler interface for UUID
func (u UUID) MarshalMsgpack() ([]byte, error) {
    return uuid.UUID(u).MarshalBinary()
}

// UnmarshalMsgpack implements msgpack.Unmarshaler interface for UUID
func (u *UUID) UnmarshalMsgpack(data []byte) error {
    return (*uuid.UUID)(u).UnmarshalBinary(data)
}

// Initialize NATS connection and register UUID type with msgpack
func init() {
    url := os.Getenv("NATS_URL")
    if url == "" {
        url = "nats://localhost:4222"
    }

    var err error
    NC, err = nats.Connect(url)
    if err != nil {
        log.Fatal("NATS connect failed:", err)
    }

    // Register the UUID type with msgpack to use custom serialization
    msgpack.RegisterExt(0, (*UUID)(nil))
}

// Publish sends data to a NATS subject
func Publish(subject string, data []byte) error {
    return NC.Publish(subject, data)
}

