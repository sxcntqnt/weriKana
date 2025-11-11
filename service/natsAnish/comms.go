// services/natsclient/client.go
package natsclient

import (
	"log"
	"os"

	"github.com/nats-io/nats.go"
)

var NC *nats.Conn

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
}

func Publish(subject string, data []byte) error {
	return NC.Publish(subject, data)
}
