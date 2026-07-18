package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("CODEX_BIN", "")
	t.Setenv("CODEX_IMAGEGEN_TIMEOUT", "")
	t.Setenv("CODEX_IMAGEGEN_DEFAULT_EFFORT", "")
	t.Setenv("CODEX_IMAGEGEN_SESSIONS_DIR", "")
	t.Setenv("CODEX_HOME", "/tmp/fakehome")
	t.Setenv("CODEX_IMAGEGEN_MANIFEST", "")
	t.Setenv("CODEX_IMAGEGEN_DEFAULT_MODEL", "")
	c := Load()
	if c.CodexBin != "codex" {
		t.Errorf("CodexBin = %q, want codex", c.CodexBin)
	}
	if c.Timeout != 180*time.Second {
		t.Errorf("Timeout = %v, want 180s", c.Timeout)
	}
	if c.DefaultEffort != "low" {
		t.Errorf("DefaultEffort = %q, want low", c.DefaultEffort)
	}
	if c.SessionsDir != "/tmp/fakehome/sessions" {
		t.Errorf("SessionsDir = %q, want /tmp/fakehome/sessions", c.SessionsDir)
	}
	if c.GeneratedImagesDir != "/tmp/fakehome/generated_images" {
		t.Errorf("GeneratedImagesDir = %q, want /tmp/fakehome/generated_images", c.GeneratedImagesDir)
	}
	if c.DefaultModel != "" {
		t.Errorf("DefaultModel = %q, want empty", c.DefaultModel)
	}
	if !strings.HasSuffix(c.ManifestPath, "/.codex-imagegen-mcp/manifest.jsonl") {
		t.Errorf("ManifestPath = %q, want suffix /.codex-imagegen-mcp/manifest.jsonl", c.ManifestPath)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("CODEX_BIN", "/opt/homebrew/bin/codex")
	t.Setenv("CODEX_IMAGEGEN_TIMEOUT", "45s")
	t.Setenv("CODEX_IMAGEGEN_DEFAULT_EFFORT", "")
	t.Setenv("CODEX_IMAGEGEN_SESSIONS_DIR", "")
	t.Setenv("CODEX_HOME", "")
	t.Setenv("CODEX_IMAGEGEN_MANIFEST", "")
	t.Setenv("CODEX_IMAGEGEN_DEFAULT_MODEL", "")
	c := Load()
	if c.CodexBin != "/opt/homebrew/bin/codex" {
		t.Errorf("CodexBin = %q", c.CodexBin)
	}
	if c.Timeout != 45*time.Second {
		t.Errorf("Timeout = %v, want 45s", c.Timeout)
	}
}
