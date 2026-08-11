package codex

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type RunOpts struct {
	Bin                      string
	Prompt                   string
	Effort                   string
	Workdir                  string
	Images                   []string
	OutputLastMessage        string
	Timeout                  time.Duration
	DangerouslyBypassSandbox bool
}

type RunResult struct {
	Stdout      []byte
	Stderr      string
	LastMessage string
	ThreadID    string
	StartedAt   time.Time
	TimedOut    bool
}

func emptyStdin() io.Reader {
	return bytes.NewReader(nil)
}

func buildArgs(o RunOpts) []string {
	// No --ephemeral: the rollout must be written so the image bytes are reachable.
	args := []string{"exec", "--json", "--skip-git-repo-check", "-C", o.Workdir}
	effort := o.Effort
	if effort == "minimal" {
		effort = "low"
	}
	if effort != "" && effort != "default" {
		args = append(args, "-c", "model_reasoning_effort="+effort)
	}
	for _, img := range o.Images {
		args = append(args, "-i", img)
	}
	// codex exec takes the whole prompt as ONE positional argument; "$imagegen"
	// is parsed by codex's skill system, not a shell. Passing it as a single
	// argv element is deliberate and spike-verified — do NOT split on spaces.
	// The -- delimiter is required because --image accepts a variadic list and
	// otherwise consumes the prompt as another image argument. All options must
	// come before -- because everything after it is the positional prompt.
	if o.OutputLastMessage != "" {
		args = append(args, "-o", o.OutputLastMessage)
	}
	if o.DangerouslyBypassSandbox {
		args = append(args, "--dangerously-bypass-approvals-and-sandbox")
	}
	args = append(args, "--")
	return append(args, "$imagegen "+o.Prompt)
}

// parseThreadID returns the thread_id from the {"type":"thread.started",...} line
// of codex --json stdout, or "" if absent.
func parseThreadID(stdout []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(stdout))
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20) // tolerate a large event line before thread.started
	for sc.Scan() {
		var e struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
		}
		if json.Unmarshal(sc.Bytes(), &e) == nil && e.Type == "thread.started" {
			return e.ThreadID
		}
	}
	return ""
}

// Run executes codex and captures its JSONL stdout, the thread_id, and the
// agent's final text (via --output-last-message) so callers can explain a
// no-image result. It records StartedAt for the rollout-mtime fallback.
func Run(ctx context.Context, o RunOpts) (*RunResult, error) {
	ctx, cancel := context.WithTimeout(ctx, o.Timeout)
	defer cancel()

	lastMsgFile, err := os.CreateTemp("", "imagegen-last-*.txt")
	if err != nil {
		return nil, err
	}
	lastMsgFile.Close()
	defer os.Remove(lastMsgFile.Name())

	o.OutputLastMessage = lastMsgFile.Name()
	started := time.Now()
	args := buildArgs(o)
	cmd := exec.CommandContext(ctx, o.Bin, args...)
	cmd.Stdin = emptyStdin()                              // explicit EOF; codex probes stdin for additional input
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // own process group
	cmd.Cancel = func() error {                           // kill the whole group on timeout
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	res := &RunResult{
		Stdout:    stdout.Bytes(),
		Stderr:    stderr.String(),
		StartedAt: started,
		ThreadID:  parseThreadID(stdout.Bytes()),
	}
	if data, _ := os.ReadFile(lastMsgFile.Name()); len(data) > 0 {
		res.LastMessage = string(bytes.TrimSpace(data))
	}
	// Success takes precedence: if codex exited 0, a deadline that expired in the
	// same instant must not turn a completed generation into a timeout error.
	if runErr == nil {
		return res, nil
	}
	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		return res, fmt.Errorf("codex imagegen timed out after %s", o.Timeout)
	}
	return res, fmt.Errorf("codex exec failed: %w (stderr: %s)", runErr, res.Stderr)
}
