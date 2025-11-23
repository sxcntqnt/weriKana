// internal/bankd/utils.go
package bankd

import (
	"fmt"
	"os"
	"strings"
	"log"

	gossh "golang.org/x/crypto/ssh"
)

func parseAmount(input string) (float64, error) {
	var amount float64
	_, err := fmt.Sscanf(input, "%f", &amount)
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("invalid or negative amount")
	}
	return amount, nil
}

func partnerCustomerID(partnerID string) string {
	return "cust_" + strings.TrimPrefix(partnerID, "partner_")
}

func loadHostKey() gossh.Signer {
	key, err := gossh.ParsePrivateKey([]byte(os.Getenv("SSH_HOST_KEY")))
	if err != nil || key == nil {
		log.Fatal("SSH host key not provided or invalid")
	}
	return key
}
