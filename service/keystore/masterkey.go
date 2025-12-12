// internal/crypto/masterkey/masterkey.go
// This file is the ONLY place that knows about Vault + master identity + TLS certs
package keystore

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/hashicorp/vault/api"
	"golang.org/x/crypto/curve25519" // X25519
	"golang.org/x/crypto/hkdf"
	"github.com/spf13/viper"
)

const (
	vaultIdentityPath = "secret/data/werikana/master-identity"
	vaultTLSCertsPath = "secret/data/werikana/tls-certs"
)

var (
	// Master Identity (existing)
	AESMasterKey []byte // 32-byte key for encrypting DB fields, etc.
	IdentityPriv [32]byte
	IdentityPub  [32]byte // Public key (derived)
	IdentityPubB64 string // Convenience: base64 version of public key

	// TLS Certs (new: PEM-encoded for easy tls.Config use)
	CACertPEM      string // Base64 PEM of CA cert
	ServerCertPEM  string // Base64 PEM of server cert
	ServerKeyPEM   string // Base64 PEM of server key
	ClientCertPEM  string // Base64 PEM of client cert
	ClientKeyPEM   string // Base64 PEM of client key
)

// Init loads or generates the master identity + TLS certs from Vault once at startup
// Call this exactly once from main.go
func Init() {
	if viper.GetString("GENERATE_MASTER_IDENTITY") == "1" || viper.GetString("GENERATE_TLS_CERTS") == "1" {
		generateAndStoreIdentityAndCertsAndExit()
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

	// Load TLS certs
	loadTLSCertsFromVault()

	log.Println("Master identity and TLS certs loaded from Vault")
	log.Printf(" Identity public key: %s", IdentityPubB64)
	log.Println(" AES master key ready (32 bytes)")
	log.Println(" TLS certs ready (CA, server, client)")
	// Optional: log.Printf(" AES key (hex): %s", hex.EncodeToString(AESMasterKey)) // only in dev!
}

// ---------------------------------------------------------------
// ONE-TIME GENERATION (run only once ever)
func generateAndStoreIdentityAndCertsAndExit() {
	log.Println("GENERATING NEW MASTER IDENTITY + TLS CERTS — THIS MUST RUN ONLY ONCE")
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

	// Store identity (existing)
	_, err = client.Logical().Write(vaultIdentityPath, map[string]interface{}{
		"data": map[string]string{
			"private_key": base64.StdEncoding.EncodeToString(priv),
			"public_key":  base64.StdEncoding.EncodeToString(pub),
			"note":        "werikana master encryption + SecureBus identity – generated once",
		},
	})
	if err != nil {
		log.Fatalf("Failed to write identity to Vault: %v", err)
	}

	// Generate TLS Certs (new)
	caCertPEM, caKeyPEM, err := generateRootCA("Werikana Root CA", 10*time.Hour*24*365) // 10 years
	if err != nil {
		log.Fatalf("Failed to generate root CA: %v", err)
	}
	serverCertPEM, serverKeyPEM, err := generateServerCert(caCertPEM, caKeyPEM, "securebus.sxcntcnqunts.com") // Adjust hostname
	if err != nil {
		log.Fatalf("Failed to generate server cert: %v", err)
	}
	clientCertPEM, clientKeyPEM, err := generateClientCert(caCertPEM, caKeyPEM, "werikana-api-client")
	if err != nil {
		log.Fatalf("Failed to generate client cert: %v", err)
	}

	// Store certs in Vault
	_, err = client.Logical().Write(vaultTLSCertsPath, map[string]interface{}{
		"data": map[string]string{
			"ca_cert_pem":      base64.StdEncoding.EncodeToString([]byte(caCertPEM)),
			"server_cert_pem":  base64.StdEncoding.EncodeToString([]byte(serverCertPEM)),
			"server_key_pem":   base64.StdEncoding.EncodeToString([]byte(serverKeyPEM)),
			"client_cert_pem":  base64.StdEncoding.EncodeToString([]byte(clientCertPEM)),
			"client_key_pem":   base64.StdEncoding.EncodeToString([]byte(clientKeyPEM)),
			"note":             "werikana TLS certs for NATS mTLS – generated once",
		},
	})
	if err != nil {
		log.Fatalf("Failed to write TLS certs to Vault: %v", err)
	}

	fmt.Println("\nMASTER IDENTITY + TLS CERTS GENERATED AND SAVED")
	fmt.Printf("Identity Vault path: %s\n", vaultIdentityPath)
	fmt.Printf("TLS Certs Vault path: %s\n\n", vaultTLSCertsPath)
	fmt.Println("PUBLIC KEY (safe to share / put in ENV):")
	fmt.Println(base64.StdEncoding.EncodeToString(pub))
	fmt.Println("\nYOUR PERMANENT AES MASTER KEY (STORE OFFLINE NOW):")
	fmt.Println(hex.EncodeToString(aesKey))
	fmt.Println("\nTLS CERTS (Base64 PEM – export to files or use directly):")
	fmt.Println("CA Cert PEM:")
	fmt.Println(caCertPEM)
	fmt.Println("\nServer Cert PEM:")
	fmt.Println(serverCertPEM)
	fmt.Println("\nServer Key PEM (keep secret):")
	fmt.Println(serverKeyPEM)
	fmt.Println("\nClient Cert PEM:")
	fmt.Println(clientCertPEM)
	fmt.Println("\nClient Key PEM (keep secret):")
	fmt.Println(clientKeyPEM)
	fmt.Println("\nNow remove GENERATE_MASTER_IDENTITY=1 and GENERATE_TLS_CERTS=1, restart normally.\n")
	os.Exit(0)
}

// ---------------------------------------------------------------
// Cert Generation Helpers (new)
func generateRootCA(subject string, validity time.Duration) (certPEM, keyPEM string, err error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048) // RSA for CA; EC/X25519 for modern
	if err != nil {
		return "", "", err
	}
	caTemplate := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{Organization: []string{subject}},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &priv.PublicKey, priv)
	if err != nil {
		return "", "", err
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes}))
	keyPEM = string(pem.EncodeToMemory(pemBlockForKey(priv)))
	return certPEM, keyPEM, nil
}

func generateServerCert(caCertPEM, caKeyPEM, hostname string) (certPEM, keyPEM string, err error) {
	caCert, err := parseCertPEM(caCertPEM)
	if err != nil {
		return "", "", err
	}
	caKey, err := parseKeyPEM(caKeyPEM)
	if err != nil {
		return "", "", err
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{Organization: []string{"Werikana NATS Server"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * time.Hour * 24 * 365),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:     []string{hostname, "localhost"},
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, &priv.PublicKey, caKey.(*rsa.PrivateKey))
	if err != nil {
		return "", "", err
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}))
	keyPEM = string(pem.EncodeToMemory(pemBlockForKey(priv)))
	return certPEM, keyPEM, nil
}

func generateClientCert(caCertPEM, caKeyPEM, subject string) (certPEM, keyPEM string, err error) {
	caCert, err := parseCertPEM(caCertPEM)
	if err != nil {
		return "", "", err
	}
	caKey, err := parseKeyPEM(caKeyPEM)
	if err != nil {
		return "", "", err
	}
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject:      pkix.Name{Organization: []string{subject}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(10 * time.Hour * 24 * 365),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certBytes, err := x509.CreateCertificate(rand.Reader, &template, caCert, &priv.PublicKey, caKey.(*rsa.PrivateKey))
	if err != nil {
		return "", "", err
	}
	certPEM = string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes}))
	keyPEM = string(pem.EncodeToMemory(pemBlockForKey(priv)))
	return certPEM, keyPEM, nil
}

// Helpers for PEM parsing/encoding
func parseCertPEM(pemStr string) (*x509.Certificate, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to parse cert PEM")
	}
	return x509.ParseCertificate(block.Bytes)
}

func parseKeyPEM(pemStr string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, fmt.Errorf("failed to parse key PEM")
	}
	return parsePKCS1PrivateKey(block.Bytes) // Assume RSA; extend for EC
}

func parsePKCS1PrivateKey(der []byte) (crypto.PrivateKey, error) {
	return x509.ParsePKCS1PrivateKey(der)
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	}
	return nil
}

// ---------------------------------------------------------------
// Load TLS Certs (new)
func loadTLSCertsFromVault() {
	client := mustVaultClient()
	secret, err := client.Logical().Read(vaultTLSCertsPath)
	if err != nil {
		log.Fatalf("Vault TLS read error: %v", err)
	}
	if secret == nil || secret.Data == nil {
		log.Fatalf("TLS certs not found in Vault at %s — run with GENERATE_TLS_CERTS=1 once", vaultTLSCertsPath)
	}
	data := secret.Data["data"].(map[string]interface{})
	caB64, ok := data["ca_cert_pem"].(string)
	if !ok {
		log.Fatal("ca_cert_pem missing in Vault")
	}
	caDecoded, err := base64.StdEncoding.DecodeString(caB64)
	if err != nil {
		log.Fatalf("Failed to decode ca_cert_pem: %v", err)
	}
	CACertPEM = string(caDecoded)

	serverCertB64, ok := data["server_cert_pem"].(string)
	if !ok {
		log.Fatal("server_cert_pem missing in Vault")
	}
	serverCertDecoded, err := base64.StdEncoding.DecodeString(serverCertB64)
	if err != nil {
		log.Fatalf("Failed to decode server_cert_pem: %v", err)
	}
	ServerCertPEM = string(serverCertDecoded)

	serverKeyB64, ok := data["server_key_pem"].(string)
	if !ok {
		log.Fatal("server_key_pem missing in Vault")
	}
	serverKeyDecoded, err := base64.StdEncoding.DecodeString(serverKeyB64)
	if err != nil {
		log.Fatalf("Failed to decode server_key_pem: %v", err)
	}
	ServerKeyPEM = string(serverKeyDecoded)

	clientCertB64, ok := data["client_cert_pem"].(string)
	if !ok {
		log.Fatal("client_cert_pem missing in Vault")
	}
	clientCertDecoded, err := base64.StdEncoding.DecodeString(clientCertB64)
	if err != nil {
		log.Fatalf("Failed to decode client_cert_pem: %v", err)
	}
	ClientCertPEM = string(clientCertDecoded)

	clientKeyB64, ok := data["client_key_pem"].(string)
	if !ok {
		log.Fatal("client_key_pem missing in Vault")
	}
	clientKeyDecoded, err := base64.StdEncoding.DecodeString(clientKeyB64)
	if err != nil {
		log.Fatalf("Failed to decode client_key_pem: %v", err)
	}
	ClientKeyPEM = string(clientKeyDecoded)
}

// ---------------------------------------------------------------
// Existing functions (unchanged)
func loadX25519PrivateFromVault() []byte {
	client := mustVaultClient()
	secret, err := client.Logical().Read(vaultIdentityPath)
	if err != nil {
		log.Fatalf("Vault read error: %v", err)
	}
	if secret == nil || secret.Data == nil {
		log.Fatalf("Master identity not found in Vault at %s — run with GENERATE_MASTER_IDENTITY=1 once", vaultIdentityPath)
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

func deriveAESMasterKey(x25519Priv []byte) []byte {
	kdf := hkdf.New(sha256.New, x25519Priv, nil, []byte("werikana-aes-256-master-key-v1"))
	key := make([]byte, 32)
	if _, err := io.ReadFull(kdf, key); err != nil {
		log.Fatalf("HKDF failed: %v", err)
	}
	return key
}

func mustVaultClient() *api.Client {
	addr := viper.GetString("VAULT_ADDR")
	token := viper.GetString("VAULT_TOKEN")
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
