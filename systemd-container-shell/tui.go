package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	ghcrTokenURL = "https://ghcr.io/token?scope=repository:kachick/ubuntu-26.04-nix-systemd:pull"
	ghcrTagsURL  = "https://ghcr.io/v2/kachick/ubuntu-26.04-nix-systemd/tags/list"
)

var (
	errSelectionCanceled = errors.New("selection canceled")
	localImageCandidates = []string{"ubuntu-nix-systemd", "ubuntu-systemd"}
)

type fzfOptions struct {
	prompt    string
	header    string
	delimiter string
	withNth   string
}

type tuiEntry struct {
	action    string // "build" or "pull"
	target    string // local image name or GHCR tag
	age       string
	local     bool
	createdAt time.Time
	rank      int // lower appears first
}

func runTUI(ctx context.Context, e env) error {
	if e.isTerminal != nil && !e.isTerminal() {
		printUsage(e.stderr)
		return errors.New("interactive selector requires a terminal; pass a subcommand (pull or build)")
	}

	client := e.httpClient
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return runTUIWithEndpoints(ctx, client, ghcrTokenURL, ghcrTagsURL, e)
}

func runTUIWithEndpoints(ctx context.Context, client *http.Client, tokenURL, tagsURL string, e env) error {
	now := time.Now()
	if e.now != nil {
		now = e.now()
	}

	entries := collectLocalBuildEntries(e)

	var notice string
	fetchCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	tags, err := fetchGHCRTags(fetchCtx, client, tokenURL, tagsURL)
	cancel()
	if err != nil {
		notice = fmt.Sprintf("Note: failed to fetch GHCR tags (%v)", err)
		if e.stderr != nil {
			fmt.Fprintln(e.stderr, notice)
		}
	} else {
		localRefs := listLocalPodmanRefs(ctx, e)
		entries = append(entries, buildGHCREntries(tags, localRefs, now)...)
	}

	if len(entries) == 0 {
		return errors.New("no container targets available (no local images/ and failed to fetch GHCR tags)")
	}

	header, rows := formatTUITable(entries)
	if notice != "" {
		header = notice + "\n" + header
	}

	picked, err := e.selectItem(ctx, rows, fzfOptions{
		prompt:    "container> ",
		header:    header,
		delimiter: "\t",
		withNth:   "4",
	})
	if err != nil {
		if errors.Is(err, errSelectionCanceled) {
			return nil
		}
		return err
	}

	fields := strings.Split(picked, "\t")
	if len(fields) < 3 {
		return fmt.Errorf("unexpected selection format: %q", picked)
	}
	action, target, localFlag := fields[0], fields[1], fields[2]

	switch action {
	case "build":
		return runBuild(ctx, target, defaultUser, e)
	case "pull":
		// Skip pulling immutable tags that are already cached locally.
		skipPull := localFlag == "1" && target != "latest"
		return runPull(ctx, ":"+target, defaultGHCRImage, defaultUser, skipPull, e)
	default:
		return fmt.Errorf("unknown action %q", action)
	}
}

func collectLocalBuildEntries(e env) []tuiEntry {
	if e.fileExists == nil {
		return nil
	}
	var entries []tuiEntry
	for _, img := range localImageCandidates {
		if e.fileExists(fmt.Sprintf("./images/%s/Containerfile", img)) {
			entries = append(entries, tuiEntry{
				action: "build",
				target: img,
				age:    "local",
				rank:   0,
			})
		}
	}
	return entries
}

func buildGHCREntries(tags []string, localRefs map[string]struct{}, now time.Time) []tuiEntry {
	entries := make([]tuiEntry, 0, len(tags))
	for _, tag := range tags {
		entry := tuiEntry{
			action: "pull",
			target: tag,
			age:    "-",
		}
		if _, ok := localRefs[defaultGHCRImage+":"+tag]; ok {
			entry.local = true
		}

		switch {
		case tag == "latest":
			entry.rank = 1
		case strings.Contains(tag, "utc"):
			entry.rank = 2
			if t, ok := parseUTCTagTime(tag); ok {
				entry.createdAt = t
				entry.age = formatRelativeAge(t, now)
			}
		case strings.HasPrefix(tag, "pr-"):
			entry.rank = 3
		case isFullCommitSHA(tag):
			entry.rank = 5
		default:
			entry.rank = 4
		}

		entries = append(entries, entry)
	}

	slices.SortFunc(entries, func(a, b tuiEntry) int {
		if c := cmp.Compare(a.rank, b.rank); c != 0 {
			return c
		}
		if c := b.createdAt.Compare(a.createdAt); c != 0 {
			return c
		}
		return cmp.Compare(b.target, a.target)
	})
	return entries
}

func formatTUITable(entries []tuiEntry) (header string, rows []string) {
	wKind := len("MODE")
	wTarget := len("TARGET")
	wAge := len("AGE")
	wLocal := len("LOCAL")

	for _, entry := range entries {
		wKind = max(wKind, utf8.RuneCountInString(entry.action))
		wTarget = max(wTarget, utf8.RuneCountInString(entry.target))
		wAge = max(wAge, utf8.RuneCountInString(entry.age))
	}

	header = strings.Join([]string{
		padRight("MODE", wKind),
		padRight("TARGET", wTarget),
		padRight("AGE", wAge),
		padRight("LOCAL", wLocal),
	}, "  ")

	rows = make([]string, len(entries))
	for i, entry := range entries {
		localMark := ""
		localFlag := "0"
		if entry.local {
			localMark = "●"
			localFlag = "1"
		}
		display := strings.Join([]string{
			padRight(entry.action, wKind),
			padRight(entry.target, wTarget),
			padRight(entry.age, wAge),
			padRight(localMark, wLocal),
		}, "  ")
		// Hidden tab-delimited fields let fzf show only the aligned table (--with-nth=4).
		rows[i] = strings.Join([]string{
			sanitizeCell(entry.action),
			sanitizeCell(entry.target),
			localFlag,
			display,
		}, "\t")
	}
	return header, rows
}

func listLocalPodmanRefs(ctx context.Context, e env) map[string]struct{} {
	refs := make(map[string]struct{})
	if e.cmdOutput == nil {
		return refs
	}
	out, err := e.cmdOutput(ctx, "podman", "images", "--format", "{{.Repository}}:{{.Tag}}")
	if err != nil {
		return refs
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			refs[line] = struct{}{}
		}
	}
	return refs
}

func fetchGHCRTags(ctx context.Context, client *http.Client, tokenURL, tagsURL string) ([]string, error) {
	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodGet, tokenURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating token request: %w", err)
	}
	tokenResp, err := client.Do(tokenReq)
	if err != nil {
		return nil, fmt.Errorf("requesting token: %w", err)
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tokenResp.Body)
		return nil, fmt.Errorf("token endpoint returned %d: %s", tokenResp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenData struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	if tokenData.Token == "" {
		return nil, errors.New("token response was empty")
	}

	tagsReq, err := http.NewRequestWithContext(ctx, http.MethodGet, tagsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating tags request: %w", err)
	}
	tagsReq.Header.Set("Authorization", "Bearer "+tokenData.Token)

	tagsResp, err := client.Do(tagsReq)
	if err != nil {
		return nil, fmt.Errorf("requesting tags: %w", err)
	}
	defer tagsResp.Body.Close()

	if tagsResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(tagsResp.Body)
		return nil, fmt.Errorf("tags endpoint returned %d: %s", tagsResp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tagsData struct {
		Tags []string `json:"tags"`
	}
	if err := json.NewDecoder(tagsResp.Body).Decode(&tagsData); err != nil {
		return nil, fmt.Errorf("decoding tags response: %w", err)
	}
	return tagsData.Tags, nil
}

func selectWithFzf(ctx context.Context, items []string, opts fzfOptions, stderr io.Writer) (string, error) {
	fzfBin, err := exec.LookPath("fzf")
	if err != nil {
		return "", fmt.Errorf("finding fzf: %w", err)
	}

	// Keep fzf's default bottom-up layout so the first item stays under the initial cursor.
	var args []string
	if opts.prompt != "" {
		args = append(args, "--prompt="+opts.prompt)
	}
	if opts.header != "" {
		args = append(args, "--header="+opts.header)
	}
	if opts.delimiter != "" {
		args = append(args, "--delimiter="+opts.delimiter)
	}
	if opts.withNth != "" {
		args = append(args, "--with-nth="+opts.withNth)
	}

	cmd := exec.CommandContext(ctx, fzfBin, args...)
	cmd.Stdin = strings.NewReader(strings.Join(items, "\n") + "\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && (exitErr.ExitCode() == 1 || exitErr.ExitCode() == 130) {
			return "", errSelectionCanceled
		}
		return "", fmt.Errorf("fzf failed: %w", err)
	}

	picked := strings.TrimSpace(out.String())
	if picked == "" {
		return "", errSelectionCanceled
	}
	return picked, nil
}

func parseUTCTagTime(tag string) (time.Time, bool) {
	if t, err := time.Parse("20060102-150405-utc", tag); err == nil {
		return t.UTC(), true
	}
	if t, err := time.Parse("20060102-utc", tag); err == nil {
		return t.UTC(), true
	}
	return time.Time{}, false
}

func isFullCommitSHA(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

func formatRelativeAge(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := max(now.Sub(t), 0)
	switch {
	case d < time.Minute:
		return "now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d/time.Minute))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d/time.Hour))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d/(24*time.Hour)))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d/(30*24*time.Hour)))
	default:
		return fmt.Sprintf("%dy ago", int(d/(365*24*time.Hour)))
	}
}

func padRight(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

func sanitizeCell(s string) string {
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
