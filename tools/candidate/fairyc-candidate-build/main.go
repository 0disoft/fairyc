// fairyc-candidate-build is a private bootstrap bridge. The selected Fairy
// candidate owns all frontend work; this command consumes its C packet only.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
	"github.com/0disoft/fairyc/internal/cdriver"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	flags := flag.NewFlagSet("fairyc-candidate-build", flag.ContinueOnError)
	candidate := flags.String("candidate", "", "candidate executable")
	manifest := flags.String("manifest", "", "private source manifest")
	workspace := flags.String("workspace", "", "Fairy workspace root; candidate owns manifest/dependency loading")
	locked := flags.Bool("locked", false, "verify workspace lock without writing")
	packet := flags.String("packet", "", "previously emitted private C packet")
	packetInput := flags.String("packet-input", "", "fresh private C packet from the calling Fairy compiler; no frontend or cache replay")
	packetDigest := flags.String("packet-sha256", "", "required cached packet SHA256 digest")
	output := flags.String("output", "", "destination executable")
	outputRoot := flags.String("output-root", "", "opened publication root; output must be relative and remain inside this root")
	directory := flags.String("directory", ".", "source paths working directory")
	release := flags.Bool("release", false, "optimized native build")
	runTrapControl := flags.Bool("run-trap-control", false, "private stable-run trap channel and exit base")
	frontendTimeout := flags.Duration("frontend-timeout", time.Minute, "candidate frontend deadline in (0,10m]")
	compileTimeout := flags.Duration("compile-timeout", 2*time.Minute, "per-compiler/link deadline in (0,10m]")
	stackReserve := flags.Uint64("stack-reserve-bytes", 0, "private compiler image stack reserve: zero/default or 64 KiB aligned, at most 16 MiB")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	sourceMode := *candidate != "" || *manifest != "" || *workspace != ""
	cacheMode := *packet != ""
	inputMode := *packetInput != ""
	selected := 0
	for _, enabled := range []bool{sourceMode, cacheMode, inputMode} {
		if enabled {
			selected++
		}
	}
	invalidSource := sourceMode && (*candidate == "" || (*manifest == "") == (*workspace == "") || *packetDigest != "" || *locked && *workspace == "")
	invalidCache := cacheMode && (*packetDigest == "" || *locked)
	invalidInput := inputMode && (*packetDigest != "" || *locked)
	if flags.NArg() != 0 || *output == "" || selected != 1 || invalidSource || invalidCache || invalidInput {
		fmt.Fprintln(os.Stderr, "output and either candidate with one manifest/workspace, packet/packet-sha256, or fresh packet-input are required; locked requires workspace")
		return 2
	}
	if *compileTimeout <= 0 || *compileTimeout > 10*time.Minute {
		fmt.Fprintln(os.Stderr, "compile-timeout must be in (0,10m]")
		return 2
	}
	if *frontendTimeout <= 0 || *frontendTimeout > 10*time.Minute {
		fmt.Fprintln(os.Stderr, "frontend-timeout must be in (0,10m]")
		return 2
	}
	if err := cdriver.ValidateWindowsProjectStackReserve(*stackReserve); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	options := candidatebuild.Options{Candidate: *candidate, Manifest: *manifest, Workspace: *workspace, Locked: *locked, Directory: *directory, Output: *output, OutputRoot: *outputRoot, Release: *release, RunTrapControl: *runTrapControl, Timeout: *frontendTimeout, CompileTimeout: *compileTimeout, StackReserveBytes: *stackReserve}
	var result candidatebuild.Result
	var err error
	if *packetInput != "" {
		result, err = candidatebuild.BuildInput(options, *packetInput)
	} else if *packet != "" {
		result, err = candidatebuild.BuildCached(options, *packet, *packetDigest)
	} else {
		result, err = candidatebuild.Build(options)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "candidate build failed:", err)
		return 2
	}
	fmt.Printf("built: %s (%s, %d C units, %d bytes)\n", *output, result.Profile, result.Units, result.ImageBytes)
	return 0
}
