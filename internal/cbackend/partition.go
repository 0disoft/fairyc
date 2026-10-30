package cbackend

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/mir"
)

// BuildUnit keeps small builds in one translation unit. Larger builds carry a
// private shared header and function-boundary sources, each including it once.
// Literal storage and callback adapters have a single definition across sources.
type BuildUnit struct {
	Unit    Unit
	Header  string
	Sources []string
}

const DefaultPartitionBytes = 1 << 20

func EmitBuild(program mir.Program, entry string, profile Profile, storageTransfers bool) (BuildUnit, error) {
	return emitBuild(context.Background(), program, entry, profile, storageTransfers, DefaultPartitionBytes)
}

func EmitBuildContext(ctx context.Context, program mir.Program, entry string, profile Profile, storageTransfers bool, partitionBytes int) (BuildUnit, error) {
	return emitBuild(ctx, program, entry, profile, storageTransfers, partitionBytes)
}

func emitBuild(ctx context.Context, program mir.Program, entry string, profile Profile, storageTransfers bool, budget int) (BuildUnit, error) {
	if budget <= 0 || budget > 2<<20 {
		return BuildUnit{}, fmt.Errorf("invalid C partition budget")
	}
	plan, err := analyzeEmissionContext(ctx, program, entry, profile, storageTransfers)
	if err != nil {
		return BuildUnit{}, err
	}
	e := &emitter{analysis: plan, partitioned: true}
	var bodies []string
	total := 0
	for index := range plan.functions {
		if err := ctx.Err(); err != nil {
			return BuildUnit{}, err
		}
		var body strings.Builder
		if err := e.emitFunction(&body, index); err != nil {
			return BuildUnit{}, err
		}
		bodies = append(bodies, body.String())
		total += body.Len()
	}
	if total <= budget {
		unit, err := (&emitter{analysis: plan}).emitUnit()
		return BuildUnit{Unit: unit}, err
	}
	var header strings.Builder
	header.WriteString("#ifndef FY_BOOTSTRAP_PROJECT_H\n#define FY_BOOTSTRAP_PROJECT_H\n")
	if err := e.emitPreamble(&header); err != nil {
		return BuildUnit{}, err
	}
	for index := range plan.foreignFunctions {
		e.emitForeignMacroGuard(&header, index)
	}
	for index := range plan.foreignFunctions {
		if err := e.emitForeignPrototype(&header, index); err != nil {
			return BuildUnit{}, err
		}
	}
	for index := range plan.functions {
		if err := e.emitPrototype(&header, index); err != nil {
			return BuildUnit{}, err
		}
	}
	for _, index := range e.callbackTargets() {
		if plan.nonReturning[plan.functions[index].Symbol] {
			header.WriteString("_Noreturn ")
		}
		fmt.Fprintf(&header, "int32_t fy_cb_%d(int32_t, void *);\n", index)
	}
	var literals []string
	for _, storage := range []map[string]string{plan.byteStringStorage, plan.stringStorage, plan.cStringStorage} {
		symbols := make([]string, 0, len(storage))
		for symbol := range storage {
			symbols = append(symbols, symbol)
		}
		slices.Sort(symbols)
		for _, symbol := range symbols {
			value := storage[symbol]
			length := max(1, len(value))
			fmt.Fprintf(&header, "extern const uint8_t %s[%d];\n", symbol, length)
			var definition strings.Builder
			emitStaticStorageLinkage(&definition, map[string]string{symbol: value}, true, "")
			literals = append(literals, definition.String())
		}
	}
	header.WriteString("#endif\n")
	const include = "#include \"fy_bootstrap.h\"\n"
	sources := make([]string, 0)
	appendGroups := func(fragments []string) {
		var group strings.Builder
		for _, fragment := range fragments {
			if group.Len() != 0 && group.Len()+len(fragment) > budget {
				sources = append(sources, include+group.String())
				group.Reset()
			}
			group.WriteString(fragment)
		}
		if group.Len() != 0 {
			sources = append(sources, include+group.String())
		}
	}
	appendGroups(bodies)
	appendGroups(literals)
	var entrySource strings.Builder
	e.emitCallbackAdapters(&entrySource)
	if err := e.emitEntry(&entrySource); err != nil {
		return BuildUnit{}, err
	}
	sources = append(sources, include+entrySource.String())
	unit, err := e.unitMetadata("")
	if err := ctx.Err(); err != nil {
		return BuildUnit{}, err
	}
	return BuildUnit{Unit: unit, Header: header.String(), Sources: sources}, err
}
