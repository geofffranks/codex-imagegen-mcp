package codex

import (
	"io"
	"reflect"
	"slices"
	"testing"
)

func TestBuildArgs(t *testing.T) {
	got := buildArgs(RunOpts{Prompt: "a red apple", Workdir: "/tmp/wd", Effort: "low"})
	// NOTE: no --ephemeral — the rollout must be written so the image is reachable.
	want := []string{"exec", "--json", "--skip-git-repo-check",
		"-C", "/tmp/wd", "-c", "model_reasoning_effort=low", "--", "$imagegen a red apple"}
	if !slices.Equal(got, want) {
		t.Fatalf("buildArgs = %v\nwant %v", got, want)
	}
}

func TestBuildArgsAddsDangerousSandboxBypassWhenEnabled(t *testing.T) {
	got := buildArgs(RunOpts{Prompt: "a red apple", Workdir: "/tmp/wd", DangerouslyBypassSandbox: true})
	want := []string{"exec", "--json", "--skip-git-repo-check", "-C", "/tmp/wd",
		"--dangerously-bypass-approvals-and-sandbox", "--", "$imagegen a red apple"}
	if !slices.Equal(got, want) {
		t.Fatalf("buildArgs = %v\nwant %v", got, want)
	}
}

func TestBuildArgsOmitsDangerousSandboxBypassWhenDisabled(t *testing.T) {
	got := buildArgs(RunOpts{Prompt: "a red apple", Workdir: "/tmp/wd"})
	if slices.Contains(got, "--dangerously-bypass-approvals-and-sandbox") {
		t.Fatalf("buildArgs = %v, did not want dangerous sandbox bypass flag", got)
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

func TestBuildArgsMapsMinimalEffortToLow(t *testing.T) {
	got := buildArgs(RunOpts{Prompt: "x", Workdir: "/w", Effort: "minimal"})
	if !slices.Contains(got, "model_reasoning_effort=low") {
		t.Fatalf("minimal effort should map to low: %v", got)
	}
	if slices.Contains(got, "model_reasoning_effort=minimal") {
		t.Fatalf("unsupported minimal effort was passed through: %v", got)
	}
}

func TestBuildArgsImagesViaDashI(t *testing.T) {
	got := buildArgs(RunOpts{
		Prompt: "a cat", Workdir: "/w", Images: []string{"/r1.png", "/r2.png"},
		OutputLastMessage: "/tmp/last-message.txt",
	})
	want := []string{"exec", "--json", "--skip-git-repo-check", "-C", "/w",
		"-i", "/r1.png", "-i", "/r2.png", "-o", "/tmp/last-message.txt", "--", "$imagegen a cat"}
	if !slices.Equal(got, want) {
		t.Fatalf("buildArgs = %v\nwant %v", got, want)
	}
}

func TestBuildArgsNoImagesNoDashI(t *testing.T) {
	got := buildArgs(RunOpts{Prompt: "x", Workdir: "/w"})
	if slices.Contains(got, "-i") {
		t.Fatalf("expected no -i flags without images: %v", got)
	}
}

func TestEmptyStdinIsExplicitEOF(t *testing.T) {
	stdin := emptyStdin()
	if stdin == nil {
		t.Fatal("empty stdin must be an explicit reader")
	}
	if got, err := stdin.Read(make([]byte, 1)); got != 0 || err != io.EOF {
		t.Fatalf("empty reader should return EOF, got n=%d err=%v", got, err)
	}
}

func TestBuildArgsModelAndDefaultEffort(t *testing.T) {
	got := buildArgs(RunOpts{Prompt: "x", Workdir: "/w", Effort: "default"})
	// effort "default" must NOT add a -c flag
	if slices.Contains(got, "model_reasoning_effort=default") {
		t.Fatal("default effort should not be passed")
	}
	if slices.Contains(got, "-m") {
		t.Fatalf("model selection must never be passed: %v", got)
	}
}

func TestRunOptsHasNoModelSelection(t *testing.T) {
	if _, ok := reflect.TypeOf(RunOpts{}).FieldByName("Model"); ok {
		t.Fatal("RunOpts must not expose model selection")
	}
}
