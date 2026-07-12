package manifest

import (
	"path/filepath"
	"testing"
)

func TestAppendAndReadRecent(t *testing.T) {
	p := filepath.Join(t.TempDir(), "manifest.jsonl")
	for i, name := range []string{"a", "b", "c"} {
		if err := Append(p, Record{Path: name, Prompt: name, Timestamp: "2026-07-12T00:0" + string(rune('0'+i)) + ":00Z", Width: 1254, Height: 1254, Bytes: 100 + i}); err != nil {
			t.Fatal(err)
		}
	}
	recs, err := ReadRecent(p, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 || recs[0].Path != "c" || recs[1].Path != "b" {
		t.Fatalf("recent = %+v; want newest-first c,b", recs)
	}
}

func TestReadRecentMissingFile(t *testing.T) {
	recs, err := ReadRecent(filepath.Join(t.TempDir(), "nope.jsonl"), 10)
	if err != nil || len(recs) != 0 {
		t.Fatalf("missing manifest should yield empty, got %v err=%v", recs, err)
	}
}
