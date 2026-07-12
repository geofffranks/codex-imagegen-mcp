# Test fixtures — codex imagegen spike findings

Captured 2026-07-12 against **codex-cli 0.144.1** by running:

```
codex exec --json --skip-git-repo-check -C <workdir> \
  -c model_reasoning_effort=low '$imagegen a plain red apple on white, centered' < /dev/null
```

## Key findings (these drove the extraction design)

1. **stdin must be `/dev/null`.** `codex exec` prints `Reading additional input
   from stdin...` and **blocks forever** if stdin is an open pipe/TTY with no EOF.
   Go's `os/exec` connects a nil `Cmd.Stdin` to `/dev/null` automatically, so the
   Go runner is safe — but it sets `Stdin` explicitly and documents why.

2. **`--json` stdout does NOT contain the image.** The `--json` stream is a
   high-level event API: `thread.started`, `turn.started`, `item.completed`
   (only `agent_message` items), `turn.completed`. No base64, no image path.

3. **The image bytes live ONLY in the session rollout**, so **`--ephemeral` must
   NOT be used** (it suppresses the rollout, making the image unreachable).

4. **`thread.started.thread_id` equals the rollout filename UUID.** Example:
   `thread_id = 019f5721-4c3f-7ac1-977e-6668665ff438` →
   `~/.codex/sessions/2026/07/12/rollout-2026-07-12T12-20-27-019f5721-4c3f-7ac1-977e-6668665ff438.jsonl`.
   So we parse `thread_id` from `--json` stdout and locate the rollout by it
   (recursive walk of the date-nested sessions dir), with "newest rollout modified
   since process start" as a fallback.

5. **In the rollout, the base64 PNG is at `payload.result` on the
   `image_generation_end` event** (~1,384,604 bytes for a ~1254×1254 PNG). The
   field-agnostic PNG-magic deep scan finds exactly this one hit, so it works
   without hard-coding the event type — but the confirmed type is
   `image_generation_end`.

## `rollout-with-image.jsonl`

A trimmed rollout that preserves the real envelope shape (`session_meta`,
`custom_tool_call`, `agent_message`, and the `image_generation_end` line) but with
the 1.3MB base64 in `payload.result` swapped for a 67-byte 1×1 PNG, so the
extractor test stays tiny. `ExtractPNG` must return a 67-byte PNG from this file.
