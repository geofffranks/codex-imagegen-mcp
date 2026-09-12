# codex-imagegen-mcp — agent rules

## Changing this MCP: build → install → reconnect → verify (required final steps)

The Ratatoskr gateway launches the **installed** binary
`/Users/gfranks/go/bin/codex-imagegen-mcp` (see `mcpClients["codex-imagegen"].command`
in the Mac host's `~/Library/Preferences/ratatoskr/config.json`) as a
long-lived stdio process. Code changes are invisible to agents until the
installed binary is replaced and the gateway re-execs it.

1. Implement and test: `go vet ./... && go test ./...`.
2. Build and install: `go build -o /Users/gfranks/go/bin/codex-imagegen-mcp .`
   — the repo-root build alone changes nothing for the gateway.
3. Reconnect: reloading the config does **not** re-exec an unchanged stdio
   upstream — the gateway keeps serving the previously spawned process. Use
   ratatoskr's `reconnect-upstream` operation (upstream Lua name:
   `codex_imagegen`), or restart the gateway.
4. Verify live before reporting the change as deployed: `list-server-tools`
   for `codex_imagegen` must show the new tool surface; probe one cheap call
   if behavior changed.
