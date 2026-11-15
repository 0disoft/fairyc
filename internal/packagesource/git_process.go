package packagesource

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

// Nil options use Background and Cmd's stdout/start/wait operations. Output.Close
// must release a pending Read and permit concurrent closure. Start/Wait overrides
// must retain process ownership; cancel stops the producer, not the caller context.
type gitProcessOptions struct {
	Context    context.Context
	OpenOutput func(*exec.Cmd) (io.ReadCloser, error)
	Start      func(*exec.Cmd) error
	Wait       func(*exec.Cmd) error
}

func gitProcessPolicy(options []gitProcessOptions) (gitProcessOptions, error) {
	if len(options) > 1 {
		return gitProcessOptions{}, fmt.Errorf("at most one Git process options value is supported")
	}
	var policy gitProcessOptions
	if len(options) == 1 {
		policy = options[0]
	}
	if policy.Context == nil {
		policy.Context = context.Background()
	}
	if policy.OpenOutput == nil {
		policy.OpenOutput = func(command *exec.Cmd) (io.ReadCloser, error) {
			output, err := command.StdoutPipe()
			if err != nil {
				return nil, err
			}
			return output, nil
		}
	}
	if policy.Start == nil {
		policy.Start = (*exec.Cmd).Start
	}
	if policy.Wait == nil {
		policy.Wait = (*exec.Cmd).Wait
	}
	return policy, nil
}

// Close the owned read pipe on caller cancellation even if a descendant inherited
// the write end. WaitDelay alone cannot bound a consumer blocked before Wait.
// Every successfully started producer is waited exactly once, including refusals.
func consumeGitProcess(command *exec.Cmd, cancel context.CancelFunc, consume func(io.Reader) error, options ...gitProcessOptions) (err error) {
	policy, err := gitProcessPolicy(options)
	if err != nil {
		return err
	}
	ctx := policy.Context
	defer func() { err = errors.Join(err, ctx.Err(), context.Cause(ctx)) }()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if command == nil || cancel == nil || consume == nil {
		return fmt.Errorf("Git process requires a command, producer cancellation and consumer")
	}
	stdout, err := policy.OpenOutput(command)
	var closeOutput func() error
	if stdout != nil {
		var once sync.Once
		var closeErr error
		closeOutput = func() error {
			once.Do(func() { closeErr = stdout.Close() })
			return closeErr
		}
		defer func() { err = errors.Join(err, closeOutput()) }()
	}
	if err != nil {
		return err
	}
	if stdout == nil {
		return fmt.Errorf("Git output pipe is missing")
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	command.Stderr = &boundedGitBuffer{limit: 4096}
	if err := policy.Start(command); err != nil {
		return fmt.Errorf("local Git object command failed: %w", err)
	}
	watchDone := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(watchDone)
		cancel()
		_ = closeOutput()
	})
	defer func() {
		if !stop() {
			<-watchDone
		}
	}()
	readErr := ctx.Err()
	if readErr == nil {
		readErr = consume(stdout)
	}
	if readErr != nil || ctx.Err() != nil {
		cancel()
	}
	_ = closeOutput()
	if err := policy.Wait(command); err != nil {
		return errors.Join(readErr, fmt.Errorf("local Git object command failed: %w", err))
	}
	return readErr
}
