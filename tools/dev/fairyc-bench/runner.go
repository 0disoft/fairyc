package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func commandOutput(argv ...string) string {
	b, _ := exec.Command(argv[0], argv[1:]...).Output()
	return strings.TrimSpace(string(b))
}
func gitValue(args ...string) string { return commandOutput(append([]string{"git"}, args...)...) }

func run(ctx context.Context, o Options) error {
	if runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return fmt.Errorf("native benchmark supports Windows and Linux, got %s", runtime.GOOS)
	}
	if err := validateWorkloadSelection(o.Languages, o.Workloads); err != nil {
		return err
	}
	var err error
	o.Output, err = filepath.Abs(o.Output)
	if err != nil {
		return err
	}
	// Atomic exclusive creation prevents accidentally reusing or deleting an
	// existing project. No user-supplied directory is recursively removed.
	if err = os.Mkdir(o.Output, 0o755); err != nil {
		return fmt.Errorf("create new output directory: %w", err)
	}
	r := Receipt{Schema: receiptSchema, Started: time.Now().UTC().Format(time.RFC3339), Commit: gitValue("rev-parse", "HEAD"), Dirty: gitValue("status", "--porcelain") != "", Host: Host{OS: runtime.GOOS, Arch: runtime.GOARCH, CPU: cpuName(), CPUs: runtime.NumCPU(), GoVersion: runtime.Version()}, Method: "Build footprints sum logical file lengths after each command in disjoint output/cache roots. Toolchain installations are inventoried separately; transient peak disk usage is not measured. Cold means empty experiment caches, not OS page caches. Timings include driver startup. C directly rebuilds all sources, including no-op. Development profiles have different debug/safety defaults.", Options: RunSettings{Languages: o.Languages, Profiles: o.Profiles, Workloads: o.Workloads, Edits: o.Edits, Modules: o.Modules, Repetitions: o.Repetitions, Timeout: o.Timeout.String()}}
	failed := false
	installed := map[string]int{}
	for _, language := range o.Languages {
		a := discover(ctx, o, language)
		if a.toolchain.Status == "ready" && o.Workloads[0] == "rust-dependencies" {
			o.dependencyLock, r.Preparation, err = resolveDependencyLock(ctx, o, a)
			if err != nil {
				a.toolchain.Status = "unavailable"
				a.toolchain.Reason = err.Error()
			}
			r.Method += " Rust dependency builds start with prefetched registry files and empty build outputs; network preparation is outside build timing. Retained registry files are included in shared cache."
		}
		r.Toolchains = append(r.Toolchains, a.toolchain)
		if a.toolchain.Status == "ready" {
			addInstallations(&r, a, installed)
		}
		for _, profile := range o.Profiles {
			if profile == "small-dev" && language != "rust" {
				continue
			}
			for _, workload := range o.Workloads {
				input := workloadInput()
				if workload == "file-search" {
					input = searchInput()
				}
				if workload == "rust-dependencies" {
					input = dependencyInput()
				}
				for repeat := 1; repeat <= o.Repetitions; repeat++ {
					c := Case{ID: fmt.Sprintf("%s-%s-%s-%02d", language, profile, workload, repeat), Language: language, Profile: profile, Workload: workload, Repetition: repeat, Status: "passed", InputHash: digest(input)}
					if a.toolchain.Status != "ready" {
						c.Status = "unavailable"
						c.Reason = a.toolchain.Reason
						failed = true
					} else {
						fmt.Fprintln(os.Stderr, "measuring", c.ID)
						if err := runCase(ctx, o, a, &c, input); err != nil {
							c.Status = "failed"
							c.Reason = err.Error()
							failed = true
							fmt.Fprintln(os.Stderr, c.ID, err)
						}
					}
					r.Cases = append(r.Cases, c)
					if err := writeJSON(filepath.Join(o.Output, "receipt.json"), r); err != nil {
						return err
					}
					if ctx.Err() != nil {
						return ctx.Err()
					}
				}
			}
		}
	}
	r.Finished = time.Now().UTC().Format(time.RFC3339)
	if err := writeJSON(filepath.Join(o.Output, "receipt.json"), r); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "receipt:", filepath.Join(o.Output, "receipt.json"))
	if failed {
		return fmt.Errorf("one or more cases failed or were unavailable; inspect receipt.json")
	}
	return nil
}

func runCase(ctx context.Context, o Options, a adapter, c *Case, input []byte) error {
	root := filepath.Join(o.Output, "cases", c.ID)
	project := filepath.Join(root, "project")
	state := filepath.Join(root, "state")
	if err := os.MkdirAll(project, 0o755); err != nil {
		return err
	}
	if err := makeState(state); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "input.txt"), input, 0o644); err != nil {
		return err
	}
	if c.Workload == "rust-dependencies" {
		if err := prepareDependencies(ctx, o, a, c, project, state); err != nil {
			return err
		}
	}
	modules := 1
	if c.Workload == "many-modules" {
		modules = o.Modules
	}
	for step := 0; step < 2+o.Edits; step++ {
		revision := max(0, step-1)
		stage, change := "cold", "none"
		if step == 1 {
			stage = "no-op"
		}
		if step > 1 {
			stage = "edit"
			change = "private"
			if revision%5 == 0 {
				change = "public-interface"
			}
		}
		files := sources(a.language, modules, revision)
		expected := reference(input, revision, modules)
		if c.Workload == "file-search" {
			files = searchSources(a.language, revision)
			expected = searchReference(input, revision)
		}
		if c.Workload == "rust-dependencies" {
			files = dependencySources(revision)
			files["Cargo.lock"] = o.dependencyLock
			expected = dependencyReference(input, revision)
		}
		if err := writeSources(project, files); err != nil {
			return err
		}
		hash, bytes, count := sourceIdentity(files)
		s := Sample{Stage: stage, Change: change, Edit: revision, SourceHash: hash, SourceBytes: bytes, SourceFiles: count, Expected: expected}
		args, env, err := a.build(project, state, c.Profile)
		if err != nil {
			return err
		}
		if c.Workload == "rust-dependencies" {
			args = append(args, "--locked")
		}
		command, out, buildErr := execute(ctx, o.Timeout, project, env, nil, args...)
		command.Log = out + command.Log
		s.WallMS = command.WallMS
		s.Commands = append(s.Commands, portable(command, o.Output))
		if buildErr == nil {
			verifyArgs := []string{a.binary(project, c.Profile)}
			var stdin = input
			if c.Workload == "file-search" || c.Workload == "rust-dependencies" {
				verifyArgs = append(verifyArgs, "../input.txt")
				stdin = nil
			}
			verification, actual, verifyErr := execute(ctx, o.Timeout, project, env, stdin, verifyArgs...)
			s.Commands = append(s.Commands, portable(verification, o.Output))
			s.Actual = strings.ReplaceAll(actual, "\r\n", "\n")
			s.Verified = verifyErr == nil && verification.Log == "" && s.Actual == s.Expected
		}
		s.Footprint, err = inventory(project, state, a.binary(project, c.Profile))
		if err != nil {
			return err
		}
		c.Samples = append(c.Samples, s)
		if buildErr != nil {
			return fmt.Errorf("%s edit %d: build failed (see command log)", stage, revision)
		}
		if !s.Verified {
			return fmt.Errorf("%s edit %d: output mismatch or program failed", stage, revision)
		}
	}
	before := c.Samples[len(c.Samples)-1].Footprint.Total
	clean := Sample{Stage: "clean", Change: "none", Edit: o.Edits, Action: "remove project build outputs; preserve experiment shared caches"}
	if a.language == "rust" {
		_, env, err := a.build(project, state, c.Profile)
		if err != nil {
			return err
		}
		command, out, err := execute(ctx, o.Timeout, project, env, nil, a.executable, "clean", "--offline")
		command.Log = out + command.Log
		clean.Commands = append(clean.Commands, portable(command, o.Output))
		clean.WallMS = command.WallMS
		clean.Action = "cargo clean --offline; preserve isolated CARGO_HOME"
		if err != nil {
			return fmt.Errorf("cargo clean failed: %s", command.Log)
		}
	} else {
		roots := []string{"build"}
		if a.language == "zig" {
			roots = []string{"zig-out", ".zig-cache"}
		}
		boundary, err := os.OpenRoot(project)
		if err != nil {
			return err
		}
		for _, name := range roots {
			if err := boundary.RemoveAll(name); err != nil {
				boundary.Close()
				return err
			}
		}
		if err := boundary.Close(); err != nil {
			return err
		}
	}
	var err error
	clean.Footprint, err = inventory(project, state, a.binary(project, c.Profile))
	if err != nil {
		return err
	}
	clean.ReclaimedBytes = before - clean.Footprint.Total
	c.Samples = append(c.Samples, clean)
	return nil
}

func cpuName() string {
	if name := os.Getenv("PROCESSOR_IDENTIFIER"); name != "" {
		return name
	}
	if b, err := os.ReadFile("/proc/cpuinfo"); err == nil {
		for _, line := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(line, "model name") {
				_, name, _ := strings.Cut(line, ":")
				return strings.TrimSpace(name)
			}
		}
	}
	return "unknown"
}
