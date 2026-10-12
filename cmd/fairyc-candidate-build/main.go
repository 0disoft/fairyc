// fairyc-candidate-build is a private bootstrap bridge. The selected Fairy
// candidate owns all frontend work; this command consumes its C packet only.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/0disoft/fairyc/internal/candidatebuild"
)

func main() { os.Exit(run(os.Args[1:])) }
func run(args []string) int {
	flags := flag.NewFlagSet("fairyc-candidate-build", flag.ContinueOnError)
	candidate := flags.String("candidate", "", "candidate executable")
	manifest := flags.String("manifest", "", "private source manifest")
	workspace := flags.String("workspace", "", "Fairy workspace root; candidate owns manifest/dependency loading")
	locked := flags.Bool("locked", false, "verify workspace lock without writing")
	packet := flags.String("packet", "", "previously emitted private C packet")
	packetDigest := flags.String("packet-sha256", "", "required cached packet SHA256 digest")
	output := flags.String("output", "", "destination executable")
	directory := flags.String("directory", ".", "source paths working directory")
	release := flags.Bool("release", false, "optimized native build")
	frontendTimeout := flags.Duration("frontend-timeout", time.Minute, "candidate frontend deadline in (0,10m]")
	compileTimeout := flags.Duration("compile-timeout", 2*time.Minute, "per-compiler/link deadline in (0,10m]")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *output == "" || (*packet == "" && (*candidate == "" || (*manifest == "") == (*workspace == "") || *packetDigest != "" || *locked && *workspace == "")) || (*packet != "" && (*packetDigest == "" || *candidate != "" || *manifest != "" || *workspace != "" || *locked)) {
		fmt.Fprintln(os.Stderr, "output and either candidate with one manifest/workspace, or packet/packet-sha256 are required; locked requires workspace")
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
	options := candidatebuild.Options{Candidate: *candidate, Manifest: *manifest, Workspace: *workspace, Locked: *locked, Directory: *directory, Output: *output, Release: *release, Timeout: *frontendTimeout, CompileTimeout: *compileTimeout}
	var result candidatebuild.Result
	var err error
	if *packet != "" {
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
