// Compiler-private Linux backend companion; the Fairy CLI owns the frontend.
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
	flags := flag.NewFlagSet("fairyc-candidate-linux-build", flag.ContinueOnError)
	input := flags.String("packet-input", "", "fresh target-marked C packet")
	stage := flags.String("stage-root", "", "caller-owned private staging directory")
	root := flags.String("output-root", "", "opened workspace publication root")
	output := flags.String("output", "", "relative executable destination")
	runControl := flags.Bool("run-trap-control", false, "private run image with fd3 trap control")
	release := flags.Bool("release", false, "optimized build")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *input == "" || *stage == "" || *root == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "candidate build failed: packet-input, stage-root, output-root and output are required")
		return 2
	}
	_, err := candidatebuild.BuildLinuxInput(candidatebuild.Options{Output: *output, OutputRoot: *root, Release: *release, RunTrapControl: *runControl, CompileTimeout: time.Minute}, *input, *stage, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, "candidate build failed:", err)
		return 1
	}
	return 0
}
