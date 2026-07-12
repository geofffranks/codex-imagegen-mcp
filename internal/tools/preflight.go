package tools

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type CheckInput struct{}

type CheckOutput struct {
	CodexInstalled bool   `json:"codex_installed"`
	Version        string `json:"version,omitempty"`
	LoggedIn       bool   `json:"logged_in"`
	SessionsDir    string `json:"sessions_dir,omitempty"`
}

func (d *Deps) CheckCodex(ctx context.Context, _ *mcp.CallToolRequest, _ CheckInput) (*mcp.CallToolResult, CheckOutput, error) {
	out := CheckOutput{}
	if _, err := exec.LookPath(d.Cfg.CodexBin); err == nil {
		out.CodexInstalled = true
		if b, err := exec.CommandContext(ctx, d.Cfg.CodexBin, "--version").Output(); err == nil {
			out.Version = strings.TrimSpace(string(b))
		}
	}
	// Report the SAME sessions dir generate_image uses (config already resolved
	// CODEX_HOME / CODEX_IMAGEGEN_SESSIONS_DIR), and look for auth.json in that
	// codex home — otherwise a custom CODEX_HOME would falsely read as logged out.
	codexHome := filepath.Dir(d.Cfg.SessionsDir)
	if _, err := os.Stat(filepath.Join(codexHome, "auth.json")); err == nil {
		out.LoggedIn = true
	}
	if _, err := os.Stat(d.Cfg.SessionsDir); err == nil {
		out.SessionsDir = d.Cfg.SessionsDir
	}
	return nil, out, nil
}
