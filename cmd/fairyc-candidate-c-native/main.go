// Test-only native consumer of Fairy-emitted C. No parser, MIR lowering or
// reference expectations run here; the declared C driver builds the input bytes.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

type job struct {
	Name    string `json:"name"`
	Source  string `json:"source"`
	Observe bool   `json:"observe"`
}
type request struct {
	Schema string `json:"schema"`
	Output string `json:"output"`
	Jobs   []job  `json:"jobs"`
}
type observation struct {
	Name            string `json:"name"`
	Profile         string `json:"profile"`
	Compiler        string `json:"compiler"`
	CompilerPath    string `json:"compilerPath"`
	CompilerVersion string `json:"compilerVersion"`
	Source          string `json:"source"`
	SourceSHA256    string `json:"sourceSHA256"`
	RuntimeSHA256   string `json:"runtimeSHA256"`
	Image           string `json:"image"`
	ImageSHA256     string `json:"imageSHA256"`
	Exit            int32  `json:"exit"`
	Stdout          string `json:"stdout"`
	Stderr          string `json:"stderr"`
	StdoutBytes     []byte `json:"stdoutBytes"`
	StderrBytes     []byte `json:"stderrBytes"`
	Trap            uint8  `json:"trap"`
	Value           int64  `json:"value"`
	Error           string `json:"error,omitempty"`
	BuildReused     bool   `json:"buildReused,omitempty"`
}

func digest(data []byte) string { value := sha256.Sum256(data); return hex.EncodeToString(value[:]) }
func save(path string, rows []observation) error {
	data, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
func run() error {
	if len(os.Args) != 3 {
		return fmt.Errorf("usage: fairyc-candidate-c-native <request.json> <report.json>")
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		return err
	}
	if len(data) > 2<<20 {
		return fmt.Errorf("request exceeds 2 MiB")
	}
	var input request
	if err = json.Unmarshal(data, &input); err != nil {
		return err
	}
	if input.Schema != "fairyc.candidate-c-native-request/v1" || len(input.Jobs) == 0 || len(input.Jobs) > 128 || !filepath.IsAbs(input.Output) {
		return fmt.Errorf("invalid request schema, job count or absolute output root")
	}
	if err = os.MkdirAll(input.Output, 0o700); err != nil {
		return err
	}
	if runtime.GOARCH != "amd64" || runtime.GOOS != "windows" && runtime.GOOS != "linux" {
		return fmt.Errorf("native comparison requires windows/amd64 or linux/amd64")
	}
	valid := regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`)
	seen := map[string]bool{}
	var windows cdriver.WindowsHostToolchain
	var linux []cdriver.Toolchain
	if runtime.GOOS == "windows" {
		windows, err = cdriver.ResolveWindowsHostMSVC()
	} else {
		for _, name := range []string{"gcc", "clang"} {
			tool, e := cdriver.Resolve(target.X8664LinuxGNU, name)
			if e != nil {
				return e
			}
			linux = append(linux, tool)
		}
	}
	if err != nil {
		return err
	}
	var rows []observation
	// Reuse only an exact successful Windows build from this local report.
	// Every reused image is hashed again and executed again; no outcome is reused.
	previous := map[string]observation{}
	if data, readErr := os.ReadFile(os.Args[2]); readErr == nil && len(data) <= 16<<20 {
		var old []observation
		if json.Unmarshal(data, &old) == nil {
			for _, row := range old {
				if row.Error == "" {
					previous[row.Name+"/"+row.Compiler+"/"+row.Profile] = row
				}
			}
		}
	}
	for _, item := range input.Jobs {
		if !valid.MatchString(item.Name) || seen[item.Name] {
			return fmt.Errorf("invalid or duplicate job name %q", item.Name)
		}
		seen[item.Name] = true
		unit, e := os.ReadFile(item.Source)
		if e != nil {
			return e
		}
		if len(unit) == 0 || len(unit) > 2<<20+4096 {
			return fmt.Errorf("C unit size invalid for %s", item.Name)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			names := []string{"msvc"}
			if runtime.GOOS == "linux" {
				names = []string{"gcc", "clang"}
			}
			for index, name := range names {
				row := observation{Name: item.Name, Profile: profile.Name, Compiler: name, Source: item.Source, SourceSHA256: digest(unit)}
				var image []byte
				var source string
				if runtime.GOOS == "windows" {
					source, e = cruntime.WindowsRunSource(120)
					row.CompilerPath = windows.Path
					row.CompilerVersion = windows.Version
					if e == nil {
						var artifact cdriver.WindowsHostArtifact
						old := previous[item.Name+"/"+name+"/"+profile.Name]
						expectedImage := filepath.Join(input.Output, item.Name+"-"+name+"-"+profile.Name+".exe")
						if old.SourceSHA256 == row.SourceSHA256 && old.RuntimeSHA256 == digest([]byte(source)) && old.CompilerPath == windows.Path && old.CompilerVersion == windows.Version && old.Image == expectedImage {
							if info, statErr := os.Stat(old.Image); statErr == nil && info.Size() > 0 && info.Size() <= 64<<20 {
								if bytes, readErr := os.ReadFile(old.Image); readErr == nil && digest(bytes) == old.ImageSHA256 {
									artifact = cdriver.WindowsHostArtifact{Bytes: bytes, Toolchain: windows, Profile: profile}
									row.BuildReused = true
								}
							}
						}
						if !row.BuildReused {
							artifact, e = cdriver.BuildWindowsHost(windows, profile, string(unit), source)
						}
						if e == nil {
							artifact.TrapControl = true
							image = artifact.Bytes
							var outcome cdriver.WindowsHostOutcome
							outcome, e = cdriver.ExecuteWindowsHost(artifact)
							row.Exit = outcome.ExitCode
							row.Stdout = outcome.Stdout
							row.Stderr = outcome.Stderr
							if len(outcome.TrapControl) == 1 {
								row.Trap = outcome.TrapControl[0]
							} else if len(outcome.TrapControl) != 0 {
								e = fmt.Errorf("invalid trap receipt")
							}
						}
					}
				} else {
					source, e = cruntime.LinuxRunSource(120)
					row.CompilerPath = linux[index].Path
					row.CompilerVersion = linux[index].Version
					if e == nil {
						var artifact cdriver.Artifact
						artifact, e = cdriver.Build(linux[index], profile, string(unit), source)
						if e == nil {
							image = artifact.Bytes
							var outcome cdriver.LinuxHostOutcome
							outcome, e = cdriver.ExecuteLinuxTarget(artifact)
							row.Exit = outcome.ExitCode
							row.Stdout = outcome.Stdout
							row.Stderr = outcome.Stderr
							row.Trap = outcome.TrapCategory
						}
					}
				}
				row.RuntimeSHA256 = digest([]byte(source))
				row.StdoutBytes = []byte(row.Stdout)
				row.StderrBytes = []byte(row.Stderr)
				if len(image) != 0 {
					row.Image = filepath.Join(input.Output, item.Name+"-"+name+"-"+profile.Name)
					if runtime.GOOS == "windows" {
						row.Image += ".exe"
					}
					row.ImageSHA256 = digest(image)
					if writeErr := os.WriteFile(row.Image, image, 0o700); writeErr != nil {
						return writeErr
					}
				}
				if e == nil && row.Trap == 0 {
					if item.Observe {
						const prefix = "fairyc-candidate-result/v1:"
						if row.Exit != 0 || !strings.HasPrefix(row.Stderr, prefix) || !strings.HasSuffix(row.Stderr, "\n") {
							e = fmt.Errorf("missing full-width result observation")
						} else {
							line := strings.TrimSuffix(strings.TrimPrefix(row.Stderr, prefix), "\n")
							row.Value, e = strconv.ParseInt(strings.TrimSuffix(line, "\r"), 10, 32)
						}
					} else {
						row.Value = int64(row.Exit)
						if row.Stderr != "" {
							e = fmt.Errorf("unexpected standalone stderr")
						}
					}
				} else if e == nil && (row.Trap > 7 || row.Exit != int32(120+row.Trap) || row.Stderr != "") {
					e = fmt.Errorf("invalid native trap outcome")
				}
				if e != nil {
					row.Error = e.Error()
				}
				rows = append(rows, row)
				if err = save(os.Args[2], rows); err != nil {
					return err
				}
				if e != nil {
					return fmt.Errorf("%s/%s/%s: %w", item.Name, name, profile.Name, e)
				}
			}
		}
	}
	fmt.Printf("PASS: %d native C executions\n", len(rows))
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
