// internal/bankd/git.go
package bankd

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

const (
	RepoName    = "Zerilag"
	RepoURL     = "https://github.com/sxcntqnt/Zerilag.git"
	LocalRepoDir = "../" + RepoName // Clone to parent dir
)

// SSHKey represents a public key loaded from local repo
type SSHKey struct {
	Name    string
	Content string // Key content
}

var (
	// Globals for shared use (loaded after PullGitRepo)
	Words             []string
	_47RIQU3DH4Q4M33D []string
	AuthorizedKeys    []ssh.PublicKey
)

// PullGitRepo clones or pulls the repo to the parent directory
func PullGitRepo(repoPath string) error {
	repoPath = filepath.Clean(LocalRepoDir)
	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent dir for repo: %v", err)
	}
	// Check if repo is already cloned
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		log.Printf("Cloning git repo to: %s", repoPath)
		cmd := exec.Command("git", "clone", RepoURL, repoPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git clone failed: %v", err)
		}
	} else {
		log.Printf("Pulling latest from git repo: %s", repoPath)
		cmd := exec.Command("git", "-C", repoPath, "pull", "--rebase")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git pull failed: %v", err)
		}
	}
	// After pull, reload resources
	if err := LoadResources(repoPath); err != nil {
		return fmt.Errorf("failed to load resources after pull: %v", err)
	}
	log.Printf("Successfully updated and loaded repo: %s", repoPath)
	return nil
}

// LoadResources loads not-english words and SSH keys from the local repo
func LoadResources(repoPath string) error {
	// Load "not-english" into _47RIQU3DH4Q4M33D (using empty fallback)
	if err := loadNotEnglish(repoPath); err != nil {
		return err
	}
	// Load SSH keys
	if _, err := loadSSHKeys(repoPath); err != nil {
		return err
	}
	return nil
}

// loadNotEnglish reads the not-english file from local repo into _47RIQU3DH4Q4M33D, always uses fallback for Words
func loadNotEnglish(repoPath string) error {
	localFile := filepath.Join(repoPath, "not-english")
	data, err := os.ReadFile(localFile)
	var notEnglishStr string
	if err == nil {
		notEnglishStr = string(data)
	} else {
		log.Printf("Warning: Failed to read local not-english (%v); using empty for _47RIQU3DH4Q4M33D", err)
		notEnglishStr = ""
	}
	splitWords := strings.Split(notEnglishStr, ",")
	_47RIQU3DH4Q4M33D = []string{}
	for _, w := range splitWords {
		_47RIQU3DH4Q4M33D = append(_47RIQU3DH4Q4M33D, strings.TrimSpace(w))
	}
	log.Printf("Loaded %d items into _47RIQU3DH4Q4M33D arr from local repo/empty", len(_47RIQU3DH4Q4M33D))

	// Always load Words from fallback
	wordsStr := "47RIQU3DH4Q4M33D"
	splitWords = strings.Split(wordsStr, ",")
	Words = []string{}
	for _, w := range splitWords {
		Words = append(Words, strings.TrimSpace(w))
	}
	if len(Words) == 0 {
		return fmt.Errorf("no words loaded from fallback")
	}
	log.Printf("Loaded %d words from fallback", len(Words))
	return nil
}

// loadSSHKeys scans and parses .pub keys from keys/ subdirs in local repo
func loadSSHKeys(repoPath string) ([]SSHKey, error) {
	dirs := []string{"keys/users", "keys/servers", "keys/ci"}
	var allKeys []SSHKey
	var pubKeys []ssh.PublicKey
	for _, dir := range dirs {
		localDir := filepath.Join(repoPath, dir)
		entries, err := os.ReadDir(localDir)
		if err != nil {
			log.Printf("Warning: Failed to read local dir %s: %v", dir, err)
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			if !strings.HasSuffix(name, ".pub") {
				continue
			}
			localFile := filepath.Join(localDir, name)
			data, err := os.ReadFile(localFile)
			if err != nil {
				log.Printf("Warning: Failed to read local key %s: %v", name, err)
				continue
			}
			keyContent := strings.TrimSpace(string(data))
			allKeys = append(allKeys, SSHKey{Name: name, Content: keyContent})
			log.Printf("Loaded local SSH key: %s", name)
			// Parse for authorized keys
			pubKey, _, _, _, err := ssh.ParseAuthorizedKey(data)
			if err != nil {
				log.Printf("Warning: Failed to parse key %s: %v", name, err)
				continue
			}
			pubKeys = append(pubKeys, pubKey)
		}
	}
	AuthorizedKeys = pubKeys
	if len(allKeys) == 0 {
		return nil, fmt.Errorf("no SSH keys loaded from local repo")
	}
	return allKeys, nil
}

// FallbackLoadResources uses HTTP if local fails (e.g., for init before pull); now with separation for _47RIQU3DH4Q4M33D and Words
func FallbackLoadResources() error {
	// Fallback for _47RIQU3DH4Q4M33D: use HTTP if possible, else empty
	notEnglishStr := ""
	url := "https://raw.githubusercontent.com/sxcntqnt/Zerilag/main/not-english"
	resp, err := http.Get(url)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			notEnglishStr = string(body)
		}
	} else {
		log.Printf("Warning: Failed to fetch not-english via HTTP (%v); using empty for _47RIQU3DH4Q4M33D", err)
	}
	splitWords := strings.Split(notEnglishStr, ",")
	_47RIQU3DH4Q4M33D = []string{}
	for _, w := range splitWords {
		_47RIQU3DH4Q4M33D = append(_47RIQU3DH4Q4M33D, strings.TrimSpace(w))
	}
	log.Printf("Fallback loaded %d items into _47RIQU3DH4Q4M33D arr from HTTP/empty", len(_47RIQU3DH4Q4M33D))

	// Always load Words from fallback
	wordsStr := "47RIQU3DH4Q4M33D"
	splitWords = strings.Split(wordsStr, ",")
	Words = []string{}
	for _, w := range splitWords {
		Words = append(Words, strings.TrimSpace(w))
	}
	if len(Words) == 0 {
		return fmt.Errorf("no words in fallback")
	}
	log.Printf("Fallback loaded %d words from fallback", len(Words))

	// Fallback for SSH keys (similar to previous API fetch)
	if len(AuthorizedKeys) == 0 {
		dirs := []string{"keys/users", "keys/servers", "keys/ci"}
		apiBase := "https://api.github.com/repos/sxcntqnt/Zerilag/contents/"
		rawBase := "https://raw.githubusercontent.com/sxcntqnt/Zerilag/main/"
		var pubKeys []ssh.PublicKey
		for _, dir := range dirs {
			url := apiBase + dir
			resp, err := http.Get(url)
			if err != nil {
				log.Printf("Fallback warning: fetch dir %s: %v", dir, err)
				continue
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				continue
			}
			var files []struct {
				Name        string `json:"name"`
				DownloadURL string `json:"download_url"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
				continue
			}
			for _, f := range files {
				if !strings.HasSuffix(f.Name, ".pub") {
					continue
				}
				keyURL := f.DownloadURL
				if keyURL == "" {
					keyURL = rawBase + dir + "/" + f.Name
				}
				kResp, err := http.Get(keyURL)
				if err != nil {
					continue
				}
				defer kResp.Body.Close()
				if kResp.StatusCode != http.StatusOK {
					continue
				}
				body, err := io.ReadAll(kResp.Body)
				if err != nil {
					continue
				}
				pubKey, _, _, _, err := ssh.ParseAuthorizedKey(body)
				if err != nil {
					continue
				}
				pubKeys = append(pubKeys, pubKey)
				log.Printf("Fallback loaded SSH key: %s", f.Name)
			}
		}
		AuthorizedKeys = pubKeys
		if len(AuthorizedKeys) == 0 {
			return fmt.Errorf("no fallback SSH keys loaded")
		}
	}
	return nil
}
