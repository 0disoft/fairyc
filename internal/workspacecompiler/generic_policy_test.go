package workspacecompiler

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
)

func TestCapabilitySessionAdmission(t *testing.T) {
	configuration := Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
	for name, capabilities := range map[string]admission.Capabilities{
		"query-only":               {Storage: admission.StorageQueries},
		"nominal-only":             {Arguments: admission.NominalArguments},
		"transfer-without-nominal": {Storage: admission.StorageTransfers},
		"candidate":                {Storage: admission.StorageTransfers, Arguments: admission.NominalArguments, Templates: admission.SDKOwnerTemplates, Vectors: admission.CandidateVectors},
		"unknown":                  {Storage: 255},
	} {
		t.Run(name, func(t *testing.T) {
			if session, err := NewGenericSessionWithCapabilities(configuration, genericplan.DefaultPlanLimits(), capabilities); err == nil {
				session.Close()
				t.Fatal("unsupported cache admission accepted")
			}
		})
	}
}

func TestSDKCapabilitySessionReauthenticatesBeforeLowering(t *testing.T) {
	root := writeCacheWorkspace(t, genericAnswer)
	writeFile(t, filepath.Join(root, "src", "main.fy"), genericMain)
	workspace := loadCacheWorkspace(t, root)
	for _, disk := range []bool{false, true} {
		name := "memory"
		if disk {
			name = "disk"
		}
		t.Run(name, func(t *testing.T) {
			configuration := Configuration{Target: target.X8664WindowsMSVC, Cache: packagecache.Limits{MaxEntries: 8, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
			if disk {
				configuration = genericPersistentConfiguration(t, t.TempDir())
			}
			session, err := NewGenericSessionWithCapabilities(configuration, genericplan.DefaultPlanLimits(), sdkStorageCapabilities())
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			if !strings.HasSuffix(session.keys.compilerBuild, ":sdk-owned-storage-v1") {
				t.Fatal("SDK cache namespace lost")
			}
			lower := session.lower
			calls := 0
			session.lower = func(analysis compiler.Analysis, capabilities admission.Capabilities) (compiler.Result, error) {
				calls++
				if capabilities != (admission.Capabilities{}) {
					t.Fatal("session profile granted unauthenticated source capabilities")
				}
				return lower(analysis, capabilities)
			}
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified || !cold.Cache.Lowered || calls != 1 {
				t.Fatalf("cold: %v %+v", err, cold.Cache)
			}
			warm, err := session.Check(workspace)
			if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != OutcomeAllHit || calls != 1 {
				t.Fatalf("warm: %v %+v", err, warm.Cache)
			}
			writeFile(t, filepath.Join(root, "src", "main.fy"), strings.Replace(genericMain, "42", "storage_size<i32>() as i32", 1))
			rejected, err := session.Check(loadCacheWorkspace(t, root))
			if err != nil || rejected.Compiler.Verified || rejected.Cache.Hits != 0 {
				t.Fatalf("unauthenticated private query: %v %+v", err, rejected)
			}
			writeFile(t, filepath.Join(root, "src", "main.fy"), genericMain)
		})
	}
}
