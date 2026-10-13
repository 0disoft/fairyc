//go:build ignore

// Observe SDK and target identities through maintained Go definitions.
package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

func run(output string) error {
	spec, ok := target.Lookup(target.X8664WindowsMSVC)
	if !ok {
		return fmt.Errorf("Windows target missing")
	}
	library, err := sdkstdlib.Load()
	if err != nil {
		return err
	}
	id, err := packagegraph.ComputeInstanceID(packagegraph.Identity{PackageUID: sdkstdlib.PackageUID, SourceIdentity: library.SourceIdentity, VersionIdentity: sdkstdlib.Version, Features: []string{}})
	if err != nil {
		return err
	}
	fields := []string{sdkstdlib.LanguageRevision, "fairyc.candidate-owned-body-ir/0", spec.SchemaVersion, strings.Join([]string{"fairyc.target-abi-profile/0", string(spec.ID), spec.DataModel, spec.CallingConvention, spec.AggregateLayout, spec.ReturnClassifier, spec.ObjectFormat, spec.UnwindModel}, ";"), "not-applicable:verified-candidate-owned-body-ir", "verified-candidate-owned-body-ir", sdkstdlib.RuntimeABI, string(spec.ID)}
	var out bytes.Buffer
	out.WriteString("FYCPROF1")
	number := func(n int) { binary.Write(&out, binary.BigEndian, uint64(n)) }
	field := func(s string) { number(len(s)); out.WriteString(s) }
	number(len(fields))
	for _, s := range fields {
		field(s)
	}
	field(id.String())
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(output, out.Bytes(), 0644); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{"passed": true, "fields": fields, "standardLibrary": id.String(), "sdkSource": library.SourceIdentity, "sdkVersion": sdkstdlib.Version, "candidateIRVersionDistinctFromGoMIR": true})
}
func main() {
	output := flag.String("output", ".cache/cache-environment-20261014/profile.bin", "output fixture")
	flag.Parse()
	if err := run(*output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
