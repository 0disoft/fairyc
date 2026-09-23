package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// The padded call aligns source length and value spans without adding labels.
// This is a causal size probe, not a runtime or compiler-time benchmark.
func TestArgumentLabelsCacheAttribution(t *testing.T) {
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	plain := "total += expression(1, 0, true);\n"
	labels := "total += expression(.minimum = 1, .depth = 0, .allow_record = true);\n"
	padded := "total += expression(" + strings.Repeat(" ", len(".minimum = ")) + "1, " + strings.Repeat(" ", len(".depth = ")) + "0, " + strings.Repeat(" ", len(".allow_record = ")) + "true);\n"
	if len(padded) != len(labels) {
		t.Fatalf("padded call length %d differs from labels %d", len(padded), len(labels))
	}
	for _, value := range []string{"1,", "0,", "true)"} {
		if strings.Index(padded, value) != strings.Index(labels, value) {
			t.Fatalf("value %q source offsets differ", value)
		}
	}
	structure := reflect.TypeOf(ast.Expression{})
	withoutLabels := make([]reflect.StructField, 0, structure.NumField()-1)
	for i := 0; i < structure.NumField(); i++ {
		field := structure.Field(i)
		if field.Name != "CallLabels" {
			withoutLabels = append(withoutLabels, field)
		}
	}
	if len(withoutLabels) != structure.NumField()-1 {
		t.Fatal("CallLabels field missing from AST expression")
	}
	t.Logf("ARGUMENT_LABEL_AST_LAYOUT current=%d without_label_field=%d", structure.Size(), reflect.StructOf(withoutLabels).Size())
	type sizeRow struct {
		Source, Memory, Disk, DiskArtifact, DiskOther int
	}
	rows := map[string]sizeRow{}
	artifacts := map[string][]byte{}
	for _, variant := range []struct{ name, call string }{{"plain", plain}, {"padded", padded}, {"labels", labels}} {
		root := t.TempDir()
		source := "module main; fn usize expression(usize minimum, usize depth, bool allow_record) { return if (allow_record) { minimum + depth } else { depth }; } fn i32 main() { var usize total = 0;\n" + strings.Repeat(variant.call, 64) + "assert(total == 64); return 0; }"
		writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.label-cost\"\nname = \"label-cost\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
		writeTestSource(t, root, "src/main.fy", source)
		workspace, err := workspaceloader.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		memory, err := workspacecompiler.NewSession(workspacecompiler.Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 4, MaxBytes: 16 << 20}, MaxArtifactBytes: 8 << 20})
		if err != nil {
			t.Fatal(err)
		}
		cold, err := memory.Check(workspace)
		if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 1 {
			t.Fatalf("%s memory store: %+v %v", variant.name, cold.Cache, err)
		}
		warm, err := memory.Check(workspace)
		if err != nil || warm.Cache.Outcome != workspacecompiler.OutcomeAllHit {
			t.Fatalf("%s memory reuse: %+v %v", variant.name, warm.Cache, err)
		}
		memoryBytes := memory.Stats().Store.Bytes
		if err := memory.Close(); err != nil {
			t.Fatal(err)
		}

		cacheRoot := t.TempDir()
		config := workspacecompiler.Configuration{Target: target.X8664WindowsMSVC, MaxArtifactBytes: 8 << 20, Persistent: &workspacecompiler.PersistentConfiguration{Root: cacheRoot, CompilerBuild: identity.String(), Limits: packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}}}
		disk, err := workspacecompiler.NewSession(config)
		if err != nil {
			t.Fatal(err)
		}
		stored, err := disk.Check(workspace)
		if err != nil || !stored.Compiler.Verified || stored.Cache.Stored != 1 {
			t.Fatalf("%s disk store: %+v %v", variant.name, stored.Cache, err)
		}
		if err := disk.Close(); err != nil {
			t.Fatal(err)
		}
		reopened, err := workspacecompiler.NewSession(config)
		if err != nil {
			t.Fatal(err)
		}
		hit, err := reopened.Check(workspace)
		if err != nil || hit.Cache.Outcome != workspacecompiler.OutcomeAllHit {
			t.Fatalf("%s disk reuse: %+v %v", variant.name, hit.Cache, err)
		}
		if err := reopened.Close(); err != nil {
			t.Fatal(err)
		}
		var record []byte
		err = filepath.WalkDir(cacheRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if record != nil {
				return fmt.Errorf("multiple persistent records")
			}
			record, walkErr = os.ReadFile(path)
			return walkErr
		})
		if err != nil {
			t.Fatal(err)
		}
		var envelope struct {
			Record struct {
				Artifacts []struct {
					Data json.RawMessage `json:"data"`
				} `json:"artifacts"`
			} `json:"record"`
		}
		if len(record) == 0 {
			t.Fatal("missing persistent record")
		}
		if err := json.Unmarshal(record, &envelope); err != nil {
			t.Fatal(err)
		}
		if len(envelope.Record.Artifacts) != 1 {
			t.Fatal("expected one disk artifact")
		}
		artifactJSON := envelope.Record.Artifacts[0].Data
		var artifact []byte
		if err := json.Unmarshal(artifactJSON, &artifact); err != nil {
			t.Fatal(err)
		}
		rows[variant.name] = sizeRow{Source: len(source), Memory: int(memoryBytes), Disk: len(record), DiskArtifact: len(artifact), DiskOther: len(record) - len(artifactJSON)}
		artifacts[variant.name] = artifact
	}
	if rows["padded"].Memory != rows["labels"].Memory || rows["padded"].Disk != rows["labels"].Disk || rows["padded"].DiskArtifact != rows["labels"].DiskArtifact || rows["padded"].DiskOther != rows["labels"].DiskOther {
		t.Fatalf("labels added cache bytes beyond equal-length, value-aligned plain calls: padded=%+v labels=%+v", rows["padded"], rows["labels"])
	}
	t.Logf("ARGUMENT_LABEL_STABLE_PAYLOAD padded_equals_labels=%t", bytes.Equal(artifacts["padded"], artifacts["labels"]))
	t.Logf("ARGUMENT_LABEL_CACHE_ATTRIBUTION plain=%+v padded=%+v labels=%+v", rows["plain"], rows["padded"], rows["labels"])
}
