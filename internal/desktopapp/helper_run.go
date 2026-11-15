package desktopapp

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
)

const (
	helperRunTimeout  = 3 * time.Minute
	helperPipeWait    = 2 * time.Second
	helperOutputBytes = 64 << 10
)

// HelperRunOptions separates caller cancellation and process I/O. Nil fields
// use Background and Cmd.Run. A Run override must return only after it stops
// using the command's streams, and must honor command cancellation itself.
type HelperRunOptions struct {
	Context context.Context
	Run     func(*exec.Cmd) error
}

// RunHelperWithOptions keeps RunHelper's default limits and containment checks.
// WaitDelay bounds inherited-pipe draining even when the helper exits before
// its descendants. Incomplete output never becomes a successful receipt.
func RunHelperWithOptions(python Python, root, script string, arguments []string, options ...HelperRunOptions) ([]byte, error) {
	if len(options) > 1 {
		return nil, errors.New("at most one desktop helper options value is supported")
	}
	var selected HelperRunOptions
	if len(options) == 1 {
		selected = options[0]
	}
	if selected.Context == nil {
		selected.Context = context.Background()
	}
	if selected.Run == nil {
		selected.Run = func(command *exec.Cmd) error { return command.Run() }
	}
	ctx, cancel := context.WithTimeout(selected.Context, helperRunTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, errors.Join(err, context.Cause(ctx))
	}
	helper, err := Helper(root, script)
	if err != nil {
		return nil, fmt.Errorf("desktop helper %s is missing or outside the project", script)
	}
	args := append(append([]string{}, python.prefix...), "-E", "-s", "-B", helper)
	args = append(args, arguments...)
	command := exec.CommandContext(ctx, python.path, args...)
	cdriver.ConfigureCommandCancellation(command)
	command.WaitDelay = helperPipeWait
	command.Dir = filepath.Dir(helper)
	var out, diag capture
	command.Stdout, command.Stderr = &out, &diag
	if err := errors.Join(selected.Run(command), ctx.Err(), context.Cause(ctx)); err != nil {
		return nil, fmt.Errorf("desktop helper %s failed; run python -B %s manually for details: %w", script, script, err)
	}
	if out.exceeded || diag.exceeded {
		return nil, errors.New("desktop helper output exceeded 64 KiB")
	}
	return out.data.Bytes(), nil
}
