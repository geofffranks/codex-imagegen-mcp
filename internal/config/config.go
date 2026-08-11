package config

import (
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type Config struct {
	CodexBin                 string
	SessionsDir              string
	GeneratedImagesDir       string
	Timeout                  time.Duration
	ManifestPath             string
	DefaultEffort            string
	DangerouslyBypassSandbox bool
}

func Load() Config {
	return Config{
		CodexBin:                 envOr("CODEX_BIN", "codex"),
		SessionsDir:              envOr("CODEX_IMAGEGEN_SESSIONS_DIR", defaultSessionsDir()),
		GeneratedImagesDir:       envOr("CODEX_IMAGEGEN_IMAGES_DIR", defaultGeneratedImagesDir()),
		Timeout:                  envDuration("CODEX_IMAGEGEN_TIMEOUT", 180*time.Second),
		ManifestPath:             envOr("CODEX_IMAGEGEN_MANIFEST", defaultManifest()),
		DefaultEffort:            envOr("CODEX_IMAGEGEN_DEFAULT_EFFORT", "low"),
		DangerouslyBypassSandbox: envBool("CODEX_IMAGEGEN_DANGEROUSLY_BYPASS_SANDBOX"),
	}
}

// defaultSessionsDir resolves $CODEX_HOME/sessions, falling back to ~/.codex/sessions.
func defaultSessionsDir() string {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".codex", "sessions") // last-resort fallback
		}
		home = filepath.Join(h, ".codex")
	}
	return filepath.Join(home, "sessions")
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// envDuration parses a duration from the named env var; unparseable values are silently discarded and def is used.
func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func envBool(key string) bool {
	v, err := strconv.ParseBool(os.Getenv(key))
	return err == nil && v
}

func defaultGeneratedImagesDir() string {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return filepath.Join(".codex", "generated_images")
		}
	}
	return filepath.Join(home, "generated_images")
}

func defaultManifest() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".codex-imagegen-mcp/manifest.jsonl" // last-resort fallback
	}
	return filepath.Join(home, ".codex-imagegen-mcp", "manifest.jsonl")
}
