// internal/bankd/handler.go
package bankd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/gliderlabs/ssh"
)

const (
	envKey contextKey = "env"
)

// ------------------------------------------------------------
// MAIN SSH SESSION HANDLER
// ------------------------------------------------------------
func SSHHandler(s ssh.Session) {
	// PTY check and ANSI-aware output
	hasPTY, _ := setupPTY(s)
	term := NewTerminalWriter(s)
	if !hasPTY {
		io.WriteString(term, "Error: PTY not allocated\n")
		s.Exit(1)
		return
	}

	username := s.User()
	if username == "" {
		io.WriteString(term, "Error: No username provided\n")
		s.Exit(1)
		return
	}

	log.Printf("Starting CLI session for %s", username)
	term.Write([]byte("Welcome to the CLI!\n"))

	// Attach session into context (used by PasswordHandler to read env vars)
	s.Context().SetValue(sessionKey, s)
	ctx := s.Context()

	// partner_id must be set during authentication
	partnerID, ok := ctx.Value(partnerIDKey).(string)
	if !ok || partnerID == "" {
		fmt.Fprintln(term, "Error: Authentication failed or partner ID missing.")
		s.Exit(1)
		return
	}

	// Retrieve environment
	env := loadEnv(ctx)
	fmt.Fprintf(term, "🚀 Ziklag Bank Ethos — Partner: %s 🚀\n", partnerID)
	fmt.Fprint(term, "Open code. Open scrutiny. Strong foundations built in the open.\n")
	fmt.Fprint(term, " Karibu kwenye himaya ya wasimamizi wa data.\n")
	fmt.Fprint(term, "Designed for builders. Shaped by precision.\n")
	fmt.Fprint(term, "Driven by open-source collaboration. Tunainuka pamoja.\n")
	fmt.Fprint(term, "Built for Africa’s pace: instant updates, smart workflows, and adaptive safeguards.\n")
	fmt.Fprint(term, "Africa’s sharpest edge: Real-time balances, resilient systems, and battle-tested engineering.\n")
	fmt.Fprint(term, "Open-source spirit. Continental ambition. Let's build the future brick by brick.\n")
	fmt.Fprint(term, "Ziklag once burned, but its warriors rose sharper than before.\n")
	fmt.Fprint(term, "That same resilience fuels our infrastructure: swift, alert, and forged in scrutiny.\n")
	fmt.Fprint(term, "Karibu katika lango la waendelezaji—where hustle meets divine tenacity.\n\n")
	fmt.Fprint(term, "Commands: balance, account, deposit, withdraw, assets, profile, help, exit\n\n")

	fmt.Fprintln(term, "Environment:")
	for k, v := range env {
		fmt.Fprintf(term, "%s=%s\n", k, v)
	}
	fmt.Fprintln(term)

	// ------------------------------
	// CLI loop
	// ------------------------------
	for {
		fmt.Fprint(term, "Ziklag> ")
		line, err := readLine(s)
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Fprintf(term, "read error: %v\n", err)
			continue
		}

		parts := strings.Fields(strings.TrimSpace(line))
		if len(parts) == 0 {
			continue
		}

		switch parts[0] {
		case "balance":
			handleBalance(s, partnerID)
		case "account":
			handleAccount(s, partnerID)
		case "deposit":
			handleDeposit(s, partnerID, parts)
		case "withdraw":
			handleWithdraw(s, partnerID, parts)
		case "assets":
			handleAssets(s, partnerID)
		case "profile":
			handleProfile(s, partnerID)
		case "help":
			showHelp(s)
		case "exit", "quit":
			fmt.Fprintln(term, "Goodbye!")
			s.Exit(0)
			return
		default:
			fmt.Fprintf(term, "Unknown command: %s\n", parts[0])
		}
	}

	// Cleanup
	s.Close()
	log.Printf("CLI session ended for %s", username)
}

// ------------------------------------------------------------
// ENVIRONMENT HANDLING
// ------------------------------------------------------------
func loadEnv(ctx ssh.Context) map[string]string {
	env, ok := ctx.Value(envKey).(map[string]string)
	if ok {
		return env
	}
	env = make(map[string]string)
	ctx.SetValue(envKey, env)
	sess, ok := ctx.Value(sessionKey).(ssh.Session)
	if ok {
		for _, kv := range sess.Environ() {
			parts := strings.SplitN(kv, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}
	return env
}

// ------------------------------------------------------------
// SUBSYSTEM: env (optional JSON setter)
// ------------------------------------------------------------
func EnvSubsystemHandler(s ssh.Session) {
	ctx := s.Context()
	env := loadEnv(ctx)
	var payload struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	decoder := json.NewDecoder(s)
	if err := decoder.Decode(&payload); err != nil {
		io.WriteString(s, "invalid json\n")
		s.Close()
		return
	}
	env[payload.Name] = payload.Value
	ctx.SetValue(envKey, env)
	json.NewEncoder(s).Encode(map[string]string{"status": "ok"})
	s.Close()
}

// ------------------------------------------------------------
// READ A LINE FROM SESSION
// ------------------------------------------------------------
func readLine(s ssh.Session) (string, error) {
	var line []byte
	buf := make([]byte, 1024)
	for {
		n, err := s.Read(buf)
		if n > 0 {
			line = append(line, buf[:n]...)
			if i := bytes.IndexByte(line, '\n'); i >= 0 {
				line = bytes.TrimRight(line[:i], "\r")
				return string(line), nil
			}
		}
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				return string(bytes.TrimRight(line, "\r\n")), nil
			}
			return "", err
		}
	}
}

