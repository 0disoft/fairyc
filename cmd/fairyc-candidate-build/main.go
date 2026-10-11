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
	output := flags.String("output", "", "destination executable")
	directory := flags.String("directory", ".", "source paths working directory")
	release := flags.Bool("release", false, "optimized native build")
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 || *candidate == "" || *manifest == "" || *output == "" {
		fmt.Fprintln(os.Stderr, "candidate, manifest and output are required")
		return 2
	}
	result, err := candidatebuild.Build(candidatebuild.Options{Candidate: *candidate, Manifest: *manifest, Directory: *directory, Output: *output, Release: *release, Timeout: time.Minute})
	if err != nil {
		fmt.Fprintln(os.Stderr, "candidate build failed:", err)
		return 2
	}
	fmt.Printf("built: %s (%s, %d C units, %d bytes)\n", *output, result.Profile, result.Units, result.ImageBytes)
	return 0
}
