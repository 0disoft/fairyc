// Package testprocess runs bounded test children without a shell or local cache.
package testprocess

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
	"unicode/utf8"
)

type Observation struct {
	ExitCode int    `json:"exitCode"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// Run captures both pipes concurrently, closes stdin, and kills the owned child
// on timeout. Output is strict UTF-8; comparison never folds case or normalizes.
func Run(executable, cwd string, argv []string, timeout time.Duration) (Observation, error) {
	if timeout <= 0 || timeout > time.Minute {
		return Observation{}, fmt.Errorf("timeout must be in (0, 1 minute]")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, argv...)
	cmd.Dir = cwd
	cmd.WaitDelay = 2 * time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	if ctx.Err() != nil {
		return Observation{}, ctx.Err()
	}
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			return Observation{}, err
		}
		code = exit.ExitCode()
	}
	if !utf8.Valid(stdout.Bytes()) || !utf8.Valid(stderr.Bytes()) {
		return Observation{}, fmt.Errorf("child output is not UTF-8")
	}
	return Observation{code, stdout.String(), stderr.String()}, nil
}
