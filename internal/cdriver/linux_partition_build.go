package cdriver

import (
	"context"
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/target"
)

// CPartitions is compiler-owned structured C, never an arbitrary input filename
// list. Only fixed ASCII names are materialized in an exclusive temp directory.
type CPartitions struct {
	Header  string
	Sources []string
}

func BuildLinuxPartitions(toolchain Toolchain, profile Profile, parts CPartitions, support string) (artifact Artifact, err error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Artifact{}, fmt.Errorf("partitioned C build requires linux/amd64")
	}
	if toolchain.Target != target.X8664LinuxGNU || toolchain.Path == "" || (toolchain.Name != "gcc" && toolchain.Name != "clang") || (profile != LowProfile && profile != OptimizedProfile) {
		return Artifact{}, fmt.Errorf("invalid partitioned Linux C toolchain or profile")
	}
	if err := validateCPartitions(parts, support); err != nil {
		return Artifact{}, err
	}
	stats := preparePartitionMemory(partitionSourceBytes(parts, support))
	directory, err := os.MkdirTemp("", "fairyc-bootstrap-parts-")
	if err != nil {
		return Artifact{}, err
	}
	defer os.RemoveAll(directory)
	// Includes runtime compilation, all bodies and literals, and final linking.
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	write := func(name, contents string) error {
		return os.WriteFile(filepath.Join(directory, name), []byte(contents), 0600)
	}
	if err := write("fy_bootstrap.h", parts.Header); err != nil {
		return Artifact{}, err
	}
	sources := append(append([]string(nil), parts.Sources...), support)
	objects := make([]string, len(sources))
	for index, source := range sources {
		if err := ctx.Err(); err != nil {
			return Artifact{}, err
		}
		if err := write(fmt.Sprintf("part_%d.c", index), source); err != nil {
			return Artifact{}, err
		}
		objects[index] = fmt.Sprintf("part_%d.o", index)
	}
	flags := compilerArguments(profile, nil)
	stats.GOMAXPROCS = runtime.GOMAXPROCS(0)
	stats.AvailableMemoryBytes = availableCompileMemory()
	stats.Workers = linuxCompileWorkers(stats.GOMAXPROCS, stats.AvailableMemoryBytes)
	defer func() {
		if err != nil {
			err = &PartitionBuildError{Stats: stats, Err: err}
		}
	}()
	err = compileParallel(ctx, stats.Workers, len(sources), func(ctx context.Context, index int) error {
		args := append(append([]string(nil), flags...), "-c", fmt.Sprintf("part_%d.c", index), "-o", objects[index])
		if err := runPartitionCommand(ctx, toolchain.Path, directory, args...); err != nil {
			return fmt.Errorf("compile part_%d.c: %w", index, err)
		}
		object, err := readArtifact(filepath.Join(directory, objects[index]))
		if err != nil {
			return err
		}
		return validateLinuxRuntimeObject(object)
	})
	if err != nil {
		return Artifact{}, err
	}
	if err := write("objects.rsp", strings.Join(objects, "\n")+"\n"); err != nil {
		return Artifact{}, err
	}
	// These are already strict-compiled objects. C-only flags at link time can
	// trigger Clang's unused-command-line-argument diagnostic.
	args := []string{"@objects.rsp", "-o", "program"}
	if err := runPartitionCommand(ctx, toolchain.Path, directory, args...); err != nil {
		return Artifact{}, fmt.Errorf("link partitions: %w", err)
	}
	image, err := readArtifact(filepath.Join(directory, "program"))
	if err != nil {
		return Artifact{}, err
	}
	if err := inspectLinuxProjectELF(image, elf.ET_EXEC, elf.ET_DYN); err != nil {
		return Artifact{}, err
	}
	return Artifact{Bytes: image, Toolchain: toolchain, Profile: profile, ConsumerMode: C11ConsumerMode, RuntimeCache: RuntimeCacheBypass, PartitionBuild: &stats}, nil
}

func partitionSourceBytes(parts CPartitions, support string) int {
	total := len(parts.Header) + len(support)
	for _, source := range parts.Sources {
		total += len(source)
	}
	return total
}

func validateCPartitions(parts CPartitions, support string) error {
	if len(parts.Header) == 0 || len(parts.Header) > 8<<20 || len(parts.Sources) < 2 || len(parts.Sources) > 4096 || len(support) == 0 || len(support) > 8<<20 {
		return fmt.Errorf("invalid bounded bootstrap C partitions")
	}
	total := len(parts.Header) + len(support)
	for _, source := range parts.Sources {
		if len(source) == 0 || len(source) > 8<<20 {
			return fmt.Errorf("invalid bounded bootstrap C partition")
		}
		total += len(source)
	}
	if total > 96<<20 {
		return fmt.Errorf("bootstrap C partitions exceed 96 MiB")
	}
	return nil
}

func runPartitionCommand(ctx context.Context, compiler, directory string, args ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, compiler, args...)
	configureToolCancellation(cmd)
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C", "TMPDIR="+directory)
	cmd.WaitDelay = 2 * time.Second
	output := &boundedBuffer{}
	cmd.Stdout, cmd.Stderr = output, output
	err := cmd.Run()
	if err != nil && cmd.Process != nil && cmd.Cancel != nil {
		_ = cmd.Cancel()
	}
	if ctx.Err() != nil {
		return fmt.Errorf("C toolchain shared %s deadline/cancellation: %w", commandTimeout, ctx.Err())
	}
	if output.truncated {
		return fmt.Errorf("C toolchain command exceeded %d output bytes", maxProcessOutput)
	}
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(output.String()))
	}
	return nil
}
