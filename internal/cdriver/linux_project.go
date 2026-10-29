package cdriver

import (
	"bytes"
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

// BuildLinuxTargetParts builds bounded, already-emitted C in a caller-owned
// private staging directory. Children inherit the companion's captured process
// group, so cancellation by the Fairy owner also kills compiler descendants.
func BuildLinuxTargetParts(directory, compiler string, profile Profile, header, literals string, units []string, support string, processArguments bool, timeout time.Duration) (Artifact, error) {
	if timeout <= 0 || timeout > 10*time.Minute {
		return Artifact{}, fmt.Errorf("Linux project compile timeout must be in (0,10m]")
	}
	if profile != LowProfile && profile != OptimizedProfile {
		return Artifact{}, fmt.Errorf("unsupported Linux project profile")
	}
	if len(header) == 0 || len(header) > 2<<20 || len(literals) == 0 || len(literals) > 2<<20 || len(units) == 0 || len(units) > 4096 || len(support) == 0 || len(support) > 8<<20 {
		return Artifact{}, fmt.Errorf("invalid bounded Linux project C inputs")
	}
	total := len(header) + len(literals) + len(support)
	for _, unit := range units {
		if len(unit) == 0 || len(unit) > 2<<20 {
			return Artifact{}, fmt.Errorf("invalid Linux project C body")
		}
		total += len(unit)
	}
	if total > 96<<20 {
		return Artifact{}, fmt.Errorf("Linux project C transport exceeds 96 MiB")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Artifact{}, fmt.Errorf("Linux project build requires linux/amd64")
	}
	if !filepath.IsAbs(directory) {
		return Artifact{}, fmt.Errorf("Linux project staging directory must be absolute")
	}
	info, err := os.Lstat(directory)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return Artifact{}, fmt.Errorf("Linux project staging directory must be a private real directory")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return Artifact{}, err
	}
	for _, entry := range entries {
		if entry.Name() != "packet.bin" || !entry.Type().IsRegular() {
			return Artifact{}, fmt.Errorf("Linux project stage contains unexpected input %q", entry.Name())
		}
	}
	toolchain, err := selectLinuxToolchain(compiler, func(name string) (Toolchain, error) { return Resolve(target.X8664LinuxGNU, name) })
	if err != nil {
		return Artifact{}, err
	}
	write := func(name, source string) error {
		file, err := os.OpenFile(filepath.Join(directory, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		_, err = file.WriteString(source)
		closeErr := file.Close()
		if err != nil {
			return err
		}
		return closeErr
	}
	if err := write("fy_project.h", header); err != nil {
		return Artifact{}, err
	}
	sources := []string{literals}
	sources = append(sources, units...)
	sources = append(sources, support)
	if processArguments {
		sources = append(sources, linuxProcessArgumentsEntry)
	}
	objects := make([]string, 0, len(sources))
	for index, source := range sources {
		name := fmt.Sprintf("fy_part_%d.c", index)
		object := fmt.Sprintf("fy_part_%d.o", index)
		if err := write(name, source); err != nil {
			return Artifact{}, err
		}
		// The transport has header-owned static helpers. Preserve the candidate
		// partitions' warning policy rather than applying single-unit -Werror.
		args := []string{"-std=c11", "-fno-common", profile.optimizationFlag, "-pipe", "-c", name, "-o", object}
		if processArguments {
			args = append(args, "-DFY_CANDIDATE_NO_MAIN")
		}
		if err := runLinuxProjectCommand(toolchain.Path, directory, timeout, args...); err != nil {
			return Artifact{}, fmt.Errorf("compile %s: %w", name, err)
		}
		data, err := readArtifact(filepath.Join(directory, object))
		if err != nil {
			return Artifact{}, err
		}
		if err := inspectLinuxProjectELF(data, elf.ET_REL); err != nil {
			return Artifact{}, err
		}
		objects = append(objects, object)
	}
	// Fixed ASCII basenames avoid response-file quoting and ARG_MAX growth.
	if err := write("fy_objects.rsp", strings.Join(objects, "\n")+"\n"); err != nil {
		return Artifact{}, err
	}
	if err := runLinuxProjectCommand(toolchain.Path, directory, timeout, "@fy_objects.rsp", "-o", "fy_program"); err != nil {
		return Artifact{}, fmt.Errorf("link Linux project: %w", err)
	}
	image, err := readArtifact(filepath.Join(directory, "fy_program"))
	if err != nil {
		return Artifact{}, err
	}
	if err := inspectLinuxProjectELF(image, elf.ET_EXEC, elf.ET_DYN); err != nil {
		return Artifact{}, err
	}
	return Artifact{Bytes: image, Toolchain: toolchain, Profile: profile, ConsumerMode: C11ConsumerMode}, nil
}

func runLinuxProjectCommand(compiler, directory string, timeout time.Duration, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, compiler, args...)
	cmd.Dir = directory
	cmd.Env = append(os.Environ(), "TMPDIR="+directory, "LC_ALL=C", "LANG=C")
	cmd.WaitDelay = 2 * time.Second
	output := &boundedBuffer{}
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(output.String()))
	}
	if output.truncated {
		return fmt.Errorf("compiler output exceeded %d bytes", maxProcessOutput)
	}
	return nil
}

func inspectLinuxProjectELF(data []byte, kinds ...elf.Type) error {
	file, err := elf.NewFile(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("inspect Linux ELF: %w", err)
	}
	defer file.Close()
	if file.Class != elf.ELFCLASS64 || file.Data != elf.ELFDATA2LSB || file.Machine != elf.EM_X86_64 {
		return fmt.Errorf("Linux project requires little-endian ELF64 x86-64")
	}
	for _, kind := range kinds {
		if file.Type == kind {
			if kind == elf.ET_EXEC || kind == elf.ET_DYN {
				executable := false
				for _, program := range file.Progs {
					if program.Type == elf.PT_LOAD && program.Flags&elf.PF_X != 0 {
						executable = true
					}
				}
				if file.Entry == 0 || !executable {
					return fmt.Errorf("Linux image has no executable load segment or entry")
				}
			}
			return nil
		}
	}
	return fmt.Errorf("unexpected Linux ELF type %s", file.Type)
}

// A separate native startup keeps the admitted packet bytes unchanged. Its
// target-neutral main is suppressed by the packet's existing opt-out macro.
const linuxProcessArgumentsEntry = `#include <stdint.h>
extern int32_t fy_candidate_entry(void);
extern void fy_std_process_set_args_v0(int, char **);
int main(int argc, char **argv) {
    fy_std_process_set_args_v0(argc, argv);
    return (int)fy_candidate_entry();
}
`
