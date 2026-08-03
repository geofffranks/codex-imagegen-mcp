package tools

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
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
func TestGenerateInputHasNoModelSelection(t *testing.T) {
	if _, ok := reflect.TypeOf(GenerateInput{}).FieldByName("Model"); ok {
		t.Fatal("GenerateInput must not expose model selection")
	}
}

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

func TestBuildPromptReferenceBlock(t *testing.T) {
	got := buildPrompt("draw a dragon", []string{"/r1.png", "/r2.png"}, "")
	if !strings.HasPrefix(got, "draw a dragon") {
		t.Fatalf("original prompt must come first: %q", got)
	}
	if !strings.Contains(got, "Reference images are attached") {
		t.Fatalf("missing reference instruction: %q", got)
	}
	if !strings.Contains(got, "Do not edit or modify them") {
		t.Fatalf("missing do-not-edit guidance: %q", got)
	}
}

func TestBuildPromptAspectRatioHint(t *testing.T) {
	cases := map[string]string{
		"16:9": "wide 16:9 landscape",
		"9:16": "tall 9:16 portrait",
		"1:1":  "square 1:1",
		"4:3":  "4:3 landscape",
		"3:4":  "3:4 portrait",
	}
	for ratio, wantFrag := range cases {
		got := buildPrompt("a hillside", nil, ratio)
		if !strings.Contains(got, wantFrag) {
			t.Fatalf("ratio %q: expected %q in prompt: %q", ratio, wantFrag, got)
		}
	}
	// Unknown ratio falls back to generic mention.
	got := buildPrompt("a hillside", nil, "21:9")
	if !strings.Contains(got, "21:9 aspect ratio") {
		t.Fatalf("unknown ratio should appear generically: %q", got)
	}
}

func TestBuildPromptNoAugmentationWhenEmpty(t *testing.T) {
	got := buildPrompt("just a prompt", nil, "")
	if got != "just a prompt" {
		t.Fatalf("expected unchanged prompt, got %q", got)
	}
}

func TestGenerateImageRejectsMissingReferenceImage(t *testing.T) {
	d := &Deps{}
	in := GenerateInput{
		Prompt:          "a dragon",
		Out:             filepath.Join(t.TempDir(), "out.png"),
		ReferenceImages: []string{"/nonexistent/ref.png"},
	}
	_, _, err := d.GenerateImage(context.Background(), nil, in)
	if err == nil {
		t.Fatal("expected error for missing reference image")
	}
	if !strings.Contains(err.Error(), "/nonexistent/ref.png") {
		t.Fatalf("error should name the missing path: %v", err)
	}
}
