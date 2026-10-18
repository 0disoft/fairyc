package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/0disoft/fairyc/internal/prevrevision"
)

var manifests = []string{
	"spec/fixtures/semantic/manifest-v0.json",
	"spec/fixtures/result/manifest-v0.json",
	"spec/fixtures/resource/manifest-v0.json",
	"spec/fixtures/region/manifest-v1.json",
	"spec/fixtures/mir/manifest-v0.json",
	"spec/fixtures/interpreter/manifest-v0.json",
	"spec/fixtures/c-emission/manifest-v0.json",
}

func main() {
	oldRef := flag.String("old-ref", "", "Git revision containing previous-revision fixture sources")
	write := flag.Bool("write", false, "atomically replace every refreshed manifest")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "fairyc-refresh-fixture-spans accepts flags only")
		os.Exit(2)
	}
	if strings.TrimSpace(*oldRef) == "" {
		fmt.Fprintln(os.Stderr, "fairyc-refresh-fixture-spans requires --old-ref")
		os.Exit(2)
	}
	report, err := prevrevision.RefreshFixtureSpans(".", *oldRef, manifests, *write)
	encoded, encodeErr := json.MarshalIndent(report, "", "  ")
	if encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
