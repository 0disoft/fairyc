package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/candidatebuild"
)

func TestCandidateInlineTrapCLI(t *testing.T) {
	t.Setenv("FAIRYC_CLOSURE_CLI_PROFILE", os.Getenv("FAIRYC_INLINE_CLI_PROFILE"))
	profiles := copiedClosureCLIProfiles(t)
	oracle := inlineTrapReference(t)
	for _, profile := range profiles {
		t.Run(profile.Name, func(t *testing.T) {
			image := copiedClosureCLIHost(t, profile)
			installGenericCLIPacketCapture(t, image)
			calls := 0
			for family := 0; family < 2; family++ {
				var last string
				for i, item := range inlineTrapCases {
					root := inlineTrapWorkspace(t, i, family == 1)
					packet := filepath.Join(root, "observed.bin")
					t.Setenv("FAIRYC_GENERIC_CLI_PACKET", packet)
					command := copiedClosureCommand{t: t, image: image, root: root}
					for warm := 0; warm < 2; warm++ {
						args := []string{"run", "--cache-root", "cache", "--cache-status"}
						status := "cache: cold stored\n"
						var before map[string][32]byte
						if warm == 1 {
							args = append(args, "--locked", "--release")
							status = "cache: hit\n"
							before = copiedClosureCacheSnapshot(t, root)
						}
						out, diagnostic := command.run(ExitProgramTrap, args...)
						if out != "" || diagnostic != status+oracle.diagnostics[family][i] {
							t.Fatalf("family=%d case=%s warm=%d: stdout=%q stderr=%q want=%q", family, item.name, warm, out, diagnostic, status+oracle.diagnostics[family][i])
						}
						if warm == 1 && !reflect.DeepEqual(before, copiedClosureCacheSnapshot(t, root)) {
							t.Fatal("trapping warm run rewrote cache generations")
						}
					}
					// Inspect the actual release packet sent unchanged to the companion.
					checkInlineTrapPacket(t, packet)
					calls += command.calls
					last = root
				}
				command := copiedClosureCommand{t: t, image: image, root: last}
				for i, flags := range inlineTrapTestFlags {
					out, diagnostic := command.run(ExitProgramTrap, append([]string{"test", "--json", "--locked"}, flags...)...)
					checkInlineTrapReport(t, out, i != 0, i == 2)
					if diagnostic != "" || out != oracle.reports[family][i] {
						t.Fatalf("family=%d engine=%d: stdout=%s stderr=%q want=%s", family, i, out, diagnostic, oracle.reports[family][i])
					}
				}
				calls += command.calls
			}
			t.Logf("%d independent CLI processes; six cold/warm trap pairs and six exact Go-matching test reports", calls)
		})
	}
}

func checkInlineTrapPacket(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// The shared decoder rejects malformed and duplicate inline/header placement.
	packet, err := candidatebuild.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(packet.Header, "static inline int32_t fy_fn_") != 2 {
		t.Fatal("expected one reachable inline prototype/definition in shared header")
	}
	for _, unit := range packet.Units {
		if strings.Contains(unit, "static inline ") {
			t.Fatal("inline trap body escaped shared header")
		}
	}
}
