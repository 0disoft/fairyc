package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

type candidate struct {
	Name, Status, Reason string
	ToolsOnPATH          []string
}
type observation struct {
	BuildMS, LaunchAndRunMS float64
	ImageBytes              int
	ObjectBytes             *int
	ImageSHA256             string
	ExitCode                int32
	TrapCategory            string
}
type measurement struct {
	Backend, Workload, Profile, Target, SourceSHA256, CSHA256, RuntimeSHA256 string
	SourceBytes, CBytes, MIRFunctions, MIRBlocks, MIROperations              int
	FrontendAndMIRMS, EmissionMS                                             float64
	Interpreter                                                              string
	InterpreterDrops                                                         int
	NativeObserver                                                           string
	Samples                                                                  []observation
}
type report struct {
	Schema, SourceRevision, GOOS, GOARCH, GoVersion, CompilerPath, CompilerVersion, CompilerSHA256 string
	WorktreeDirty                                                                                  bool
	Candidates                                                                                     []candidate
	Measurements                                                                                   []measurement
	Limitations                                                                                    []string
}

func digest(data []byte) string { v := sha256.Sum256(data); return hex.EncodeToString(v[:]) }
func fileDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func elapsed(t time.Time) float64 { return float64(time.Since(t)) / float64(time.Millisecond) }
func candidates() []candidate {
	values := []candidate{{Name: "c11", Status: "measured", Reason: "existing constrained C emitter and fresh native builds"}, {Name: "llvm-ir", Status: "not-measured", Reason: "no Fairy MIR-to-LLVM adapter; tool availability alone is not backend evidence"}, {Name: "qbe-il", Status: "not-measured", Reason: "no Fairy MIR-to-QBE adapter or native Windows parity evidence"}}
	for i, names := range [][]string{{}, {"clang", "llc", "llvm-as"}, {"qbe"}} {
		for _, name := range names {
			if path, err := exec.LookPath(name); err == nil {
				values[i].ToolsOnPATH = append(values[i].ToolsOnPATH, path)
			}
		}
	}
	return values
}
func compileWorkload(w workload) (compiler.Result, float64, error) {
	start := time.Now()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(w.Source)), source.DefaultLimits())
	if err != nil {
		return compiler.Result{}, 0, err
	}
	if parsed.AST == nil {
		return compiler.Result{}, 0, fmt.Errorf("%s syntax: %+v", w.Name, parsed.Tree.Diagnostics)
	}
	checked, err := compiler.CheckExplicitFiles([]semantic.File{{Package: hir.DefaultPackage, Path: "main.fy", Module: parsed.AST}})
	if err != nil || !checked.Verified {
		return compiler.Result{}, 0, fmt.Errorf("%s MIR: %v %+v %+v", w.Name, err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	return checked, elapsed(start), nil
}
func oracle(w workload, p mir.Program) (string, int, error) {
	if w.Foreign {
		return "not-run: foreign calls; native C observer required", 0, nil
	}
	out, err := interpreter.Execute(p, "main::main")
	if err != nil {
		return "", 0, err
	}
	if w.Trap != "" {
		if out.Kind != interpreter.OutcomeTrap || out.TrapCategory != w.Trap {
			return "", 0, fmt.Errorf("wrong interpreter trap: %+v", out)
		}
	} else if out.Kind != interpreter.OutcomeReturn || out.Value != 42 || len(out.Stdout) != 0 || len(out.Stderr) != 0 {
		return "", 0, fmt.Errorf("wrong interpreter result: %+v", out)
	}
	return "passed", len(out.DropTrace), nil
}
func checkNative(w workload, out cdriver.NativeOutcome) error {
	if out.Stdout != "" || out.Stderr != "" {
		return fmt.Errorf("unexpected program streams")
	}
	if w.Trap != "" {
		if out.TrapCategory != w.Trap {
			return fmt.Errorf("expected receipt for %s; got %+v", w.Trap, out)
		}
	} else if out.TrapCategory != "" || out.ExitCode != 42 {
		return fmt.Errorf("expected normal return 42; got %+v", out)
	}
	return nil
}
func run(samples int, cc string) (report, error) {
	if samples < 1 || samples > 5 {
		return report{}, fmt.Errorf("samples must be 1..5")
	}
	r := report{Schema: "fairyc.backend-study/v1", GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, GoVersion: runtime.Version(), Candidates: candidates(), Limitations: []string{"C baseline only; LLVM/QBE have no MIR adapter or native measurements", "fresh compiler processes; no compiler object cache or LTO; OS file caches uncontrolled", "compiler executable hash is recorded; headers, libraries, SDK and compiler DLLs are not exhaustively hashed", "execution timing includes temp image I/O, process launch and output capture; not program throughput", "small fixtures, not a full compiler/parser or clean-machine distribution workload", "target maturity, debug/unwind conformance and ABI completeness are not promoted"}}
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		return r, fmt.Errorf("native study requires existing Windows/Linux amd64 target")
	}
	win := runtime.GOOS == "windows"
	profile := cbackend.Stage0Profile()
	id := target.X8664LinuxGNU
	var windows cdriver.WindowsHostToolchain
	var linux cdriver.Toolchain
	var err error
	var scalar string
	if win {
		if cc != "" && cc != "msvc" {
			return r, fmt.Errorf("Windows study requires msvc")
		}
		windows, err = cdriver.ResolveWindowsHostMSVC()
		r.CompilerPath = windows.Path
		r.CompilerVersion = windows.Version
		profile = cbackend.Stage0WindowsMSVCProfile()
		id = target.X8664WindowsMSVC
		if err == nil {
			scalar, err = cruntime.WindowsRunSource(cruntime.NativeTrapExitBase)
		}
	} else {
		if cc == "" {
			cc = "gcc"
		}
		linux, err = cdriver.Resolve(id, cc)
		r.CompilerPath = linux.Path
		r.CompilerVersion = linux.Version
		if err == nil {
			scalar, err = cruntime.LinuxRunSource(cruntime.NativeTrapExitBase)
		}
	}
	if err != nil {
		return r, err
	}
	r.CompilerSHA256, err = fileDigest(r.CompilerPath)
	if err != nil {
		return r, err
	}
	for _, w := range workloads() {
		checked, frontMS, err := compileWorkload(w)
		if err != nil {
			return r, err
		}
		oracleStatus, drops, err := oracle(w, checked.MIR)
		if err != nil {
			return r, fmt.Errorf("%s oracle: %w", w.Name, err)
		}
		start := time.Now()
		unit, err := cbackend.Emit(checked.MIR, "main::main", profile)
		emitMS := elapsed(start)
		if err != nil {
			return r, err
		}
		support := scalar + "\n" + w.Support
		for _, p := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			m := measurement{Backend: "c11", Workload: w.Name, Profile: p.Name, Target: string(id), SourceSHA256: digest([]byte(w.Source)), CSHA256: digest([]byte(unit.Source)), RuntimeSHA256: digest([]byte(support)), SourceBytes: len(w.Source), CBytes: len(unit.Source), MIRFunctions: len(checked.MIR.Functions), FrontendAndMIRMS: frontMS, EmissionMS: emitMS, Interpreter: oracleStatus, InterpreterDrops: drops}
			if w.Foreign {
				m.NativeObserver = "independent C function result/counter checked by the program"
			}
			for _, fn := range checked.MIR.Functions {
				m.MIRBlocks += len(fn.Blocks)
				for _, b := range fn.Blocks {
					m.MIROperations += len(b.Operations)
				}
			}
			for i := 0; i < samples; i++ {
				var obs observation
				var outcome cdriver.NativeOutcome
				start := time.Now()
				if win {
					a, e := cdriver.BuildWindowsHost(windows, p, unit.Source, support)
					obs.BuildMS = elapsed(start)
					if e != nil {
						return r, fmt.Errorf("%s/%s sample %d: %w", w.Name, p.Name, i+1, e)
					}
					a.TrapControl = true
					if _, e := pecoff.InspectExecutable(a.Bytes); e != nil {
						return r, fmt.Errorf("%s/%s sample %d: %w", w.Name, p.Name, i+1, e)
					}
					if _, e := pecoff.InspectAMD64Unwind(a.Bytes); e != nil {
						return r, fmt.Errorf("%s/%s sample %d: %w", w.Name, p.Name, i+1, e)
					}
					obs.ImageBytes = len(a.Bytes)
					obs.ImageSHA256 = digest(a.Bytes)
					objectBytes := 0
					for _, o := range a.Objects {
						objectBytes += len(o.Bytes)
					}
					obs.ObjectBytes = &objectBytes
					start = time.Now()
					out, e := cdriver.ExecuteWindowsHost(a)
					obs.LaunchAndRunMS = elapsed(start)
					if e != nil {
						return r, fmt.Errorf("%s/%s sample %d: %w", w.Name, p.Name, i+1, e)
					}
					outcome, e = cdriver.ClassifyWindowsNativeOutcome(out)
					if e != nil {
						return r, fmt.Errorf("%s/%s sample %d: %w", w.Name, p.Name, i+1, e)
					}
				} else {
					a, e := cdriver.Build(linux, p, unit.Source, support)
					obs.BuildMS = elapsed(start)
					if e != nil {
						return r, fmt.Errorf("%s/%s sample %d: %w", w.Name, p.Name, i+1, e)
					}
					obs.ImageBytes = len(a.Bytes)
					obs.ImageSHA256 = digest(a.Bytes)
					start = time.Now()
					out, e := cdriver.ExecuteLinuxTarget(a)
					obs.LaunchAndRunMS = elapsed(start)
					if e != nil {
						return r, fmt.Errorf("%s/%s sample %d: %w", w.Name, p.Name, i+1, e)
					}
					category := ""
					if out.TrapCategory != 0 {
						var ok bool
						category, ok = cdriver.NativeTrapCategory(out.TrapCategory)
						if !ok {
							return r, fmt.Errorf("unrecognized native trap category")
						}
					}
					outcome = cdriver.NativeOutcome{ExitCode: out.ExitCode, Stdout: out.Stdout, Stderr: out.Stderr, TrapCategory: category}
				}
				if err := checkNative(w, outcome); err != nil {
					return r, fmt.Errorf("%s/%s: %w", w.Name, p.Name, err)
				}
				obs.ExitCode = outcome.ExitCode
				obs.TrapCategory = outcome.TrapCategory
				m.Samples = append(m.Samples, obs)
			}
			r.Measurements = append(r.Measurements, m)
		}
	}
	return r, nil
}
