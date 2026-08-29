package ipatool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/logger"
)

var runnerLog = logger.WithComponent("ipatool")

type Runner struct {
	Bin        string
	Passphrase string
}

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

func (r Runner) Run(ctx context.Context, homeDir string, args ...string) (Result, error) {
	return r.run(ctx, homeDir, false, args...)
}

func (r Runner) run(ctx context.Context, homeDir string, verbose bool, args ...string) (Result, error) {
	if r.Bin == "" {
		r.Bin = "ipatool"
	}
	if homeDir == "" {
		return Result{}, fmt.Errorf("home dir required")
	}
	if err := os.MkdirAll(homeDir, 0o700); err != nil {
		return Result{}, err
	}

	global := []string{
		"--non-interactive",
		"--keychain-passphrase", r.Passphrase,
		"--keyring-backend", "file",
		"--format", "json",
	}
	if verbose {
		global = append(global, "--verbose")
	}
	full := append(global, args...)

	cmd := exec.CommandContext(ctx, r.Bin, full...)
	env := append(os.Environ(), "HOME="+homeDir)
	// On Linux, keyring may try SecretService over D-Bus; force it unavailable so
	// --keyring-backend file is used. Unnecessary on macOS/Windows.
	if runtime.GOOS == "linux" {
		env = append(env, "DBUS_SESSION_BUS_ADDRESS=unix:path=/nonexistent")
	}
	cmd.Env = env
	runnerLog.WithFields(map[string]any{
		"bin":     r.Bin,
		"home":    homeDir,
		"verbose": verbose,
		"args":    redactArgs(args),
	}).Debug("executing command")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	res := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
		} else {
			res.ExitCode = -1
			return res, err
		}
		runnerLog.WithFields(map[string]any{
			"exit_code": res.ExitCode,
			"args":      redactArgs(args),
		}).Warn("command failed")
	}
	return res, nil
}

func redactArgs(args []string) string {
	out := make([]string, len(args))
	copy(out, args)
	for i := 0; i < len(out)-1; i++ {
		switch out[i] {
		case "--password", "-p", "--auth-code":
			out[i+1] = "***"
		}
	}
	return strings.Join(out, " ")
}

func (r Runner) AuthInfo(ctx context.Context, homeDir string) (Result, error) {
	return r.Run(ctx, homeDir, "auth", "info")
}

func (r Runner) AuthLogin(ctx context.Context, homeDir, email, password, authCode string) (Result, error) {
	args := []string{"auth", "login", "--email", email, "--password", password}
	if authCode != "" {
		args = append(args, "--auth-code", authCode)
	}
	// Verbose: surface SAP / Apple auth detail in FailureMessage.
	return r.run(ctx, homeDir, true, args...)
}

func (r Runner) AuthRevoke(ctx context.Context, homeDir string) (Result, error) {
	return r.Run(ctx, homeDir, "auth", "revoke")
}

func (r Runner) Purchase(ctx context.Context, homeDir, bundleID string) (Result, error) {
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return Result{}, fmt.Errorf("bundle id required for purchase")
	}
	return r.Run(ctx, homeDir, "purchase", "--bundle-identifier", bundleID)
}

func (r Runner) Download(ctx context.Context, homeDir, bundleID string, appID int64, output, externalVersionID string, purchase bool) (Result, error) {
	args := []string{"download", "-o", output}
	if bundleID != "" {
		args = append(args, "--bundle-identifier", bundleID)
	}
	if appID > 0 {
		args = append(args, "--app-id", fmt.Sprintf("%d", appID))
	}
	if externalVersionID != "" {
		args = append(args, "--external-version-id", externalVersionID)
	}
	if purchase {
		args = append(args, "--purchase")
	}
	return r.Run(ctx, homeDir, args...)
}

func (r Runner) ListVersions(ctx context.Context, homeDir, bundleID string, appID int64) (Result, error) {
	args := []string{"list-versions"}
	if bundleID != "" {
		args = append(args, "--bundle-identifier", bundleID)
	}
	if appID > 0 {
		args = append(args, "--app-id", fmt.Sprintf("%d", appID))
	}
	return r.Run(ctx, homeDir, args...)
}

func (r Runner) GetVersionMetadata(ctx context.Context, homeDir, bundleID string, appID int64, externalVersionID string) (Result, error) {
	args := []string{"get-version-metadata", "--external-version-id", externalVersionID}
	if bundleID != "" {
		args = append(args, "--bundle-identifier", bundleID)
	}
	if appID > 0 {
		args = append(args, "--app-id", fmt.Sprintf("%d", appID))
	}
	return r.Run(ctx, homeDir, args...)
}

func (r Runner) Search(ctx context.Context, homeDir, term string, limit int) (Result, error) {
	if limit <= 0 {
		limit = 5
	}
	return r.Run(ctx, homeDir, "search", term, "--limit", fmt.Sprintf("%d", limit))
}

func WithTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, d)
}

func CombinedOutput(res Result) string {
	parts := []string{strings.TrimSpace(res.Stdout), strings.TrimSpace(res.Stderr)}
	out := make([]string, 0, 2)
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, "\n")
}

func ResultMessage(res Result, err error) string {
	if msg := FailureMessage(res.Stdout); msg != "" {
		return msg
	}
	if msg := strings.TrimSpace(res.Stderr); msg != "" {
		return msg
	}
	if msg := strings.TrimSpace(res.Stdout); msg != "" {
		return msg
	}
	if err != nil {
		return err.Error()
	}
	return ""
}
