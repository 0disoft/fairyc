package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/target"
)

type nativeReport struct {
	report
	Engine      string `json:"engine"`
	Profile     string `json:"profile"`
	BodyHash    string `json:"bodyHash"`
	ProgramHash string `json:"programHash"`
	// An acknowledged but inconsistent receipt may already be published.
	Publication *contractstate.PersistentReceipt `json:"publication,omitempty"`
	Failure     string                           `json:"failure,omitempty"`
}

func runNative(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	fail := func(code int, err error) int { _, _ = fmt.Fprintln(stderr, err); return code }
	if len(args) != 7 || (args[0] != "init" && args[0] != "execute" && args[0] != "snapshot") ||
		args[1] == "" || filepath.Ext(args[2]) != ".fy" || args[3] != "--engine" || args[4] != "native-graph" ||
		args[5] != "--profile" || (args[6] != "low" && args[6] != "optimized") {
		return fail(2, errors.New(usage))
	}
	if ctx == nil {
		return fail(4, errors.New("native graph execution requires a context"))
	}
	if err := ctx.Err(); err != nil {
		return fail(4, err)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return fail(4, errors.New("native graph execution requires Windows amd64 with x64 MSVC"))
	}
	compiled, err := compileSourceResult(args[2], compiler.CheckExperimentalStateGraph)
	if err != nil {
		var pathError *os.PathError
		if errors.As(err, &pathError) {
			return fail(4, err)
		}
		return fail(3, err)
	}
	owner := compiled.StateTransactions[0]
	layout, err := storageLayout(owner.Program.Layout)
	if err != nil {
		return fail(3, err)
	}
	coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: owner.Program.Layout.ContractUID, VersionIdentity: "experimental-state-graph-source/1"})
	if err != nil {
		return fail(3, err)
	}
	handoff, err := contractstatenative.PrepareCompiled(compiled, owner.Package, owner.Function, coordinate)
	if err != nil {
		return fail(3, err)
	}
	prepared, err := contractstatenative.PrepareGraphTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		return fail(3, err)
	}
	profile := cdriver.LowProfile
	if args[6] == "optimized" {
		profile = cdriver.OptimizedProfile
	}
	var executable contractstatenative.GraphExecutable
	if err := ctx.Err(); err != nil {
		return fail(4, err)
	}
	if args[0] == "execute" {
		// A build failure cannot create, open, observe or publish persistent state.
		executable, err = contractstatenative.BuildGraph(prepared, profile)
		if err != nil {
			return fail(4, err)
		}
	}
	if err := ctx.Err(); err != nil {
		return fail(4, err)
	}
	open := contractstate.OpenLocalPersistence
	if args[0] == "init" {
		open = contractstate.CreateLocalPersistence
	}
	store, err := open(args[1], layout, localStateLimits())
	if err != nil {
		return fail(storageExit(err), err)
	}
	defer store.Close()
	result := nativeReport{report: report{Schema: "fairyc.local-state-run/v2", Action: args[0], LayoutHash: layout.Hash().String()},
		Engine: "native-graph", Profile: profile.Name, BodyHash: prepared.BodyHash().String(), ProgramHash: fmt.Sprintf("%x", prepared.ProgramHash())}
	code := 0
	if args[0] == "execute" {
		outcome, trap, err := executeNative(ctx, store, executable)
		if err != nil {
			var acknowledged contractstatenative.PublicationReceiptError
			if !errors.As(err, &acknowledged) {
				return fail(storageExit(err), err)
			}
			// Preserve the acknowledgement and outcome in the report. Never make
			// this look like an uncommitted, replayable process failure.
			result.Outcome, result.Publication, result.Failure = &outcome, &acknowledged.Receipt, err.Error()
			code = fail(4, err)
		} else if trap != "" {
			result.TrapCategory, code = trap, 8
		} else {
			result.Outcome = &outcome
			if outcome.Exit == contractstatemir.ExitReject {
				code = 6
			}
		}
	} else {
		snapshot, err := store.Snapshot(layout.Hash())
		if err != nil {
			return fail(storageExit(err), err)
		}
		result.Snapshot = &snapshot
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		return fail(4, fmt.Errorf("report output failed; an executed commit may already be published: %w", err))
	}
	return code
}

func executeNative(ctx context.Context, store contractstate.PersistencePort, executable contractstatenative.GraphExecutable) (outcome contractstatemir.Outcome, category string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if trap, ok := recovered.(contractstatemir.Trap); ok {
				category = trap.Category
			} else {
				panic(recovered)
			}
		}
	}()
	outcome, err = contractstatenative.ExecuteGraphPersistent(ctx, store, executable, stateExecutionLimits())
	return outcome, "", err
}
