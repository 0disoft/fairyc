package workspacecompiler

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/compiler"
)

type graphCacheProcessRequest struct{ Root, Cache, Report string }
type graphCacheProcessReport struct {
	Cache                 Report
	Body, Program, Source string
}

func TestStateGraphSessionPersistenceProcess(t *testing.T) {
	const childEnvironment = "FAIRYC_STATE_GRAPH_CACHE_PROCESS"
	if request := os.Getenv(childEnvironment); request != "" {
		var input graphCacheProcessRequest
		if err := json.Unmarshal([]byte(request), &input); err != nil {
			t.Fatal(err)
		}
		session, err := NewExperimentalStateGraphSession(persistentTestConfiguration(input.Cache, "test-state-graph-build-a"))
		if err != nil {
			t.Fatal(err)
		}
		defer session.Close()
		session.lower = func(context.Context, compiler.Analysis) (compiler.Result, error) {
			t.Fatal("fresh process cache hit lowered MIR")
			return compiler.Result{}, nil
		}
		workspace := loadCacheWorkspace(t, input.Root)
		result, err := session.Check(workspace)
		if err != nil || result.Cache.Outcome != OutcomeAllHit || result.Cache.Hits != 1 || result.Cache.Lowered {
			t.Fatalf("fresh process cache %+v: %v", result.Cache, err)
		}
		prepared := currentGraphTransport(t, workspace, result.Compiler)
		source, err := prepared.Source()
		if err != nil {
			t.Fatal(err)
		}
		output, err := json.Marshal(graphCacheProcessReport{Cache: result.Cache, Body: prepared.BodyHash().String(), Program: fmt.Sprintf("%x", prepared.ProgramHash()), Source: fmt.Sprintf("%x", sha256.Sum256([]byte(source)))})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(input.Report, output, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []string{"transaction.fy", "helpers/transaction.fy", "borrows/transaction.fy", "dynamic/transaction.fy"} {
		t.Run(fixture, func(t *testing.T) {
			root, _ := stateGraphCacheWorkspace(t, fixture)
			configuration := graphCacheConfiguration(t, StorageDisk)
			session, err := NewExperimentalStateGraphSession(configuration)
			if err != nil {
				t.Fatal(err)
			}
			defer session.Close()
			workspace := loadCacheWorkspace(t, root)
			cold, err := session.Check(workspace)
			if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 1 {
				t.Fatal("parent cold state cache", err)
			}
			prepared := currentGraphTransport(t, workspace, cold.Compiler)
			source, err := prepared.Source()
			if err != nil {
				t.Fatal(err)
			}
			if err := session.Close(); err != nil {
				t.Fatal(err)
			}
			reportPath := filepath.Join(t.TempDir(), "restored.json")
			request, err := json.Marshal(graphCacheProcessRequest{root, configuration.Persistent.Root, reportPath})
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, self, "-test.run=^TestStateGraphSessionPersistenceProcess$", "-test.count=1")
			command.Env = append(os.Environ(), childEnvironment+"="+string(request))
			if output, err := command.CombinedOutput(); err != nil {
				t.Fatalf("cache process: %v: %s", err, output)
			}
			data, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatal(err)
			}
			var restored graphCacheProcessReport
			if err := json.Unmarshal(data, &restored); err != nil {
				t.Fatal(err)
			}
			if restored.Cache.Hits != 1 || restored.Cache.Lowered || restored.Body != prepared.BodyHash().String() || restored.Program != fmt.Sprintf("%x", prepared.ProgramHash()) || restored.Source != fmt.Sprintf("%x", sha256.Sum256([]byte(source))) {
				t.Fatal("fresh process changed current-source state graph identities or C")
			}
		})
	}
}
