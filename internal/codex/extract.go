package codex

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"
)

var ErrNoImage = errors.New("no image found in codex output")
var ErrNoRollout = errors.New("no matching codex rollout file found")

var pngMagic = []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}

// ExtractPNG scans a JSONL stream (codex --json stdout, or a rollout file) and
// returns the first PNG it finds. It walks every JSON string value and selects
// any whose base64 decoding begins with the PNG signature. This is deliberately
// field-agnostic so it survives codex renaming event fields across versions.
func ExtractPNG(r io.Reader) ([]byte, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20) // PNGs are large; raise line cap to 64MB
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var v any
		if err := json.Unmarshal(line, &v); err != nil {
			continue // tolerate non-JSON lines
		}
		if png := findPNG(v); png != nil {
			return png, nil
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return nil, ErrNoImage
}

// decodePNGCandidate tries all four base64 encoding variants (standard/URL,
// padded/unpadded) and returns the decoded bytes if any variant produces data
// beginning with the PNG magic signature, or nil otherwise.
func decodePNGCandidate(s string) []byte {
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding,
		base64.RawStdEncoding,
		base64.URLEncoding,
		base64.RawURLEncoding,
	} {
		// Cheap peek: decode ~9 bytes and check the PNG magic before committing to a
		// full decode of a possibly multi-MB string that isn't an image.
		if len(s) >= 12 {
			if head, err := enc.DecodeString(s[:12]); err != nil || !bytes.HasPrefix(head, pngMagic) {
				continue
			}
		}
		if d, err := enc.DecodeString(s); err == nil && bytes.HasPrefix(d, pngMagic) {
			return d
		}
	}
	return nil
}

func findPNG(v any) []byte {
	switch t := v.(type) {
	case string:
		// Real PNGs base64-encode to far more than this; the guard just cheaply
		// skips short non-image strings before attempting four base64 decodes.
		if len(t) < 64 {
			return nil
		}
		return decodePNGCandidate(t)
	case []any:
		for _, e := range t {
			if png := findPNG(e); png != nil {
				return png
			}
		}
	case map[string]any:
		for _, e := range t {
			if png := findPNG(e); png != nil {
				return png
			}
		}
	}
	return nil
}

// PNGDimensions reads width/height from the IHDR chunk (bytes 16..24).
func PNGDimensions(data []byte) (w, h int, ok bool) {
	if len(data) < 24 || !bytes.HasPrefix(data, pngMagic) {
		return 0, 0, false
	}
	return int(binary.BigEndian.Uint32(data[16:20])),
		int(binary.BigEndian.Uint32(data[20:24])), true
}

// FindRollout locates the codex session rollout for a generation. If threadID is
// non-empty it returns the rollout-*.jsonl whose filename contains that UUID.
// Otherwise (fallback) it returns the newest rollout-*.jsonl with mtime >= since.
func FindRollout(sessionsDir, threadID string, since time.Time) (string, error) {
	var newest string
	var newestMod time.Time
	err := filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == sessionsDir {
				return err // sessions dir itself unreadable — surface it, don't mask as ErrNoRollout
			}
			return nil // skip unreadable subtrees
		}
		if d.IsDir() || !strings.HasPrefix(d.Name(), "rollout-") || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		if threadID != "" {
			if strings.Contains(d.Name(), threadID) {
				newest = path
				return fs.SkipAll // exact match; stop walking
			}
			return nil
		}
		info, err := d.Info()
		if err != nil || info.ModTime().Before(since) {
			return nil // keeps only files with mtime >= since
		}
		if info.ModTime().After(newestMod) {
			newestMod, newest = info.ModTime(), path
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if newest == "" {
		return "", ErrNoRollout
	}
	return newest, nil
}
