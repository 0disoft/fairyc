// fairyc-backend-study is a manual research harness, not a compiler backend selector.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if err := execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func execute() error {
	output := flag.String("output", "", "new JSON report path (required)")
	samples := flag.Int("samples", 1, "fresh builds per workload/profile, 1..5")
	cc := flag.String("cc", "", "msvc on Windows; gcc or clang on Linux")
	flag.Parse()
	if *output == "" || flag.NArg() != 0 {
		return fmt.Errorf("use --output PATH [--samples 1..5] [--cc NAME]")
	}
	if _, err := os.Lstat(*output); err == nil {
		return fmt.Errorf("output already exists: %s", *output)
	} else if !os.IsNotExist(err) {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	revision, err := exec.CommandContext(ctx, "git", "rev-parse", "HEAD").Output()
	if err != nil {
		return fmt.Errorf("read source revision: %w", err)
	}
	status, err := exec.CommandContext(ctx, "git", "status", "--porcelain").Output()
	if err != nil {
		return err
	}
	r, err := run(*samples, *cc)
	if err != nil {
		return err
	}
	r.SourceRevision = strings.TrimSpace(string(revision))
	r.WorktreeDirty = len(status) != 0
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	fmt.Printf("wrote %s: %d workload/profile rows; LLVM/QBE unmeasured\n", *output, len(r.Measurements))
	return nil
}
