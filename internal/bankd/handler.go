// internal/bankd/handler.go
package bankd

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
	"strings"
	"encoding/json"
	"bytes"

	"github.com/gliderlabs/ssh"
)

const (
	envKey contextKey = "env"
)


// SSHHandler runs each SSH session inside a fully isolated RootlessKit environment
func SSHHandler(s ssh.Session) {
	username := s.User()
	if username == "" {
		io.WriteString(s, "Ziklag CLI Error: No username provided\n")
		s.Exit(1)
		return
	}

	// PTY setup
	_, term := initPTY(s)
	if term == nil {
		return
	}

	log.Printf("Starting CLI session for %s", username)
	showWelcome(term)

	// Attach session to context
	s.Context().SetValue(sessionKey, s)
	ctx := s.Context()

	partnerID, ok := getPartnerID(ctx, term)
	if !ok {
		return
	}

	env := loadEnv(ctx)
	showBankEthos(term, partnerID, env)

	// Ensure RootlessKit exists
	if err := ensureRootlessKit(term, username); err != nil {
		return
	}

	cmd, tempHome := prepareRootlessCmd(s, username)
	defer cleanupTempHome(tempHome, username)

	runRootlessSession(s, cmd, username)
}

// -----------------------------
// Helper: PTY & terminal
// -----------------------------
func initPTY(s ssh.Session) (bool, *TerminalWriter) {
	hasPTY, _ := setupPTY(s)
	term := NewTerminalWriter(s)
	if !hasPTY {
		io.WriteString(term.writer, "Ziklag CLI Error: PTY not allocated\n")
		s.Exit(1)
		return false, nil
	}
	return hasPTY, &term
}

// -----------------------------
// Helper: get partnerID
// -----------------------------
func getPartnerID(ctx ssh.Context, term *TerminalWriter) (string, bool) {
	partnerID, ok := ctx.Value(partnerIDKey).(string)
	if !ok || partnerID == "" {
		fmt.Fprintln(term.writer, "Error: Authentication failed or partner ID missing.")
		s := ctx.Value(sessionKey).(ssh.Session)
		s.Exit(1)
		return "", false
	}
	return partnerID, true
}

// -----------------------------
// Helper: show welcome
// -----------------------------
func showWelcome(term *TerminalWriter) {
	term.Write([]byte("Welcome to the CLI!\n"))
}

// -----------------------------
// Helper: show full Bank Ethos
// -----------------------------
func showBankEthos(term *TerminalWriter, partnerID string, env map[string]string) {
	fmt.Fprintf(term.writer, "🚀 Ziklag Bank Ethos — Partner: %s 🚀\n", partnerID)
	fmt.Fprint(term.writer, "Open code. Open scrutiny. Strong foundations built in the open.\n")
	fmt.Fprint(term.writer, " Karibu kwenye himaya ya wasimamizi wa data.\n")
	fmt.Fprint(term.writer, "Designed for builders. Shaped by precision.\n")
	fmt.Fprint(term.writer, "Driven by open-source collaboration. Tunainuka pamoja.\n")
	fmt.Fprint(term.writer, "Built for Africa’s pace: instant updates, smart workflows, and adaptive safeguards.\n")
	fmt.Fprint(term.writer, "Africa’s sharpest edge: Real-time balances, resilient systems, and battle-tested engineering.\n")
	fmt.Fprint(term.writer, "Open-source spirit. Continental ambition. Let's build the future brick by brick.\n")
	fmt.Fprint(term.writer, "Ziklag once burned, but its warriors rose sharper than before.\n")
	fmt.Fprint(term.writer, "That same resilience fuels our infrastructure: swift, alert, and forged in scrutiny.\n")
	fmt.Fprint(term.writer, "Karibu katika lango la waendelezaji—where hustle meets divine tenacity.\n\n")
	fmt.Fprint(term.writer, "Commands: balance, account, deposit, withdraw, assets, profile, help, exit\n\n")

	fmt.Fprintln(term.writer, "Environment:")
	for k, v := range env {
		fmt.Fprintf(term.writer, "%s=%s\n", k, v)
	}
	fmt.Fprintln(term.writer)
}

// -----------------------------
// Helper: ensure rootlesskit
// -----------------------------
func ensureRootlessKit(term *TerminalWriter, username string) error {
	if _, err := exec.LookPath("rootlesskit"); err != nil {
		log.Printf("rootlesskit not found for %s: %v", username, err)
		io.WriteString(term.writer, "Ziklag CLI Error: rootlesskit not installed\n")
		s := term.s
		s.Exit(1)
		return err
	}
	return nil
}

// -----------------------------
// Helper: prepare rootless command
// -----------------------------
func prepareRootlessCmd(s ssh.Session, username string) (*exec.Cmd, string) {
	rkArgs := []string{
		"--net=none",
		"--pidns",
		"--ipcns",
		"--utsns",
		"--reaper=true",
		"--copy-up=/etc",
		"--copy-up=/var",
		"--copy-up=/run",
		"--copy-up=/tmp",
		"--propagation=rprivate",
		"--disable-host-loopback",
		"--subid-source=static",
	}

	userCmd := s.Command()
	if len(userCmd) == 0 {
		rkArgs = append(rkArgs, "/bin/bash", "-l")
	} else {
		raw := s.RawCommand()
		if raw == "" {
			io.WriteString(s, "Ziklag CLI Error: Empty command provided\n")
			s.Exit(1)
			return nil, ""
		}
		rkArgs = append(rkArgs, "/bin/bash", "-c", raw)
	}

	cmd := exec.Command("rootlesskit", rkArgs...)

	tempHome, _ := os.MkdirTemp("/tmp", "home-"+username+"-")
	cmd.Env = []string{
		"USER=" + username,
		"LOGNAME=" + username,
		"HOME=" + tempHome,
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
	}
	cmd.Dir = tempHome
	cmd.Stdin = s
	cmd.Stdout = s
	cmd.Stderr = s

	return cmd, tempHome
}

// -----------------------------
// Helper: cleanup temporary HOME
// -----------------------------
func cleanupTempHome(tempHome, username string) {
	if tempHome != "" {
		if err := os.RemoveAll(tempHome); err != nil {
			log.Printf("Cleanup failed for %s HOME (%s): %v", username, tempHome, err)
		}
	}
}

// -----------------------------
// Helper: run rootless session
// -----------------------------
func runRootlessSession(s ssh.Session, cmd *exec.Cmd, username string) {
	ctx := s.Context()
	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	start := time.Now()
	remoteAddr := s.RemoteAddr()
	log.Printf("Rootless session started for %s (remote: %s)", username, remoteAddr)

	select {
	case err := <-done:
		duration := time.Since(start)
		if err != nil {
			log.Printf("Rootless session failed for %s (%v, remote: %s): %v", username, duration, remoteAddr, err)
			select {
			case <-ctx.Done():
			default:
				io.WriteString(s, fmt.Sprintf("Ziklag CLI Error: Session failed - %v\n", err))
			}
			s.Exit(1)
		} else {
			log.Printf("Rootless session ended for %s (%v, remote: %s)", username, duration, remoteAddr)
		}
	case <-ctx.Done():
		log.Printf("Rootless session cancelled for %s (remote: %s)", username, remoteAddr)
		if cmd.Process != nil {
			cmd.Process.Signal(syscall.SIGINT)
			time.Sleep(300 * time.Millisecond)
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		s.Exit(1)
	}
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

