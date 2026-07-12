package manifest

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// appendMu serializes Append within this process so concurrent generate_image
// calls can't interleave partial lines (O_APPEND is only atomic up to PIPE_BUF).
var appendMu sync.Mutex

type Record struct {
	Path      string `json:"path"`
	Prompt    string `json:"prompt"`
	Timestamp string `json:"timestamp"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Bytes     int    `json:"bytes"`
}

func Append(path string, r Record) error {
	appendMu.Lock()
	defer appendMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	line, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return err
}

// ReadRecent returns up to `limit` records, newest first. Missing file → empty.
func ReadRecent(path string, limit int) ([]Record, error) {
	f, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var all []Record
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 1<<20)
	for sc.Scan() {
		var r Record
		if json.Unmarshal(sc.Bytes(), &r) == nil {
			all = append(all, r)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	// newest-first
	out := make([]Record, 0, limit)
	for i := len(all) - 1; i >= 0 && len(out) < limit; i-- {
		out = append(out, all[i])
	}
	return out, nil
}
