// Package installepoch answers one narrow question: "has this app data
// folder just landed on a machine/user profile it hasn't run on before?"
// (a fresh install, or the app's own data folder having been copied
// wholesale to a new machine or a new OS user account).
//
// It deliberately does NOT live inside the app's own (copyable) data root
// (see app.go's appDataRoot(), which defaults to
// os.UserConfigDir()/ai-console). If the marker lived inside that folder,
// copying the folder to another machine would carry the marker along and
// defeat the whole point. Instead this package writes a small random id to
// a hidden file that is a *sibling* of that data folder, directly under the
// OS user-config directory — something a user copying only the
// "ai-console" subfolder would not bring with them.
//
// Callers (see shared/settings/persona_code.go) compare this machine id
// against a value they persist inside their own state; a mismatch means
// "treat this as a fresh install" and act accordingly (e.g. regenerate
// per-persona codes). A normal restart on the same machine/profile always
// sees the same id, so nothing resets on ordinary use.
package installepoch

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

const markerFileName = ".ai-console-machine-id"

// LocalMachineID returns a stable random id for this OS user profile on
// this machine. It is created on first read and persisted thereafter. An
// empty string with a non-nil error means the id could not be determined
// (e.g. no writable config dir); callers should treat that as "unknown"
// rather than force a reset.
func LocalMachineID() (string, error) {
	path, err := markerPath()
	if err != nil {
		return "", err
	}

	if existing, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(existing))
		if id != "" {
			return id, nil
		}
	}

	id, err := newRandomID()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(id), 0o600); err != nil {
		return "", err
	}
	return id, nil
}

func markerPath() (string, error) {
	if override := strings.TrimSpace(os.Getenv("AI_CONSOLE_MACHINE_ID_DIR")); override != "" {
		return filepath.Join(override, markerFileName), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil || strings.TrimSpace(configDir) == "" {
		home, herr := os.UserHomeDir()
		if herr != nil || strings.TrimSpace(home) == "" {
			if err == nil {
				err = herr
			}
			return "", err
		}
		return filepath.Join(home, markerFileName), nil
	}
	return filepath.Join(configDir, markerFileName), nil
}

func newRandomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
