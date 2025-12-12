// internal/bankd/utils.go
package bankd

import (
	"fmt"
	"os"
	"strings"
	"strconv"
	"log"

	gossh "golang.org/x/crypto/ssh"
)

func parseAmount(input string) (float64, error) {
    // Remove commas, spaces, and normal separators
    cleaned := strings.ReplaceAll(input, ",", "")
    cleaned = strings.TrimSpace(cleaned)

    if cleaned == "" {
        return 0, fmt.Errorf("empty amount")
    }

    amount, err := strconv.ParseFloat(cleaned, 64)
    if err != nil || amount <= 0 {
        return 0, fmt.Errorf("invalid or non-positive amount")
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
