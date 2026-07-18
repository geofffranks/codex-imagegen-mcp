package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"codex-imagegen-mcp/internal/codex"
	"codex-imagegen-mcp/internal/manifest"
)

type GenerateInput struct {
	Prompt  string `json:"prompt" jsonschema:"the image description to generate"`
	Out     string `json:"out" jsonschema:"file path to write the PNG to (absolute, or relative to the server's working directory)"`
	Model   string `json:"model,omitempty" jsonschema:"optional codex model override"`
	Effort  string `json:"effort,omitempty" jsonschema:"reasoning effort: minimal, low, default, medium, high, or xhigh"`
	Workdir string `json:"workdir,omitempty" jsonschema:"optional codex working directory; defaults to a temp dir"`
}

type GenerateOutput struct {
	OK      bool   `json:"ok"`
	Path    string `json:"path,omitempty"`
	Width   int    `json:"width,omitempty"`
	Height  int    `json:"height,omitempty"`
	Bytes   int    `json:"bytes,omitempty"`
	Message string `json:"message,omitempty"`
}

func writePNG(out string, data []byte) (w, h, n int, err error) {
	if err = os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return 0, 0, 0, err
	}
	if err = os.WriteFile(out, data, 0o644); err != nil {
		return 0, 0, 0, err
	}
	w, h, _ = codex.PNGDimensions(data)
	return w, h, len(data), nil
}

func (d *Deps) GenerateImage(ctx context.Context, _ *mcp.CallToolRequest, in GenerateInput) (*mcp.CallToolResult, GenerateOutput, error) {
	if in.Prompt == "" || in.Out == "" {
		return nil, GenerateOutput{}, fmt.Errorf("both 'prompt' and 'out' are required")
	}

	workdir := in.Workdir
	if workdir == "" {
		tmp, err := os.MkdirTemp("", "imagegen-wd-*")
		if err != nil {
			return nil, GenerateOutput{}, err
		}
		defer os.RemoveAll(tmp)
		workdir = tmp
	}

	effort := in.Effort
	if effort == "" {
		effort = d.Cfg.DefaultEffort
	}
	model := in.Model
	if model == "" {
		model = d.Cfg.DefaultModel
	}

	res, runErr := codex.Run(ctx, codex.RunOpts{
		Bin: d.Cfg.CodexBin, Prompt: in.Prompt, Model: model,
		Effort: effort, Workdir: workdir, Timeout: d.Cfg.Timeout,
	})
	if runErr != nil {
		return nil, GenerateOutput{}, runErr
	}

	// Newer Codex versions write generated images as files under
	// generated_images/<thread-id>; older versions embedded base64 PNG bytes in
	// the session rollout, so retain that path as a compatibility fallback.
	var png []byte
	var err error
	if imagePath, imageErr := codex.FindGeneratedImage(ctx, d.Cfg.GeneratedImagesDir, res.ThreadID, res.StartedAt, 5*time.Second); imageErr == nil {
		png, err = os.ReadFile(imagePath)
	} else {
		rollout, rolloutErr := codex.FindRollout(d.Cfg.SessionsDir, res.ThreadID, res.StartedAt)
		if rolloutErr != nil {
			return nil, noImage(res, "could not locate codex output: "+imageErr.Error()+"; "+rolloutErr.Error()), nil
		}
		f, openErr := os.Open(rollout)
		if openErr != nil {
			return nil, GenerateOutput{}, openErr
		}
		defer f.Close()
		png, err = codex.ExtractPNG(f)
	}
	if err != nil {
		// No image: surface the agent's own explanation instead of a bare failure.
		return nil, noImage(res, "codex returned no image"), nil
	}

	w, h, n, err := writePNG(in.Out, png)
	if err != nil {
		return nil, GenerateOutput{}, err
	}

	_ = manifest.Append(d.Cfg.ManifestPath, manifest.Record{
		Path: in.Out, Prompt: in.Prompt, Timestamp: d.Now(),
		Width: w, Height: h, Bytes: n,
	})

	return nil, GenerateOutput{OK: true, Path: in.Out, Width: w, Height: h, Bytes: n}, nil
}

func noImage(res *codex.RunResult, fallback string) GenerateOutput {
	msg := res.LastMessage
	if msg == "" {
		msg = fallback
		// The agent left no message; codex stderr is the best remaining diagnostic.
		if s := strings.TrimSpace(res.Stderr); s != "" {
			msg += " (codex stderr: " + tail(s, 500) + ")"
		}
	}
	return GenerateOutput{OK: false, Message: msg}
}

// tail returns the last n bytes of s, marking truncation with a leading ellipsis.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "…" + s[len(s)-n:]
}
