package codex

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Smallest valid PNG (1x1, from the Go image/png encoder), captured as bytes.
var onePxPNG = []byte{
	0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D,
	0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00,
	0x0D, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x62, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49,
	0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82,
}

func TestExtractPNGFromNestedEvent(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString(onePxPNG)
	line := `{"type":"response_item","payload":{"type":"custom_tool_call_output","result":"` + b64 + `"}}`
	got, err := ExtractPNG(strings.NewReader("{\"type\":\"session_meta\"}\n" + line + "\n"))
	if err != nil {
		t.Fatalf("ExtractPNG: %v", err)
	}
	if !bytes.Equal(got, onePxPNG) {
		t.Fatalf("bytes mismatch: got %d, want %d", len(got), len(onePxPNG))
	}
}

func TestExtractPNGNoImage(t *testing.T) {
	_, err := ExtractPNG(strings.NewReader(`{"payload":{"type":"agent_message","message":"I can't do that"}}` + "\n"))
	if !errors.Is(err, ErrNoImage) {
		t.Fatalf("err = %v, want ErrNoImage", err)
	}
}

func TestPNGDimensions(t *testing.T) {
	w, h, ok := PNGDimensions(onePxPNG)
	if !ok || w != 1 || h != 1 {
		t.Fatalf("dims = %d,%d ok=%v; want 1,1,true", w, h, ok)
	}
}

// Real-shape fixture captured from codex 0.144.1 (base64 swapped for a 1x1 PNG).
func TestExtractPNGFromRealFixture(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "testdata", "rollout-with-image.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := ExtractPNG(f)
	if err != nil {
		t.Fatalf("ExtractPNG on fixture: %v", err)
	}
	if w, h, ok := PNGDimensions(got); !ok || w != 1 || h != 1 {
		t.Fatalf("fixture dims = %d,%d ok=%v", w, h, ok)
	}
}

func TestFindRolloutByThreadID(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "2026", "07", "12")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	tid := "019f5721-4c3f-7ac1-977e-6668665ff438"
	want := filepath.Join(nested, "rollout-2026-07-12T12-20-27-"+tid+".jsonl")
	for _, name := range []string{want, filepath.Join(nested, "rollout-other-deadbeef.jsonl")} {
		if err := os.WriteFile(name, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := FindRollout(dir, tid, time.Time{})
	if err != nil || got != want {
		t.Fatalf("FindRollout = %q, %v; want %q", got, err, want)
	}
}

func TestFindRolloutNoneReturnsErr(t *testing.T) {
	_, err := FindRollout(t.TempDir(), "nope", time.Time{})
	if !errors.Is(err, ErrNoRollout) {
		t.Fatalf("err = %v, want ErrNoRollout", err)
	}
}

func TestExtractPNGUrlSafeBase64(t *testing.T) {
	b64 := base64.RawURLEncoding.EncodeToString(onePxPNG)
	line := `{"type":"response_item","payload":{"result":"` + b64 + `"}}`
	got, err := ExtractPNG(strings.NewReader(line + "\n"))
	if err != nil {
		t.Fatalf("ExtractPNG with RawURLEncoding: %v", err)
	}
	if !bytes.Equal(got, onePxPNG) {
		t.Fatalf("bytes mismatch: got %d, want %d", len(got), len(onePxPNG))
	}
}

func TestFindRolloutNewestSinceFallback(t *testing.T) {
	dir := t.TempDir()
	older := filepath.Join(dir, "rollout-older.jsonl")
	newer := filepath.Join(dir, "rollout-newer.jsonl")
	for _, name := range []string{older, newer} {
		if err := os.WriteFile(name, []byte("{}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// Set deterministic mtimes: older=T+1s, newer=T+2s.
	base := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	if err := os.Chtimes(older, base.Add(time.Second), base.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(newer, base.Add(2*time.Second), base.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	got, err := FindRollout(dir, "", base)
	if err != nil {
		t.Fatalf("FindRollout: %v", err)
	}
	if got != newer {
		t.Fatalf("FindRollout = %q; want %q (newest)", got, newer)
	}
}
