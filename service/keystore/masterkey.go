// internal/crypto/masterkey/masterkey.go
// This file is the ONLY place that knows about Vault + master identity
package keystore

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/hashicorp/vault/api"
	"golang.org/x/crypto/curve25519" // X25519
	"golang.org/x/crypto/hkdf"
)

const vaultPath = "secret/data/werikana/master-identity"

var (
	// Exported so other packages can use it directly
	AESMasterKey   []byte // 32-byte key for encrypting DB fields, etc.
	IdentityPriv   [32]byte
	// X25519 private key → used as your SecureBus identity
	IdentityPub    [32]byte // Public key (derived)
	IdentityPubB64 string   // Convenience: base64 version of public key
)

// Init loads or generates the master identity from Vault once at startup
// Call this exactly once from main.go
func Init() {
	if os.Getenv("GENERATE_MASTER_IDENTITY") == "1" {
		generateAndStoreIdentityAndExit()
	}
	priv := loadX25519PrivateFromVault()
	copy(IdentityPriv[:], priv)
	// Derive public key
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		log.Fatalf("Failed to compute public key: %v", err)
	}
	copy(IdentityPub[:], pub)
	IdentityPubB64 = base64.StdEncoding.EncodeToString(pub)
	// Derive the permanent AES-256 master key (same every time)
	AESMasterKey = deriveAESMasterKey(priv)
	log.Println("Master identity loaded from Vault")
	log.Printf(" Identity public key: %s", IdentityPubB64)
	log.Println(" AES master key ready (32 bytes)")
	// Optional: log.Printf(" AES key (hex): %s", hex.EncodeToString(AESMasterKey)) // only in dev!
}

// ---------------------------------------------------------------
// ONE-TIME GENERATION (run only once ever)
func generateAndStoreIdentityAndExit() {
	log.Println("GENERATING NEW MASTER IDENTITY — THIS MUST RUN ONLY ONCE")
	priv := make([]byte, 32)
	if _, err := rand.Read(priv); err != nil {
		log.Fatalf("rand.Read failed: %v", err)
	}
	// Clamp private key for X25519
	priv[0] &= 248
	priv[31] &= 127
	priv[31] |= 64
	pub, err := curve25519.X25519(priv, curve25519.Basepoint)
	if err != nil {
		log.Fatalf("X25519 failed: %v", err)
	}
	aesKey := deriveAESMasterKey(priv)
	client := mustVaultClient()
	_, err = client.Logical().Write(vaultPath, map[string]interface{}{
		"data": map[string]string{
			"private_key": base64.StdEncoding.EncodeToString(priv),
			"public_key":  base64.StdEncoding.EncodeToString(pub),
			"note":        "werikana master encryption + SecureBus identity – generated once",
		},
	})
	if err != nil {
		log.Fatalf("Failed to write to Vault: %v", err)
	}
	fmt.Println("\nMASTER IDENTITY GENERATED AND SAVED")
	fmt.Printf("Vault path: %s\n\n", vaultPath)
	fmt.Println("PUBLIC KEY (safe to share / put in ENV):")
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	fmt.Println("\nYOUR PERMANENT AES MASTER KEY (STORE OFFLINE NOW):")
	fmt.Println(hex.EncodeToString(aesKey))
	fmt.Println("\nNow remove GENERATE_MASTER_IDENTITY=1 and restart normally.\n")
	os.Exit(0)
}

// ---------------------------------------------------------------
// Normal load path
func loadX25519PrivateFromVault() []byte {
	client := mustVaultClient()
	secret, err := client.Logical().Read(vaultPath)
	if err != nil {
		log.Fatalf("Vault read error: %v", err)
	}
	if secret == nil || secret.Data == nil {
		log.Fatalf("Master identity not found in Vault at %s — run with GENERATE_MASTER_IDENTITY=1 once", vaultPath)
	}
	data := secret.Data["data"].(map[string]interface{})
	privB64, ok := data["private_key"].(string)
	if !ok || privB64 == "" {
		log.Fatal("private_key missing or corrupted in Vault")
	}
	priv, err := base64.StdEncoding.DecodeString(privB64)
	if err != nil || len(priv) != 32 {
		log.Fatal("Invalid private_key in Vault")
	}
	return priv
}

// HKDF-SHA256 derivation of the 32-byte AES key
func deriveAESMasterKey(x25519Priv []byte) []byte {
	kdf := hkdf.New(sha256.New, x25519Priv, nil, []byte("werikana-aes-256-master-key-v1"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(kdf, key); err != nil {
		log.Fatalf("HKDF failed: %v", err)
	}
	return key
}

func mustVaultClient() *api.Client {
	addr := os.Getenv("VAULT_ADDR")
	token := os.Getenv("VAULT_TOKEN")
	if addr == "" || token == "" {
		log.Fatal("VAULT_ADDR and VAULT_TOKEN must be set")
	}
	client, err := api.NewClient(&api.Config{Address: addr})
	if err != nil {
		log.Fatalf("Vault client error: %v", err)
	}
	client.SetToken(token)
	return client
}
