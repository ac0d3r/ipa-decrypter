// Package fairplay decrypts FairPlay-encrypted IPAs with a 16-byte
// account credential. It is a Go port of fairplay-decrypt's core.py and
// decrypt.py; the table-backed block transform keeps running as the
// external C helper (`transform --batch const-data.bin`), exactly like
// the Python BlockTransform class does.
package fairplay

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// BlockTransform drives the external `transform` helper in batch mode.
// It is not safe for concurrent use; Run serializes access internally.
type BlockTransform struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr *bytes.Buffer
}

// NewBlockTransform spawns the transform helper in batch mode. The
// executable and constant-data file must exist, mirroring the Python
// implementation's FileNotFoundError.
func NewBlockTransform(executable, constants string) (*BlockTransform, error) {
	if executable == "" {
		return nil, fmt.Errorf("transform executable path required")
	}
	if constants == "" {
		return nil, fmt.Errorf("transform constant-data path required")
	}
	if !isFile(executable) || !isFile(constants) {
		return nil, fmt.Errorf("build transform executable and keep constant-data file")
	}

	cmd := exec.Command(executable, "--batch", constants)
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("transform stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("transform stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start transform: %w", err)
	}
	return &BlockTransform{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		stderr: stderr,
	}, nil
}

// Close shuts the helper down and waits for it to exit.
func (b *BlockTransform) Close() error {
	if b == nil || b.cmd == nil {
		return nil
	}
	_ = b.stdin.Close()
	done := make(chan error, 1)
	go func() { done <- b.cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(30 * time.Second):
		_ = b.cmd.Process.Kill()
		waitErr := <-done
		if waitErr != nil {
			return fmt.Errorf("transform did not exit within 30s: %w", waitErr)
		}
		return fmt.Errorf("transform did not exit within 30s")
	}
}

// Run transforms one 16-byte source block with the 16-byte work key and
// returns the 16-byte output.
func (b *BlockTransform) Run(source, work []byte) ([]byte, error) {
	if len(source) != 16 || len(work) != 16 {
		return nil, fmt.Errorf("block-transform inputs require 16 bytes")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.cmd == nil || b.cmd.Process == nil {
		return nil, fmt.Errorf("block transform pipes unavailable")
	}
	if _, err := fmt.Fprintf(b.stdin, "%x %x %x\n", source, work, constant); err != nil {
		return nil, fmt.Errorf("write transform request: %w (stderr: %s)", err, b.stderr.String())
	}
	line, err := b.stdout.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("block transform exited early: %s", strings.TrimSpace(b.stderr.String()))
		}
		return nil, fmt.Errorf("read transform output: %w", err)
	}
	output, err := hex.DecodeString(strings.TrimSpace(line))
	if err != nil {
		return nil, fmt.Errorf("invalid transform output %q: %w", line, err)
	}
	if len(output) != 16 {
		return nil, fmt.Errorf("block-transform output has wrong length")
	}
	return output, nil
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
