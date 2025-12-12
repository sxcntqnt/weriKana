// internal/bankd/server.go
package bankd

import (
	"context"
	"log"
	"sync"

	"weriKana/internal/appcontext"

	"github.com/gliderlabs/ssh"
)

var (
	appCtx   *appcontext.AppContext
	server   *ssh.Server
	serverMu sync.Mutex
)

// StartSSHServer starts the SSH server with git-backed authorized keys.
func StartSSHServer(ctx *appcontext.AppContext, gitRepoPath string) error {
	serverMu.Lock()
	defer serverMu.Unlock()

	appCtx = ctx
	gitAuth := NewGitAuthorizedKeys(gitRepoPath)

	server = &ssh.Server{
		Addr:             ":2222",
		Handler:          SSHHandler,                     // main session handler
		PublicKeyHandler: PublicKeyAuthHandler(gitAuth), // git-backed keys
		PasswordHandler:  PasswordAuthHandler,           // password auth
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"env": EnvSubsystemHandler,
		},
	}

	server.AddHostKey(loadHostKey())

	if err := PullGitRepo(gitRepoPath); err != nil {
		log.Printf("Warning: Failed to pull git repo on startup: %v", err)
	}

	// Run server in goroutine (ListenAndServe is blocking)
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Printf("SSH server stopped: %v", err)
		}
	}()

	return nil
}

// ShutdownSSH gracefully stops the SSH server.
func ShutdownSSH(ctx context.Context) error {
	serverMu.Lock()
	defer serverMu.Unlock()

	if server == nil {
		return nil
	}

	done := make(chan struct{})
	go func() {
		server.Close()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
		server = nil
		return nil
	}
}

