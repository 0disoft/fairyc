package candidatebuild

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/nativesupport"
	"github.com/0disoft/fairyc/internal/target"
)

const LinuxPacketMarker = "/* fairyc.candidate-target/x86_64-linux-gnu/v1 */"

var linuxRuntimeSymbols = regexp.MustCompile(`\b((?:fy_std_|fy_private_vector_|fairyc_private_)[A-Za-z0-9_]+)\s*\(`)

func linuxRuntimeSource(header string) (string, error) {
	return linuxRuntimeSourceForRun(header, false)
}

func linuxRuntimeSourceForRun(header string, run bool) (string, error) {
	if !strings.Contains(header, LinuxPacketMarker) {
		return "", fmt.Errorf("Linux candidate packet target marker is required")
	}
	var symbols []string
	for _, match := range linuxRuntimeSymbols.FindAllStringSubmatch(header, -1) {
		if strings.HasPrefix(match[1], "fy_std_windows_") || strings.HasPrefix(match[1], "fairyc_private_windows_") {
			return "", fmt.Errorf("Windows runtime symbol cannot enter Linux build: %s", match[1])
		}
		symbols = append(symbols, match[1])
	}
	return nativesupport.Source(target.X8664LinuxGNU, symbols, nativesupport.Options{SDKStorage: true, CandidateHost: true, TrapReceipt: run})
}

// BuildLinuxInput consumes only a fresh, bounded, target-marked C packet. Its
// caller owns the private stage directory and process-group cleanup. No Fairy
// parser/frontend, source discovery, cached artifact or MSVC fallback runs here.
func BuildLinuxInput(options Options, input, stage, compiler string) (Result, error) {
	if options.Candidate != "" || options.Manifest != "" || options.Workspace != "" || options.Locked || options.StackReserveBytes != 0 {
		return Result{}, fmt.Errorf("Linux packet build cannot select frontend, cache or Windows stack options")
	}
	timeout := options.CompileTimeout
	if timeout == 0 {
		timeout = time.Minute
	}
	if timeout <= 0 || timeout > 10*time.Minute {
		return Result{}, fmt.Errorf("Linux compile timeout must be in (0,10m]")
	}
	if options.RunTrapControl && (options.Output != "program" || options.OutputRoot != stage) {
		return Result{}, fmt.Errorf("Linux run image must be private stage/program")
	}
	if options.Output == "" || options.OutputRoot == "" || stage == "" {
		return Result{}, fmt.Errorf("Linux packet build requires stage, output and publication root")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Result{}, fmt.Errorf("Linux packet build requires linux/amd64")
	}
	data, err := readPacketInput(input)
	if err != nil {
		return Result{}, err
	}
	packet, err := Decode(data)
	if err != nil {
		return Result{}, err
	}
	support, err := linuxRuntimeSourceForRun(packet.Header, options.RunTrapControl)
	if err != nil {
		return Result{}, fmt.Errorf("select Linux runtime: %w", err)
	}
	profile := cdriver.LowProfile
	if options.Release {
		profile = cdriver.OptimizedProfile
	}
	artifact, err := cdriver.BuildLinuxTargetParts(stage, compiler, profile, packet.Header, packet.Literals, packet.Units, support, linuxProcessArguments(packet.Header), timeout)
	if err != nil {
		return Result{}, err
	}
	if err := PublishWithin(options.OutputRoot, options.Output, artifact.Bytes); err != nil {
		return Result{}, err
	}
	return result(data, len(packet.Units), len(artifact.Bytes), profile.Name), nil
}

// The same closed symbol scan that selects process storage also selects its
// native startup. Generic packet main(void) cannot initialize Linux OS argv.
func linuxProcessArguments(header string) bool {
	for _, match := range linuxRuntimeSymbols.FindAllStringSubmatch(header, -1) {
		if match[1] == "fy_std_process_arg_count_v0" || match[1] == "fy_std_process_arg_v0" {
			return true
		}
	}
	return false
}
