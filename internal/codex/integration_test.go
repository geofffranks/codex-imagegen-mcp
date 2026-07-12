package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunRealImagegen(t *testing.T) {
	if os.Getenv("CODEX_IMAGEGEN_E2E") != "1" {
		t.Skip("set CODEX_IMAGEGEN_E2E=1 to run the real codex integration test")
	}
	wd := t.TempDir()
	res, err := Run(context.Background(), RunOpts{
		Bin: "codex", Prompt: "a plain blue circle on white", Workdir: wd,
		Effort: "low", Timeout: 240 * time.Second,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if res.ThreadID == "" {
		t.Fatal("no thread_id parsed from --json stdout")
	}
	sessionsDir := filepath.Join(os.Getenv("HOME"), ".codex", "sessions")
	if h := os.Getenv("CODEX_HOME"); h != "" {
		sessionsDir = filepath.Join(h, "sessions")
	}
	rollout, err := FindRollout(sessionsDir, res.ThreadID, res.StartedAt)
	if err != nil {
		t.Fatalf("FindRollout: %v", err)
	}
	f, err := os.Open(rollout)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	png, err := ExtractPNG(f)
	if err != nil {
		t.Fatalf("ExtractPNG: %v (last message: %s)", err, res.LastMessage)
	}
	if w, h, ok := PNGDimensions(png); !ok || w == 0 || h == 0 {
		t.Fatalf("bad dims %d,%d ok=%v", w, h, ok)
	}
}
