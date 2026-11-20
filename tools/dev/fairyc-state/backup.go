package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/semantic"
)

func runStateBackup(ctx context.Context, args []string, stdout io.Writer, fail func(int, error) int) int {
	if ctx == nil {
		return fail(4, errors.New("state backup and restore require a context"))
	}
	if err := ctx.Err(); err != nil {
		return fail(4, err)
	}
	compiled, err := compileSourceResult(args[2], func(file semantic.File) (compiler.Result, error) {
		return compiler.CheckExperimentalStateGraphContext(ctx, file)
	})
	if err != nil {
		var pathError *os.PathError
		if errors.As(err, &pathError) || ctx.Err() != nil {
			return fail(4, err)
		}
		return fail(3, err)
	}
	layout, err := storageLayout(compiled.StateTransactions[0].Program.Layout)
	if err != nil {
		return fail(3, err)
	}
	if err := ctx.Err(); err != nil {
		return fail(4, err)
	}
	if err := transferLocalState(args[0], args[1], args[3], layout); err != nil {
		return fail(storageExit(err), err)
	}
	result := report{Schema: "fairyc.local-state-run/v1", Action: args[0], LayoutHash: layout.Hash().String()}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		return fail(4, fmt.Errorf("report output failed after %s completed: %w", args[0], err))
	}
	return 0
}

func transferLocalState(action, storeDirectory, backupDirectory string, layout contractstate.Layout) error {
	if action == "restore" {
		store, err := contractstate.RestoreLocalPersistence(storeDirectory, backupDirectory, layout, localStateLimits())
		if err != nil {
			return err
		}
		_ = store.Close() // Publication already succeeded.
		return nil
	}
	store, err := contractstate.OpenLocalPersistence(storeDirectory, layout, localStateLimits())
	if err != nil {
		return err
	}
	defer store.Close()
	return store.BackupTo(layout.Hash(), backupDirectory)
}
