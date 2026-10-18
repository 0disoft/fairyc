// Package cbackend emits the constrained Stage 0 C reference transport.
package cbackend

import (
	"context"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

// Emit verifies MIR, rejects unsupported semantic families, and emits one deterministic C unit.
func Emit(program mir.Program, entry string, profile Profile) (Unit, error) {
	return emit(program, entry, profile, false)
}

func emit(program mir.Program, entry string, profile Profile, storageTransfers bool) (Unit, error) {
	return emitContext(context.Background(), program, entry, profile, storageTransfers)
}

func EmitContext(ctx context.Context, program mir.Program, entry string, profile Profile) (Unit, error) {
	return emitContext(ctx, program, entry, profile, false)
}
func EmitStorageTransfersContext(ctx context.Context, program mir.Program, entry string, profile Profile) (Unit, error) {
	return emitContext(ctx, program, entry, profile, true)
}

func emitContext(ctx context.Context, program mir.Program, entry string, profile Profile, storageTransfers bool) (result Unit, err error) {
	if err := ctx.Err(); err != nil {
		return Unit{}, err
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			result, err = Unit{}, canceled
		}
	}()
	plan, err := analyzeEmissionContext(ctx, program, entry, profile, storageTransfers)
	if err != nil {
		return Unit{}, err
	}
	return (&emitter{analysis: plan}).emitUnit()
}

func (value *emitter) emitUnit() (Unit, error) {
	profile := value.analysis.profile
	functions := value.analysis.functions
	foreignFunctions := value.analysis.foreignFunctions
	entry := functions[value.analysis.entryIndex].Name

	output := &value.output
	if err := value.emitPreamble(output); err != nil {
		return Unit{}, err
	}
	value.emitByteStringStorage(output)
	value.emitStringStorage(output)
	value.emitCStringStorage(output)
	for index := range foreignFunctions {
		value.emitForeignMacroGuard(output, index)
	}
	for index := range foreignFunctions {
		if err := value.emitForeignPrototype(output, index); err != nil {
			return Unit{}, err
		}
	}
	if len(foreignFunctions) != 0 {
		output.WriteByte('\n')
	}
	for index := range functions {
		if err := value.contextError(); err != nil {
			return Unit{}, err
		}
		if err := value.emitPrototype(output, index); err != nil {
			return Unit{}, err
		}
	}
	output.WriteByte('\n')
	for index := range functions {
		if err := value.contextError(); err != nil {
			return Unit{}, err
		}
		if err := value.emitFunction(output, index); err != nil {
			return Unit{}, err
		}
	}
	_, usesArgCount := value.analysis.usedForeignSymbols["fy_std_process_arg_count_v0"]
	_, usesArg := value.analysis.usedForeignSymbols["fy_std_process_arg_v0"]
	if profile.Target == target.X8664LinuxGNU && (usesArgCount || usesArg) {
		output.WriteString("extern void fy_std_process_set_args_v0(int, char **);\nint main(int argc, char **argv) {\n    fy_std_process_set_args_v0(argc, argv);\n")
	} else {
		output.WriteString("int main(void) {\n")
	}
	for index := range functions {
		if err := value.contextError(); err != nil {
			return Unit{}, err
		}
		if index != value.analysis.entryIndex {
			fmt.Fprintf(output, "    (void)&fy_fn_%d;\n", index)
		}
	}
	if value.analysis.nonReturning[functions[value.analysis.entryIndex].Symbol] {
		fmt.Fprintf(output, "    fy_fn_%d();\n}\n", value.analysis.entryIndex)
	} else {
		fmt.Fprintf(output, "    return (int)fy_fn_%d();\n}\n", value.analysis.entryIndex)
	}

	names := make([]string, len(functions))
	for index := range functions {
		if err := value.contextError(); err != nil {
			return Unit{}, err
		}
		names[index] = functions[index].Name
	}
	features := make([]string, 0, len(value.analysis.features))
	for feature := range value.analysis.features {
		features = append(features, feature)
	}
	slices.Sort(features)
	foreignSymbols := make([]string, 0, len(value.analysis.usedForeignSymbols))
	for symbol := range value.analysis.usedForeignSymbols {
		foreignSymbols = append(foreignSymbols, symbol)
	}
	slices.Sort(foreignSymbols)
	return Unit{
		Target: string(profile.Target), Dialect: profile.Dialect, Instrumentation: profile.Instrumentation,
		Entry: entry, Functions: names, ForeignSymbols: foreignSymbols, Features: features,
		DropSites: slices.Clone(value.analysis.dropSites), Source: output.String(),
	}, nil
}
