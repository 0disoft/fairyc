package cli

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestArgumentLabelDiskCost(t *testing.T) {
	if os.Getenv("FAIRYC_LABEL_DISK_COST") != "1" {
		t.Skip("opt-in disk measurement")
	}
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	rows := []map[string]any{}
	for repetition := 0; repetition < 3; repetition++ {
		for _, variant := range []string{"plain", "labels"} {
			root := t.TempDir()
			call := "total += expression(1, 0, true);\n"
			if variant == "labels" {
				call = "total += expression(.minimum = 1, .depth = 0, .allow_record = true);\n"
			}
			text := "module main; fn usize expression(usize minimum, usize depth, bool allow_record) { return if (allow_record) { minimum + depth } else { depth }; } fn i32 main() { var usize total = 0;\n" + strings.Repeat(call, 64) + "assert(total == 64); return 0; }"
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.label-cost\"\nname = \"label-cost\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", text)
			w, err := workspaceloader.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			cacheRoot := t.TempDir()
			config := workspacecompiler.Configuration{Target: target.X8664WindowsMSVC, MaxArtifactBytes: 8 << 20, Persistent: &workspacecompiler.PersistentConfiguration{Root: cacheRoot, CompilerBuild: identity.String(), Limits: packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}}}
			cold, err := workspacecompiler.NewSession(config)
			if err != nil {
				t.Fatal(err)
			}
			stored, err := cold.Check(w)
			if err != nil || !stored.Compiler.Verified || stored.Cache.Stored != 1 {
				t.Fatalf("store: %+v %v", stored.Cache, err)
			}
			if err = cold.Close(); err != nil {
				t.Fatal(err)
			}
			warm, err := workspacecompiler.NewSession(config)
			if err != nil {
				t.Fatal(err)
			}
			hit, err := warm.Check(w)
			if err != nil || !hit.Compiler.Verified || hit.Cache.Outcome != workspacecompiler.OutcomeAllHit {
				t.Fatalf("reuse: %+v %v", hit.Cache, err)
			}
			stats, err := warm.PersistentStats()
			if err != nil {
				t.Fatal(err)
			}
			if err = warm.Close(); err != nil {
				t.Fatal(err)
			}
			files := map[string]int64{}
			var total int64
			err = filepath.WalkDir(cacheRoot, func(p string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if d.IsDir() {
					return nil
				}
				info, err := d.Info()
				if err != nil {
					return err
				}
				relative, err := filepath.Rel(cacheRoot, p)
				if err != nil {
					return err
				}
				files[filepath.ToSlash(relative)] = info.Size()
				total += info.Size()
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			rows = append(rows, map[string]any{"variant": variant, "repetition": repetition, "source_sha256": tokenIndexHash([]byte(text)), "file_bytes": total, "files": files, "cache_stats": stats, "warm_hit": hit.Cache.Outcome})
		}
	}
	encoded, err := json.Marshal(map[string]any{"schema": "fairyc.argument-label-disk-cost/v1", "compiler": "d5dc062b44e413f8e308bd402052513e034e7875", "test_executable": identity.String(), "method": "Current compiler only; same 64-call corpus as frontend measurement. Three fresh directories per variant; store one package, close session, reopen and verify all-hit. Sum actual regular file lengths after closing; no disk allocation-block/RSS/fymap claim.", "rows": rows})
	if err != nil {
		t.Fatal(err)
	}
	t.Log("LABEL_DISK_REPORT " + string(encoded))
}
