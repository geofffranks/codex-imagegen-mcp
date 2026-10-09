# codex-imagegen-mcp — agent rules

## Changing this MCP: build → install → reconnect → verify

Develop and run checks directly on the operator's Mac. The gateway has previously
been documented as launching `/Users/gfranks/go/bin/codex-imagegen-mcp`; confirm
the active command, arguments, environment and timeouts in the Mac's
`~/Library/Preferences/ratatoskr/config.json` (`mcpClients["codex-imagegen"]`)
before building or deploying. The checked-in destination is not proof of the
current Mac configuration. Use the exact native artifact path named by that
configuration, commonly an existing sibling-repository `bin/` directory only
when the active command confirms it.

1. Implement and test on the Mac: `go vet ./... && go test ./...`.
2. Build the native Mac binary into a staging path, inspect the artifact and
   install it only at the verified active command path. A repo-root build alone
   does not update an installed command. Preserve existing dependencies.
3. Before reconnecting, assess all pending Ratatoskr config changes, eligible
   `NeedsLogin` peers, affected services and active owners. Coordinate with every
   affected owner and establish recovery. If host config or impact cannot be
   inspected, defer reconnect/reload for operator coordination.
4. After authorized installation, reconnect the `codex_imagegen` upstream so
   the gateway re-executes the unchanged stdio definition. A config reload alone
   does not replace the running process. Do not restart the gateway as a shortcut.
5. Verify the live tool surface with `list-server-tools` for `codex_imagegen`;
   probe one cheap call if behavior changed. Report source changes, checks,
   artifact installation, reconnect and live behavior as separate facts.
