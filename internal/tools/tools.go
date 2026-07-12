package tools

import (
	"time"

	"codex-imagegen-mcp/internal/config"
)

type Deps struct {
	Cfg config.Config
	Now func() string
}

func New(cfg config.Config) *Deps {
	return &Deps{Cfg: cfg, Now: func() string { return time.Now().UTC().Format(time.RFC3339) }}
}
