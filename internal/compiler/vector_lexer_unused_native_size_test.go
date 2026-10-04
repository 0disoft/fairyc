//go:build windows

package compiler_test

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/pecoff"
)

// The same indexed source must not acquire native size when the private
// vector-loop parser path is enabled but its syntax is absent.
func TestPrivateVectorSDKCandidateUnusedNativeFootprint(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" {
		t.Fatal("real scanner variants changed")
	}
	stable := variants[0]
	private := stable
	private.name = "candidate mode, indexed source"
	private.candidate = true
	support, err := os.ReadFile(filepath.Join("..", "..", "tools", "benchmarks", "c-budget", "support.c"))
	if err != nil {
		t.Fatal(err)
	}
	extraSupport := "\n" + string(support) + "\nvoid fc_budget_flush(void) { fflush(stdout); }\n"
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			type native struct {
				generatedC      string
				image           []byte
				specializations int
				sections        []pecoff.Section
			}
			var images [2]native
			for i, variant := range []realScannerVectorVariant{stable, private} {
				program, entry, specializations := scannerBudgetFairyProgram(t, variant)
				if specializations == 0 {
					t.Fatalf("%s specializations = %d", variant.name, specializations)
				}
				artifact, unit := buildRealScannerVectorNative(t, program, entry, profile, extraSupport)
				pe, err := pecoff.InspectExecutable(artifact.Bytes)
				if err != nil {
					t.Fatal(err)
				}
				images[i] = native{generatedC: unit.Source, image: artifact.Bytes, sections: pe.Sections, specializations: specializations}
			}
			if images[0].specializations != images[1].specializations || images[0].generatedC != images[1].generatedC ||
				len(images[0].image) != len(images[1].image) ||
				!reflect.DeepEqual(images[0].sections, images[1].sections) {
				t.Fatalf("unused vector mode changed %s native footprint: C bytes %d/%d, EXE bytes %d/%d, sections %v/%v",
					profile.Name, len(images[0].generatedC), len(images[1].generatedC),
					len(images[0].image), len(images[1].image), images[0].sections, images[1].sections)
			}
			t.Logf("unused vector mode %s: generated C=%d EXE=%d sections=%d equal",
				profile.Name, len(images[0].generatedC), len(images[0].image), len(images[0].sections))
		})
	}
}
