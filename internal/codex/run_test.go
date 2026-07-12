package codex

import (
	"slices"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	got := buildArgs(RunOpts{Prompt: "a red apple", Workdir: "/tmp/wd", Effort: "low"})
	// NOTE: no --ephemeral — the rollout must be written so the image is reachable.
	want := []string{"exec", "--json", "--skip-git-repo-check",
		"-C", "/tmp/wd", "-c", "model_reasoning_effort=low", "$imagegen a red apple"}
	if !slices.Equal(got, want) {
		t.Fatalf("buildArgs = %v\nwant %v", got, want)
	}
}

func TestParseThreadID(t *testing.T) {
	stdout := []byte(`{"type":"thread.started","thread_id":"019f5721-4c3f-7ac1-977e-6668665ff438"}` + "\n" +
		`{"type":"turn.started"}` + "\n")
	if got := parseThreadID(stdout); got != "019f5721-4c3f-7ac1-977e-6668665ff438" {
		t.Fatalf("parseThreadID = %q", got)
	}
	if got := parseThreadID([]byte(`{"type":"turn.started"}`)); got != "" {
		t.Fatalf("parseThreadID(no thread.started) = %q, want empty", got)
	}
}

func TestBuildArgsModelAndDefaultEffort(t *testing.T) {
	got := buildArgs(RunOpts{Prompt: "x", Workdir: "/w", Model: "gpt-5", Effort: "default"})
	// effort "default" must NOT add a -c flag
	if slices.Contains(got, "model_reasoning_effort=default") {
		t.Fatal("default effort should not be passed")
	}
	if i := slices.Index(got, "-m"); i < 0 || got[i+1] != "gpt-5" {
		t.Fatalf("model flag missing in %v", got)
	}
}
