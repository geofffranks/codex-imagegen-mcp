package tools

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codex-imagegen-mcp/internal/config"
)

// TestGenerateAndListRecentE2E drives the real GenerateImage handler end to end
// (codex $imagegen -> rollout -> PNG on disk -> manifest) and then ListRecent.
// Opt-in: requires codex installed + logged in.
func TestGenerateAndListRecentE2E(t *testing.T) {
	if os.Getenv("CODEX_IMAGEGEN_E2E") != "1" {
		t.Skip("set CODEX_IMAGEGEN_E2E=1 to run the real codex tools E2E")
	}
	cfg := config.Load()
	cfg.ManifestPath = filepath.Join(t.TempDir(), "manifest.jsonl") // don't touch the real manifest
	if cfg.Timeout < 240*time.Second {
		cfg.Timeout = 240 * time.Second
	}
	d := New(cfg)

	out := filepath.Join(t.TempDir(), "circle.png")
	prompt := "a plain green circle on white, centered"

	_, gen, err := d.GenerateImage(context.Background(), nil, GenerateInput{Prompt: prompt, Out: out})
	if err != nil {
		t.Fatalf("GenerateImage: %v", err)
	}
	if !gen.OK {
		t.Fatalf("generate not OK: %q", gen.Message)
	}

	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("output not written: %v", err)
	}
	if len(data) < 1024 || !bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
		t.Fatalf("output at %s is not a PNG (len=%d)", out, len(data))
	}

	_, recent, err := d.ListRecent(context.Background(), nil, RecentInput{Limit: 5})
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(recent.Images) == 0 || recent.Images[0].Path != out {
		t.Fatalf("list_recent should show %s newest-first; got %+v", out, recent.Images)
	}
	if recent.Images[0].Prompt != prompt {
		t.Fatalf("prompt mismatch: got %q", recent.Images[0].Prompt)
	}
}
