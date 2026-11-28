package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestResolveImageRef(t *testing.T) {
	tests := []struct {
		reference    string
		defaultImage string
		want         string
	}{
		{":latest", defaultGHCRImage, defaultGHCRImage + ":latest"},
		{":20260618-072235-utc", defaultGHCRImage, defaultGHCRImage + ":20260618-072235-utc"},
		{"20260618-072235-utc", defaultGHCRImage, defaultGHCRImage + ":20260618-072235-utc"},
		{"ghcr.io/example/img", defaultGHCRImage, "ghcr.io/example/img:latest"},
		{"ghcr.io/example/img:v1", defaultGHCRImage, "ghcr.io/example/img:v1"},
	}

	for _, tt := range tests {
		if got := resolveImageRef(tt.reference, tt.defaultImage); got != tt.want {
			t.Errorf("resolveImageRef(%q, %q) = %q, want %q", tt.reference, tt.defaultImage, got, tt.want)
		}
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	e := env{stdout: &stdout, stderr: &stderr}

	if err := run(context.Background(), []string{"--help"}, e); err != nil {
		t.Fatalf("run(--help) returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("expected Usage in stdout, got %q", stdout.String())
	}
}

func TestRunBuildDefaultImage(t *testing.T) {
	var calls [][]string
	var stdout, stderr bytes.Buffer

	e := env{
		stdout: &stdout,
		stderr: &stderr,
		runCmd: func(_ context.Context, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			return nil
		},
		cmdOutput: func(_ context.Context, name string, args ...string) ([]byte, error) {
			calls = append(calls, append([]string{name}, args...))
			return []byte("cid-123\n"), nil
		},
		sleep: func(_ context.Context, _ time.Duration) error { return nil },
	}

	if err := run(context.Background(), []string{"build"}, e); err != nil {
		t.Fatalf("run(build) failed: %v", err)
	}

	want := [][]string{
		{"podman", "build", "--tag", baseImageTag, "--file", baseContainerfile, "--build-arg", "username=visitor", "."},
		{"podman", "build", "--tag", "ubuntu-nix-systemd", "--file", "./images/ubuntu-nix-systemd/Containerfile", "--build-arg", "BASE_IMAGE=" + baseImageTag, "."},
		{"podman", "run", "--rm", "--detach", "ubuntu-nix-systemd"},
		{"podman", "exec", "--user", "visitor", "--workdir", "/home/visitor", "--interactive", "--tty", "cid-123", "bash"},
		{"podman", "stop", "cid-123"},
	}

	if len(calls) != len(want) {
		t.Fatalf("got %d calls %v, want %d calls %v", len(calls), calls, len(want), want)
	}
	for i := range want {
		if !slices.Equal(calls[i], want[i]) {
			t.Errorf("call[%d] = %v, want %v", i, calls[i], want[i])
		}
	}
}

func TestRunPullSkipPullMissing(t *testing.T) {
	var stdout, stderr bytes.Buffer
	e := env{
		stdout: &stdout,
		stderr: &stderr,
		cmdExitOK: func(_ context.Context, _ string, _ ...string) (bool, error) {
			return false, nil
		},
	}

	err := run(context.Background(), []string{"pull", "--skip-pull", ":latest"}, e)
	if err == nil || !strings.Contains(err.Error(), "not found locally") {
		t.Fatalf("expected missing local image error, got %v", err)
	}
}

func TestRunTUIRequiresTerminal(t *testing.T) {
	var stdout, stderr bytes.Buffer
	e := env{
		stdout:     &stdout,
		stderr:     &stderr,
		isTerminal: func() bool { return false },
	}

	err := run(context.Background(), nil, e)
	if err == nil || !strings.Contains(err.Error(), "requires a terminal") {
		t.Fatalf("expected terminal error, got %v", err)
	}
}

func TestRunTUISelectsCachedGHCRTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"test-token"}`))
		case "/tags":
			if r.Header.Get("Authorization") != "Bearer test-token" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tags":["20260618-072235-utc","latest","20260619-072235-utc","pr-10-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	var (
		capturedItems []string
		calls         [][]string
		stdout        bytes.Buffer
		stderr        bytes.Buffer
	)

	fixedNow := time.Date(2026, 6, 19, 10, 22, 35, 0, time.UTC)

	e := env{
		stdout:     &stdout,
		stderr:     &stderr,
		isTerminal: func() bool { return true },
		now:        func() time.Time { return fixedNow },
		fileExists: func(path string) bool {
			return path == "./images/ubuntu-nix-systemd/Containerfile"
		},
		cmdOutput: func(_ context.Context, name string, args ...string) ([]byte, error) {
			if name == "podman" && len(args) > 0 && args[0] == "images" {
				return []byte(defaultGHCRImage + ":20260619-072235-utc\n"), nil
			}
			calls = append(calls, append([]string{name}, args...))
			return []byte("cid-tui\n"), nil
		},
		cmdExitOK: func(_ context.Context, _ string, _ ...string) (bool, error) {
			return true, nil
		},
		runCmd: func(_ context.Context, name string, args ...string) error {
			calls = append(calls, append([]string{name}, args...))
			return nil
		},
		selectItem: func(_ context.Context, items []string, _ fzfOptions) (string, error) {
			capturedItems = items
			// Pick the cached timestamp tag (rank order: build, latest, 20260619, 20260618, pr-10).
			return items[2], nil
		},
		sleep: func(_ context.Context, _ time.Duration) error { return nil },
	}

	if err := runTUIWithEndpoints(context.Background(), srv.Client(), srv.URL+"/token", srv.URL+"/tags", e); err != nil {
		t.Fatalf("runTUIWithEndpoints failed: %v", err)
	}

	if len(capturedItems) != 5 {
		t.Fatalf("expected 5 items, got %d: %v", len(capturedItems), capturedItems)
	}
	if !strings.HasPrefix(capturedItems[0], "build\tubuntu-nix-systemd\t") {
		t.Errorf("expected local build first, got %q", capturedItems[0])
	}
	if !strings.HasPrefix(capturedItems[1], "pull\tlatest\t") {
		t.Errorf("expected latest second, got %q", capturedItems[1])
	}
	if !strings.HasPrefix(capturedItems[2], "pull\t20260619-072235-utc\t1\t") {
		t.Errorf("expected newest cached UTC tag third, got %q", capturedItems[2])
	}

	// Cached immutable tag must skip `podman pull` and jump straight to `podman run`.
	for _, call := range calls {
		if len(call) >= 2 && call[0] == "podman" && call[1] == "pull" {
			t.Errorf("expected podman pull to be skipped for cached immutable tag, got calls: %v", calls)
		}
	}
}

func TestRunTUICanceledIsNoError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	e := env{
		stdout:     &stdout,
		stderr:     &stderr,
		isTerminal: func() bool { return true },
		fileExists: func(path string) bool {
			return path == "./images/ubuntu-systemd/Containerfile"
		},
		selectItem: func(_ context.Context, _ []string, _ fzfOptions) (string, error) {
			return "", errSelectionCanceled
		},
	}

	// Invalid endpoint triggers fallback to local build candidate only.
	if err := runTUIWithEndpoints(context.Background(), &http.Client{}, "http://127.0.0.1:0/token", "http://127.0.0.1:0/tags", e); err != nil {
		t.Fatalf("expected nil on selection cancel, got %v", err)
	}
}
