package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codex-imagegen-mcp/internal/codex"
)

var onePxPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
	0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}

func TestWritePNGCreatesDirsAndReportsDims(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sub", "img.png")
	w, h, n, err := writePNG(out, onePxPNG)
	if err != nil {
		t.Fatal(err)
	}
	if w != 1 || h != 1 || n != len(onePxPNG) {
		t.Fatalf("dims/bytes = %d,%d,%d", w, h, n)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("file not written: %v", err)
	}
}

// GenerateImage must reject empty prompt/out before ever invoking codex.
func TestGenerateImageRequiresPromptAndOut(t *testing.T) {
	d := &Deps{}
	for _, in := range []GenerateInput{
		{Prompt: "", Out: "x.png"},
		{Prompt: "a cat", Out: ""},
	} {
		_, out, err := d.GenerateImage(context.Background(), nil, in)
		if err == nil {
			t.Errorf("expected error for %+v", in)
		}
		if out.OK {
			t.Errorf("OK should be false for %+v", in)
		}
	}
}

func TestNoImagePrefersLastMessage(t *testing.T) {
	out := noImage(&codex.RunResult{LastMessage: "I can't draw that", Stderr: "noise"}, "fallback")
	if out.OK {
		t.Fatal("OK should be false")
	}
	if out.Message != "I can't draw that" {
		t.Fatalf("message = %q; want the agent's LastMessage", out.Message)
	}
}

func TestNoImageFallsBackToStderr(t *testing.T) {
	out := noImage(&codex.RunResult{LastMessage: "", Stderr: "boom: not logged in"}, "codex returned no image")
	if out.OK {
		t.Fatal("OK should be false")
	}
	if !strings.Contains(out.Message, "codex returned no image") || !strings.Contains(out.Message, "boom") {
		t.Fatalf("message should include fallback + stderr; got %q", out.Message)
	}
}
