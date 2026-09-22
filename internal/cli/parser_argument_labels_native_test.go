package cli

import (
	"bytes"
	"debug/pe"
	"encoding/json"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

// This compares one source edit, not Fairy's cumulative overhead against C.
// Sizes are observations, not timing claims or binary-byte identity claims.
func TestArgumentLabelsParserNativeSizes(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root, plain, labeled := parserLabelWorkspace(t)
	type row struct {
		Profile, Variant, MSVC string
		EXE, PDB               int
		Objects                map[string]int
		Sections               map[string]uint32
	}
	var rows []row
	var toolchain string
	inputs := []struct {
		text     string
		problems string
	}{
		{"module main;fn void f(){call(.value=R{.x},.mode=true);}", "0"},
		{"module main;fn void f(){for(var i=0;i<2;i+=1){if(i==1){break;}}}", "0"},
		{"module main;fn void f(){return 1}", "1"},
		{labeled, "0"},
	}
	baseline := make([]string, len(inputs))
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		for variant, input := range []string{plain, labeled} {
			writeTestSource(t, root, "src/bodies.fy", input)
			var diagnostics strings.Builder
			compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("%s variant %d: %s", profile.Name, variant, diagnostics.String())
			}
			artifact := compiled.artifact
			if toolchain == "" {
				toolchain = artifact.Toolchain.Version
			}
			if toolchain == "" || artifact.Toolchain.Version != toolchain {
				t.Fatal("missing or changed MSVC version")
			}
			r := row{Profile: profile.Name, Variant: []string{"plain", "labeled"}[variant], MSVC: toolchain,
				EXE: len(artifact.Bytes), PDB: len(artifact.ProgramDatabase), Objects: map[string]int{}, Sections: map[string]uint32{}}
			for _, object := range artifact.Objects {
				r.Objects[object.Name] = len(object.Bytes)
			}
			image, err := pe.NewFile(bytes.NewReader(artifact.Bytes))
			if err != nil {
				t.Fatal(err)
			}
			for _, section := range image.Sections {
				r.Sections[section.Name] = section.VirtualSize
			}
			image.Close()
			if r.EXE == 0 || len(r.Objects) == 0 || r.Sections[".text"] == 0 {
				t.Fatal("incomplete artifact measurements")
			}
			rows = append(rows, r)
			for i, sample := range inputs {
				out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(artifact, strings.NewReader(sample.text))
				if err != nil || out.ExitCode != 0 || out.Stderr != "" || out.TrapControl != "" {
					t.Fatalf("%s variant %d input %d: %+v %v", profile.Name, variant, i, out, err)
				}
				fields := strings.Fields(out.Stdout)
				if len(fields) < 6 || fields[0] != "0" || fields[2] != sample.problems {
					t.Fatalf("input %d unexpected parse status: %q", i, out.Stdout)
				}
				if baseline[i] == "" {
					baseline[i] = out.Stdout
				} else if baseline[i] != out.Stdout {
					t.Fatalf("%s variant %d input %d changed declaration/diagnostic output", profile.Name, variant, i)
				}
			}
		}
	}
	report, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("PARSER_LABEL_NATIVE_SIZES %s", report)
	t.Logf("matching declaration/diagnostic output for %d inputs across 4 builds", len(inputs))
}
