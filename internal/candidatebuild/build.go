package candidatebuild

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

type Options struct {
	Candidate, Manifest, Directory, Output string
	Release                                bool
	Timeout                                time.Duration
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
	if candidate == "" || manifest == "" || timeout <= 0 || timeout > time.Minute {
		return nil, fmt.Errorf("candidate, manifest and timeout in (0,60s] are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	executable, err := filepath.Abs(candidate)
	if err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, executable, "--c-project-file", manifest)
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

func runtimeSource(header string) (string, error) {
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
	if err != nil {
		return "", err
	}
	return scalar + "\n" + storage, nil
}

func Build(options Options) (Result, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return Result{}, fmt.Errorf("candidate native bridge requires Windows x64 MSVC")
	}
	if options.Output == "" {
		return Result{}, fmt.Errorf("output executable is required")
	}
	data, err := Emit(options.Candidate, options.Manifest, options.Directory, options.Timeout)
	if err != nil {
		return Result{}, err
	}
	packet, err := Decode(data)
	if err != nil {
		return Result{}, fmt.Errorf("decode candidate output: %w", err)
	}
	support, err := runtimeSource(packet.Header)
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
	artifact, err := cdriver.BuildWindowsMSVCTargetParts(toolchain, profile, packet.Header, packet.Literals, packet.Units, support)
	if err != nil {
		return Result{}, err
	}
	if err := Publish(options.Output, artifact.Bytes); err != nil {
		return Result{}, err
	}
	return result(data, len(packet.Units), len(artifact.Bytes), profile.Name), nil
}
