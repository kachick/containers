package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

const (
	defaultUser       = "visitor"
	defaultLocalImage = "ubuntu-nix-systemd"
	defaultGHCRImage  = "ghcr.io/kachick/ubuntu-26.04-nix-systemd"
	baseImageTag      = "localhost/ubuntu-systemd:26.04"
	baseContainerfile = "./images/ubuntu-systemd/Containerfile"
)

type env struct {
	stdin      io.Reader
	stdout     io.Writer
	stderr     io.Writer
	runCmd     func(ctx context.Context, name string, args ...string) error
	cmdOutput  func(ctx context.Context, name string, args ...string) ([]byte, error)
	cmdExitOK  func(ctx context.Context, name string, args ...string) (bool, error)
	selectItem func(ctx context.Context, items []string, opts fzfOptions) (string, error)
	isTerminal func() bool
	fileExists func(path string) bool
	sleep      func(ctx context.Context, d time.Duration) error
	now        func() time.Time
	httpClient *http.Client
}

func isCharDevice(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func defaultEnv() env {
	return env{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		runCmd: func(ctx context.Context, name string, args ...string) error {
			cmd := exec.CommandContext(ctx, name, args...)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			return cmd.Run()
		},
		cmdOutput: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			return cmd.Output()
		},
		cmdExitOK: func(ctx context.Context, name string, args ...string) (bool, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			err := cmd.Run()
			if err == nil {
				return true, nil
			}
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				return false, nil
			}
			return false, err
		},
		selectItem: func(ctx context.Context, items []string, opts fzfOptions) (string, error) {
			return selectWithFzf(ctx, items, opts, os.Stderr)
		},
		isTerminal: func() bool {
			return isCharDevice(os.Stdin) && isCharDevice(os.Stderr)
		},
		fileExists: func(path string) bool {
			_, err := os.Stat(path)
			return err == nil
		},
		sleep: func(ctx context.Context, d time.Duration) error {
			timer := time.NewTimer(d)
			defer timer.Stop()
			select {
			case <-timer.C:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		},
		now:        time.Now,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, os.Args[1:], defaultEnv()); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, e env) error {
	if len(args) == 0 {
		return runTUI(ctx, e)
	}

	switch args[0] {
	case "-h", "--help", "help":
		printUsage(e.stdout)
		return nil
	case "build":
		return runBuildCmd(ctx, args[1:], e)
	case "pull":
		return runPullCmd(ctx, args[1:], e)
	default:
		printUsage(e.stderr)
		return fmt.Errorf("unknown subcommand %q", args[0])
	}
}

func printUsage(w io.Writer) {
	if w == nil {
		return
	}
	fmt.Fprint(w, `systemd-container-shell - Run and enter a systemd container shell

Usage:
  systemd-container-shell                          Open interactive selector (fzf)
  systemd-container-shell pull [flags] [reference] Pull an image from a registry and run it
  systemd-container-shell build [flags] [image]    Build a local image and run it

Run 'systemd-container-shell <subcommand> --help' for subcommand flags.
`)
}

func runBuildCmd(ctx context.Context, args []string, e env) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	user := fs.String("user", defaultUser, "Username for the container")
	fs.Usage = func() {
		fmt.Fprint(e.stderr, "Usage: systemd-container-shell build [--user <name>] [image]\n\nFlags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("unexpected extra arguments: %v", fs.Args()[1:])
	}

	image := defaultLocalImage
	if fs.NArg() == 1 {
		image = fs.Arg(0)
	}
	return runBuild(ctx, image, *user, e)
}

func runPullCmd(ctx context.Context, args []string, e env) error {
	fs := flag.NewFlagSet("pull", flag.ContinueOnError)
	fs.SetOutput(e.stderr)
	image := fs.String("image", defaultGHCRImage, "Override default image path (registry + package)")
	fs.StringVar(image, "i", defaultGHCRImage, "Shorthand for --image")
	user := fs.String("user", defaultUser, "Username for the container")
	skipPull := fs.Bool("skip-pull", false, "Use local image without pulling; fail if missing")
	fs.Usage = func() {
		fmt.Fprint(e.stderr, "Usage: systemd-container-shell pull [flags] [reference]\n\nFlags:\n")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 1 {
		return fmt.Errorf("unexpected extra arguments: %v", fs.Args()[1:])
	}

	reference := ":latest"
	if fs.NArg() == 1 {
		reference = fs.Arg(0)
	}
	return runPull(ctx, reference, *image, *user, *skipPull, e)
}

func resolveImageRef(reference, defaultImage string) string {
	if strings.HasPrefix(reference, ":") {
		return defaultImage + reference
	}
	if strings.Contains(reference, "/") {
		if strings.Contains(reference, ":") {
			return reference
		}
		return reference + ":latest"
	}
	return defaultImage + ":" + reference
}

func buildImage(ctx context.Context, tag, containerfile string, buildArgs []string, e env) error {
	args := []string{
		"build",
		"--tag", tag,
		"--file", containerfile,
	}
	for _, arg := range buildArgs {
		args = append(args, "--build-arg", arg)
	}
	args = append(args, ".")

	if err := e.runCmd(ctx, "podman", args...); err != nil {
		return fmt.Errorf("podman build failed for %s: %w", tag, err)
	}
	return nil
}

func runBuild(ctx context.Context, image, user string, e env) error {
	fmt.Fprintf(e.stdout, "Building image: %s for user: %s\n", image, user)

	buildArgs := []string{"username=" + user}
	if image == defaultLocalImage {
		fmt.Fprintf(e.stdout, "Building base image: %s\n", baseImageTag)
		if err := buildImage(ctx, baseImageTag, baseContainerfile, buildArgs, e); err != nil {
			return err
		}
		buildArgs = []string{"BASE_IMAGE=" + baseImageTag}
	}

	containerfile := fmt.Sprintf("./images/%s/Containerfile", image)
	if err := buildImage(ctx, image, containerfile, buildArgs, e); err != nil {
		return err
	}

	return startAndEnterContainer(ctx, image, user, e)
}

func runPull(ctx context.Context, reference, defaultImage, user string, skipPull bool, e env) error {
	fullImage := resolveImageRef(reference, defaultImage)

	if skipPull {
		exists, err := imageExists(ctx, fullImage, e)
		if err != nil {
			return fmt.Errorf("checking local image %s: %w", fullImage, err)
		}
		if !exists {
			return fmt.Errorf("image %s not found locally and --skip-pull specified", fullImage)
		}
		fmt.Fprintf(e.stdout, "Image %s found locally, skipping pull\n", fullImage)
	} else {
		fmt.Fprintf(e.stdout, "Pulling image: %s\n", fullImage)
		if err := e.runCmd(ctx, "podman", "pull", fullImage); err != nil {
			return fmt.Errorf("podman pull failed: %w", err)
		}
	}

	return startAndEnterContainer(ctx, fullImage, user, e)
}

func imageExists(ctx context.Context, image string, e env) (bool, error) {
	return e.cmdExitOK(ctx, "podman", "image", "exists", image)
}

func startAndEnterContainer(ctx context.Context, imageID, user string, e env) error {
	fmt.Fprintln(e.stdout, "Starting container...")
	out, err := e.cmdOutput(ctx, "podman", "run", "--rm", "--detach", imageID)
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
			return fmt.Errorf("podman run failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return fmt.Errorf("podman run failed: %w", err)
	}

	containerID := strings.TrimSpace(string(out))
	if containerID == "" {
		return errors.New("podman run returned an empty container ID")
	}
	fmt.Fprintf(e.stdout, "Container started: %s\n", containerID)

	defer func() {
		fmt.Fprintf(e.stdout, "Stopping container %s...\n", containerID)
		// Use a fresh context so cleanup still runs when interrupted by a signal.
		stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = e.runCmd(stopCtx, "podman", "stop", containerID)
	}()

	// Wait briefly for systemd inside the container to initialize.
	if err := e.sleep(ctx, time.Second); err != nil {
		return err
	}

	fmt.Fprintln(e.stdout, "Entering container shell...")
	err = e.runCmd(
		ctx,
		"podman",
		"exec",
		"--user", user,
		"--workdir", "/home/"+user,
		"--interactive",
		"--tty",
		containerID,
		"bash",
	)
	if err != nil {
		return fmt.Errorf("podman exec session failed: %w", err)
	}
	return nil
}
