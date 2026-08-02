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
	Prompt          string   `json:"prompt" jsonschema:"the image description to generate"`
	Out             string   `json:"out" jsonschema:"file path to write the PNG to (absolute, or relative to the server's working directory)"`
	Model           string   `json:"model,omitempty" jsonschema:"optional codex model override"`
	Effort          string   `json:"effort,omitempty" jsonschema:"reasoning effort: minimal, low, default, medium, high, or xhigh"`
	Workdir         string   `json:"workdir,omitempty" jsonschema:"optional codex working directory; defaults to a temp dir"`
	ReferenceImages []string `json:"reference_images,omitempty" jsonschema:"optional local file paths of reference images to attach as vision input"`
	AspectRatio     string   `json:"aspect_ratio,omitempty" jsonschema:"optional best-effort aspect ratio hint, e.g. 16:9, 1:1, 4:3"`
}

type GenerateOutput struct {
	OK              bool     `json:"ok"`
	Path            string   `json:"path,omitempty"`
	Width           int      `json:"width,omitempty"`
	Height          int      `json:"height,omitempty"`
	Bytes           int      `json:"bytes,omitempty"`
	Message         string   `json:"message,omitempty"`
	ReferenceImages []string `json:"reference_images,omitempty"`
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

// buildPrompt assembles the final prompt sent to codex by appending minimal,
// non-conflicting context for reference images and aspect ratio.
func buildPrompt(prompt string, refs []string, aspectRatio string) string {
	out := prompt
	if len(refs) > 0 {
		out += "\n\nReference images are attached. Use them for style, composition, or subject guidance. Do not edit or modify them."
	}
	if aspectRatio != "" {
		out += "\n" + aspectRatioHint(aspectRatio)
	}
	return out
}

// aspectRatioHint maps common ratios to descriptive framing language; unknown
// ratios get a generic mention. This is a best-effort prompt hint — the built-in
// image_gen tool does not accept an explicit size parameter.
func aspectRatioHint(ratio string) string {
	switch ratio {
	case "16:9":
		return "Output in a wide 16:9 landscape aspect ratio."
	case "9:16":
		return "Output in a tall 9:16 portrait aspect ratio."
	case "1:1":
		return "Output in a square 1:1 aspect ratio."
	case "4:3":
		return "Output in a 4:3 landscape aspect ratio."
	case "3:4":
		return "Output in a 3:4 portrait aspect ratio."
	case "3:2":
		return "Output in a 3:2 landscape aspect ratio."
	case "2:3":
		return "Output in a 2:3 portrait aspect ratio."
	default:
		return "Output in a " + ratio + " aspect ratio."
	}
}

func (d *Deps) GenerateImage(ctx context.Context, _ *mcp.CallToolRequest, in GenerateInput) (*mcp.CallToolResult, GenerateOutput, error) {
	if in.Prompt == "" || in.Out == "" {
		return nil, GenerateOutput{}, fmt.Errorf("both 'prompt' and 'out' are required")
	}

	for _, ref := range in.ReferenceImages {
		if _, err := os.Stat(ref); err != nil {
			return nil, GenerateOutput{}, fmt.Errorf("reference image not found: %s", ref)
		}
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
		Bin: d.Cfg.CodexBin, Prompt: buildPrompt(in.Prompt, in.ReferenceImages, in.AspectRatio), Model: model,
		Effort: effort, Workdir: workdir, Images: in.ReferenceImages, Timeout: d.Cfg.Timeout,
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
			out := noImage(res, "could not locate codex output: "+imageErr.Error()+"; "+rolloutErr.Error())
			out.ReferenceImages = in.ReferenceImages
			return nil, out, nil
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
		out := noImage(res, "codex returned no image")
		out.ReferenceImages = in.ReferenceImages
		return nil, out, nil
	}

	w, h, n, err := writePNG(in.Out, png)
	if err != nil {
		return nil, GenerateOutput{}, err
	}

	_ = manifest.Append(d.Cfg.ManifestPath, manifest.Record{
		Path: in.Out, Prompt: in.Prompt, Timestamp: d.Now(),
		Width: w, Height: h, Bytes: n,
		ReferenceImages: in.ReferenceImages, AspectRatio: in.AspectRatio,
	})

	return nil, GenerateOutput{OK: true, Path: in.Out, Width: w, Height: h, Bytes: n, ReferenceImages: in.ReferenceImages}, nil
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
