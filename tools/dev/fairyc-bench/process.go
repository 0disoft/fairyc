package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if left := (1 << 20) - b.Len(); left > 0 {
		_, _ = b.Buffer.Write(p[:min(left, len(p))])
	}
	return n, nil
}

func environment(overrides []string) []string {
	values := map[string]string{}
	for _, entry := range os.Environ() {
		name, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(name)
		if strings.HasPrefix(upper, "CARGO_") || strings.HasPrefix(upper, "RUSTFLAGS") || strings.HasPrefix(upper, "RUSTC_WRAPPER") || upper == "RUSTC_WORKSPACE_WRAPPER" || upper == "CARGO_ENCODED_RUSTFLAGS" || upper == "CL" || upper == "_CL_" || upper == "LINK" || upper == "CC" || upper == "CFLAGS" || upper == "LDFLAGS" {
			continue
		}
		if runtime.GOOS == "windows" {
			name = upper
		}
		values[name] = value
	}
	for _, entry := range overrides {
		name, value, _ := strings.Cut(entry, "=")
		if runtime.GOOS == "windows" {
			name = strings.ToUpper(name)
		}
		values[name] = value
	}
	result := make([]string, 0, len(values))
	for name, value := range values {
		result = append(result, name+"="+value)
	}
	return result
}

func execute(ctx context.Context, timeout time.Duration, dir string, overrides []string, input []byte, argv ...string) (Command, string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = environment(overrides)
	cmd.Stdin = bytes.NewReader(input)
	cmd.WaitDelay = 2 * time.Second
	configureProcess(cmd)
	var stdout, stderr limitedBuffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	start := time.Now()
	err := cmd.Run()
	result := Command{Argv: argv, Env: overrides, WallMS: float64(time.Since(start).Microseconds()) / 1000, Log: stderr.String(), Exit: 0, TimedOut: ctx.Err() != nil}
	if err != nil {
		result.Exit = -1
		if cmd.ProcessState != nil {
			result.Exit = cmd.ProcessState.ExitCode()
		}
		result.Log += "\n" + err.Error()
	}
	return result, stdout.String(), err
}

// Persist only explicitly selected environment overrides. Local absolute roots
// in commands and compiler diagnostics become portable labels before publishing.
func portable(command Command, output string) Command {
	replace := func(s string) string {
		s = strings.ReplaceAll(s, output, "$RUN")
		if home, err := os.UserHomeDir(); err == nil {
			s = strings.ReplaceAll(s, home, "$HOME")
		}
		return s
	}
	command.Argv = append([]string(nil), command.Argv...)
	command.Env = append([]string(nil), command.Env...)
	for i := range command.Argv {
		command.Argv[i] = replace(command.Argv[i])
	}
	for i := range command.Env {
		command.Env[i] = replace(command.Env[i])
	}
	command.Log = replace(command.Log)
	return command
}

func isolatedEnvironment(state string) []string {
	shared := filepath.Join(state, "shared")
	tmp := filepath.Join(state, "tmp")
	return []string{"LOCALAPPDATA=" + shared, "XDG_CACHE_HOME=" + shared, "TMPDIR=" + tmp, "TMP=" + tmp, "TEMP=" + tmp, "CARGO_HOME=" + filepath.Join(state, "cargo"), "CARGO_NET_OFFLINE=true", "CARGO_TERM_COLOR=never", "NO_COLOR=1"}
}

func makeState(state string) error {
	for _, name := range []string{"shared", "tmp", "cargo"} {
		if err := os.MkdirAll(filepath.Join(state, name), 0o755); err != nil {
			return fmt.Errorf("create isolated state: %w", err)
		}
	}
	return nil
}
