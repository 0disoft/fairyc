// Command fairyc-state executes one experimental source state transaction through
// the Go MIR oracle and a bounded local store. It is not a chain deployment tool.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

const usage = `Experimental local Fairy state runner (Go verified-MIR execution)

Usage: fairyc-state <init|execute|snapshot> <existing-store-directory> <source.fy>

One source file and exactly one transaction are accepted. init exclusively creates
state; execute performs one attempt; snapshot never runs the transaction body.
Store policy: 16 MiB file, 65536 entries, 64 KiB/value.
Execution policy: 1024 writes, 1 MiB write bytes, 64 KiB/value, 1024-byte map keys.
Exit codes: 0 success, 2 invocation, 3 source rejection, 4 I/O/backend failure,
6 transaction rejection, 7 conflict/layout/busy, 8 language trap.
No retry, target flag, RPC, deployment or power-loss guarantee is provided.
`

type report struct {
	Schema       string                            `json:"schema"`
	Action       string                            `json:"action"`
	LayoutHash   string                            `json:"layoutHash"`
	Outcome      *contractstatemir.Outcome         `json:"outcome,omitempty"`
	TrapCategory string                            `json:"trapCategory,omitempty"`
	Snapshot     *contractstate.PersistentSnapshot `json:"snapshot,omitempty"`
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	if stdout == nil || stderr == nil {
		return 4
	}
	fail := func(code int, err error) int { _, _ = fmt.Fprintln(stderr, err); return code }
	if len(args) == 1 && args[0] == "--help" {
		if _, err := io.WriteString(stdout, usage); err != nil {
			return 4
		}
		return 0
	}
	if len(args) != 3 || (args[0] != "init" && args[0] != "execute" && args[0] != "snapshot") || args[1] == "" || filepath.Ext(args[2]) != ".fy" {
		return fail(2, errors.New(usage))
	}
	program, err := compileSource(args[2])
	if err != nil {
		var pathError *os.PathError
		if errors.As(err, &pathError) {
			return fail(4, err)
		}
		return fail(3, err)
	}
	layout, err := storageLayout(program.Layout)
	if err != nil {
		return fail(3, err)
	}
	limits := contractstate.LocalPersistenceLimits{MaxStateBytes: 16 << 20, MaxEntries: 65536, MaxValueBytes: 64 << 10}
	open := contractstate.OpenLocalPersistence
	if args[0] == "init" {
		open = contractstate.CreateLocalPersistence
	}
	store, err := open(args[1], layout, limits)
	if err != nil {
		return fail(storageExit(err), err)
	}
	defer store.Close()
	result := report{Schema: "fairyc.local-state-run/v1", Action: args[0], LayoutHash: layout.Hash().String()}
	code := 0
	if args[0] == "execute" {
		outcome, trap, err := execute(store, program)
		if err != nil {
			return fail(storageExit(err), err)
		}
		if trap != "" {
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

func compileSource(path string) (contractstatemir.Program, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return contractstatemir.Program{}, err
	}
	defer root.Close()
	name := filepath.Base(path)
	info, err := root.Lstat(name)
	if err != nil {
		return contractstatemir.Program{}, err
	}
	if !info.Mode().IsRegular() {
		return contractstatemir.Program{}, fmt.Errorf("source must be a regular file")
	}
	file, err := root.Open(name)
	if err != nil {
		return contractstatemir.Program{}, err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil {
		return contractstatemir.Program{}, err
	}
	if !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return contractstatemir.Program{}, fmt.Errorf("source identity changed")
	}
	limits := source.DefaultLimits()
	if opened.Size() > int64(limits.MaxBytes) {
		return contractstatemir.Program{}, fmt.Errorf("source size limit exceeded")
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(limits.MaxBytes)+1))
	if err != nil {
		return contractstatemir.Program{}, err
	}
	if len(data) > limits.MaxBytes {
		return contractstatemir.Program{}, fmt.Errorf("source size limit exceeded")
	}
	parsed, err := frontend.AnalyzeSyntax(source.NewFile(name, data), limits)
	if err != nil {
		return contractstatemir.Program{}, err
	}
	if parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		return contractstatemir.Program{}, fmt.Errorf("source syntax rejected")
	}
	compiled, err := compiler.CheckExperimentalState(semantic.File{Path: name, Module: parsed.AST})
	if err != nil {
		return contractstatemir.Program{}, err
	}
	if !compiled.Verified || len(compiled.StateTransactions) != 1 {
		return contractstatemir.Program{}, fmt.Errorf("source requires exactly one verified experimental state transaction")
	}
	return compiled.StateTransactions[0].Program, nil
}

func storageLayout(layout contractstatemir.Layout) (contractstate.Layout, error) {
	builder, err := contractstate.NewLayoutBuilder(layout.ContractUID)
	if err != nil {
		return contractstate.Layout{}, err
	}
	for _, field := range layout.Fields {
		switch field.Kind {
		case contractstate.FieldCell:
			_, err = contractstate.DeclareEncodedCell(builder, field.Name, field.ValueTypeID)
		case contractstate.FieldMap:
			_, err = contractstate.DeclareEncodedMap(builder, field.Name, field.KeyTypeID, field.ValueTypeID)
		default:
			return contractstate.Layout{}, fmt.Errorf("unknown layout field kind")
		}
		if err != nil {
			return contractstate.Layout{}, err
		}
	}
	result, err := builder.Build()
	if err != nil {
		return contractstate.Layout{}, err
	}
	if result.Hash() != layout.Hash {
		return contractstate.Layout{}, contractstate.ErrLayoutMismatch
	}
	return result, nil
}

func execute(store contractstate.PersistencePort, program contractstatemir.Program) (outcome contractstatemir.Outcome, category string, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if trap, ok := recovered.(contractstatemir.Trap); ok {
				category = trap.Category
			} else {
				panic(recovered)
			}
		}
	}()
	outcome, err = contractstatemir.ExecutePersistent(store, program, contractstate.Limits{MaxWrites: 1024, MaxWriteBytes: 1 << 20, MaxValueBytes: 64 << 10, MaxMapKeyBytes: 1024})
	return outcome, "", err
}

func storageExit(err error) int {
	if errors.Is(err, contractstate.ErrConflict) || errors.Is(err, contractstate.ErrLayoutMismatch) || errors.Is(err, contractstate.ErrPersistenceBusy) {
		return 7
	}
	return 4
}
