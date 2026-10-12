package candidatebuild

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/candidatehost"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

type Options struct {
	Candidate, Manifest, Directory, Output string
	Workspace                              string
	OutputRoot                             string
	Locked                                 bool
	Release                                bool
	RunTrapControl                         bool
	Timeout                                time.Duration
	CompileTimeout                         time.Duration
	StackReserveBytes                      uint64
}
type Result struct {
	PacketSHA256      string
	Units, ImageBytes int
	Profile           string
}

type limitedBuffer struct {
	data     bytes.Buffer
	limit    int
	exceeded bool
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	size := len(p)
	room := b.limit - b.data.Len()
	if len(p) > room {
		p = p[:room]
		b.exceeded = true
	}
	_, err := b.data.Write(p)
	return size, err
}

func Emit(candidate, manifest, directory string, timeout time.Duration) ([]byte, error) {
	if candidate == "" || manifest == "" || timeout <= 0 || timeout > 10*time.Minute {
		return nil, fmt.Errorf("candidate, manifest and timeout in (0,10m] are required")
	}
	return emit(candidate, []string{"--c-project-file", manifest}, directory, timeout)
}

// EmitWorkspace delegates manifest, dependency, lock and frontend processing
// entirely to the selected Fairy executable. The bridge owns process transport.
func EmitWorkspace(candidate, workspace, directory string, locked bool, timeout time.Duration) ([]byte, error) {
	if candidate == "" || workspace == "" || timeout <= 0 || timeout > 10*time.Minute {
		return nil, fmt.Errorf("candidate, workspace and timeout in (0,10m] are required")
	}
	arguments := []string{"--c-workspace", workspace}
	if locked {
		arguments = append(arguments, "--locked")
	}
	return emit(candidate, arguments, directory, timeout)
}

func emit(candidate string, arguments []string, directory string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	executable, err := filepath.Abs(candidate)
	if err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, executable, arguments...)
	cdriver.ConfigureCommandCancellation(command)
	command.Dir = directory
	command.WaitDelay = 2 * time.Second
	stdout := limitedBuffer{limit: maxPacket}
	stderr := limitedBuffer{limit: 65536}
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("candidate frontend timeout: %w", ctx.Err())
	}
	if stdout.exceeded || stderr.exceeded {
		return nil, fmt.Errorf("candidate frontend output limit")
	}
	if err != nil {
		return nil, fmt.Errorf("candidate frontend failed: %w: %s", err, strings.TrimSpace(stderr.data.String()))
	}
	if stderr.data.Len() != 0 {
		return nil, fmt.Errorf("successful candidate emitted diagnostics: %s", strings.TrimSpace(stderr.data.String()))
	}
	return stdout.data.Bytes(), nil
}

var stdSymbols = regexp.MustCompile(`\b(fy_std_[A-Za-z0-9_]+)\s*\(`)

func runtimeSource(header string, runTrapControl bool) (string, error) {
	set := map[string]bool{}
	for _, match := range stdSymbols.FindAllStringSubmatch(header, -1) {
		set[match[1]] = true
	}
	vector := strings.Contains(header, "fy_private_vector_")
	symbols := make([]string, 0, len(set))
	for symbol := range set {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	storage, err := cstdlib.WindowsMSVCCandidateSourceForSymbols(symbols, vector)
	if err != nil {
		return "", err
	}
	scalar, err := cruntime.ScalarSource(100)
	if runTrapControl {
		scalar, err = cruntime.WindowsRunSource(128)
	}
	if err != nil {
		return "", err
	}
	host, err := candidatehost.WindowsSource(header)
	if err != nil {
		return "", err
	}
	if host == "" {
		return scalar + "\n" + storage, nil
	}
	return scalar + "\n" + storage + "\n" + host, nil
}

func Build(options Options) (Result, error) {
	if (options.Manifest == "") == (options.Workspace == "") || options.Locked && options.Workspace == "" {
		return Result{}, fmt.Errorf("exactly one manifest or workspace is required; locked requires workspace")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return Result{}, fmt.Errorf("candidate native bridge requires Windows x64 MSVC")
	}
	if options.Output == "" {
		return Result{}, fmt.Errorf("output executable is required")
	}
	var data []byte
	var err error
	if options.Workspace != "" {
		data, err = EmitWorkspace(options.Candidate, options.Workspace, options.Directory, options.Locked, options.Timeout)
	} else {
		data, err = Emit(options.Candidate, options.Manifest, options.Directory, options.Timeout)
	}
	if err != nil {
		return Result{}, err
	}
	return buildPacket(options, data)
}

// BuildCached consumes an immutable previously emitted packet. The digest is
// mandatory, so cache reuse cannot silently select different frontend output.
func BuildCached(options Options, path, expected string) (Result, error) {
	if options.Candidate != "" || options.Manifest != "" || options.Workspace != "" || options.Locked {
		return Result{}, fmt.Errorf("cached packet cannot be combined with candidate, manifest, workspace or locked")
	}
	data, err := cachedPacket(path, expected)
	if err != nil {
		return Result{}, err
	}
	return buildPacket(options, data)
}

// BuildInput consumes a fresh packet produced by the calling Fairy compiler.
// Unlike cache replay it selects no prior artifact: the opened file is copied
// once, bounded and decoded before any destination publication. No frontend runs.
func BuildInput(options Options, path string) (Result, error) {
	if options.Candidate != "" || options.Manifest != "" || options.Workspace != "" || options.Locked {
		return Result{}, fmt.Errorf("packet input cannot be combined with candidate, manifest, workspace or locked")
	}
	data, err := readPacketInput(path)
	if err != nil {
		return Result{}, err
	}
	return buildPacket(options, data)
}

func readPacketInput(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("packet input path is required")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !stat.Mode().IsRegular() || stat.Size() > maxPacket {
		return nil, fmt.Errorf("packet input must be a regular file within 96 MiB")
	}
	data, err := io.ReadAll(io.LimitReader(file, maxPacket+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPacket {
		return nil, fmt.Errorf("packet input exceeds 96 MiB")
	}
	return data, nil
}

func cachedPacket(path, expected string) ([]byte, error) {
	want, err := hex.DecodeString(expected)
	if err != nil || len(want) != sha256.Size || path == "" {
		return nil, fmt.Errorf("packet path and SHA256 digest are required")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxPacket+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxPacket {
		return nil, fmt.Errorf("cached packet exceeds 96 MiB")
	}
	got := sha256.Sum256(data)
	if !bytes.Equal(got[:], want) {
		return nil, fmt.Errorf("cached packet SHA256 mismatch")
	}
	return data, nil
}

func buildPacket(options Options, data []byte) (Result, error) {
	if err := cdriver.ValidateWindowsProjectStackReserve(options.StackReserveBytes); err != nil {
		return Result{}, err
	}
	compileTimeout := options.CompileTimeout
	if compileTimeout == 0 {
		compileTimeout = 2 * time.Minute
	}
	if compileTimeout <= 0 || compileTimeout > 10*time.Minute {
		return Result{}, fmt.Errorf("candidate compile timeout must be in (0,10m]")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" || options.Output == "" {
		return Result{}, fmt.Errorf("candidate native bridge requires Windows x64 MSVC and output executable")
	}
	packet, err := Decode(data)
	if err != nil {
		return Result{}, fmt.Errorf("decode candidate output: %w", err)
	}
	support, err := runtimeSource(packet.Header, options.RunTrapControl)
	if err != nil {
		return Result{}, fmt.Errorf("select candidate runtime: %w", err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return Result{}, err
	}
	profile := cdriver.LowProfile
	if options.Release {
		profile = cdriver.OptimizedProfile
	}
	artifact, err := cdriver.BuildWindowsMSVCTargetPartsWithStackReserve(toolchain, profile, packet.Header, packet.Literals, packet.Units, support, compileTimeout, options.StackReserveBytes)
	if err != nil {
		return Result{}, err
	}
	var publication error
	if options.OutputRoot != "" {
		publication = PublishWithin(options.OutputRoot, options.Output, artifact.Bytes)
	} else {
		publication = Publish(options.Output, artifact.Bytes)
	}
	if publication != nil {
		return Result{}, publication
	}
	return result(data, len(packet.Units), len(artifact.Bytes), profile.Name), nil
}
