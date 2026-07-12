package tools

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"codex-imagegen-mcp/internal/manifest"
)

type RecentInput struct {
	Limit int `json:"limit,omitempty" jsonschema:"max number of recent images to return (default 10)"`
}

type RecentOutput struct {
	Images []manifest.Record `json:"images"`
}

func (d *Deps) ListRecent(_ context.Context, _ *mcp.CallToolRequest, in RecentInput) (*mcp.CallToolResult, RecentOutput, error) {
	limit := in.Limit
	if limit <= 0 {
		limit = 10
	}
	recs, err := manifest.ReadRecent(d.Cfg.ManifestPath, limit)
	if err != nil {
		return nil, RecentOutput{}, err
	}
	if recs == nil {
		recs = []manifest.Record{}
	}
	return nil, RecentOutput{Images: recs}, nil
}
