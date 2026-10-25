package packageartifact

import (
	"fmt"
	"strings"
	"testing"
)

func TestStableRebindRetainsBodyValidationAtBothEntryPoints(t *testing.T) {
	process := typedProjectionArtifact(t, 1)
	for _, transfers := range []bool{false, true} {
		t.Run(fmt.Sprintf("transfers=%v", transfers), func(t *testing.T) {
			stable, err := describeStableArtifact(process, stablePackageArtifactLimit, transfers)
			if err != nil {
				t.Fatal(err)
			}
			rebind := func(artifact StableArtifact) error {
				if transfers {
					_, err := RebindStorageTransferStableArtifacts([]StableArtifact{artifact}, process.PackageCoordinates, 30000, 40000, 1)
					return err
				}
				_, err := RebindStableArtifactsWithBindings([]StableArtifact{artifact}, process.PackageCoordinates, 30000, 40000, 1)
				return err
			}
			if err := rebind(stable); err != nil {
				t.Fatalf("valid initial reassembly: %v", err)
			}
			for _, change := range []string{"function identity", "local count"} {
				t.Run(change, func(t *testing.T) {
					changed := stable
					changed.Functions = append([]StableBodyArtifact(nil), stable.Functions...)
					want := "retains a build-local function identity"
					if change == "function identity" {
						// Rebinding overwrites this field, so final MIR verification
						// alone would not reject the original invalid artifact.
						changed.Functions[0].Function.Symbol = 7
					} else {
						// An unused extra local could otherwise disappear on rebind.
						changed.Functions[0].LocalCount++
						want = "graph is not canonical"
					}
					if err := rebind(changed); err == nil || !strings.Contains(err.Error(), want) {
						t.Fatalf("batch entry point bypassed body validation: %v", err)
					}
					_, err := RebindStableLeafBody(changed.Functions[0], changed.Context, process.PackageCoordinates, 30000, 1, stablePackageArtifactLimit)
					if err == nil || !strings.Contains(err.Error(), want) {
						t.Fatalf("standalone entry point bypassed body validation: %v", err)
					}
				})
			}
		})
	}
}
