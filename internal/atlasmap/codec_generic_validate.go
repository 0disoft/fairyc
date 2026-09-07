package atlasmap

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func validSpecializationID(id string) bool {
	if len(id) != 67 || !strings.HasPrefix(id, "g0_") || id != strings.ToLower(id) {
		return false
	}
	_, err := hex.DecodeString(id[3:])
	return err == nil
}

func validateGenericPartition(partition Partition, files map[string]struct{}, keys map[semanticsnapshot.DeclarationKey]struct{}, instances map[string]string) error {
	instance := partition.data.header.PackageInstance
	declarations := map[semanticsnapshot.DeclarationKey]semanticsnapshot.Declaration{}
	templates := 0
	for _, declaration := range partition.data.declarations {
		declarations[declaration.Key] = declaration
		if len(declaration.TypeParameters) > 0 {
			templates++
		}
	}
	implementations := map[string]semanticsnapshot.SpecializationSummary{}
	type bindingKey struct {
		declaration semanticsnapshot.DeclarationKey
		arguments   string
	}
	bindings := map[bindingKey]bool{}
	all := partition.Functions()
	for i, item := range partition.data.specializations {
		if !validSpecializationID(item.ID) || i > 0 && partition.data.specializations[i-1].ID >= item.ID {
			return fmt.Errorf("specialization IDs are invalid, duplicated, or noncanonical")
		}
		declaration, exists := declarations[item.Declaration]
		if !exists || item.Declaration.PackageInstance != instance || len(declaration.TypeParameters) == 0 || item.Function.Function != item.Declaration {
			return fmt.Errorf("specialization has no matching local generic declaration")
		}
		if len(item.TypeArguments) != len(declaration.TypeParameters) {
			return fmt.Errorf("specialization argument arity mismatch")
		}
		names := make([]string, len(item.TypeArguments))
		for j, argument := range item.TypeArguments {
			if err := validateTypeRef(argument, 0); err != nil {
				return err
			}
			if argument.Kind != semanticsnapshot.TypeBuiltin {
				return fmt.Errorf("specialization requires an admitted scalar argument")
			}
			switch argument.Name {
			case "bool", "i8", "i16", "i32", "i64", "isize", "u8", "u16", "u32", "u64", "usize", "f32", "f64":
			default:
				return fmt.Errorf("specialization requires an admitted scalar argument")
			}
			names[j] = argument.Name
		}
		binding := bindingKey{item.Declaration, strings.Join(names, ",")}
		if bindings[binding] {
			return fmt.Errorf("specialization repeats a declaration and argument binding")
		}
		bindings[binding] = true
		if err := validateFunction(item.Function, instance, files, keys, instances, true); err != nil {
			return err
		}
		implementations[item.ID] = item
		all = append(all, item.Function)
	}
	for _, function := range all {
		for _, call := range function.Calls {
			if call.Specialization != "" {
				if !validSpecializationID(call.Specialization) || call.TargetKind != semanticsnapshot.CallTargetResolved {
					return fmt.Errorf("invalid specialization call")
				}
				if call.Target.PackageInstance == instance {
					item, exists := implementations[call.Specialization]
					if !exists || item.Declaration != call.Target {
						return fmt.Errorf("specialization call has no matching implementation")
					}
				}
			} else if len(declarations[call.Target].TypeParameters) != 0 {
				return fmt.Errorf("generic call is missing its specialization target")
			}
		}
	}
	if templates > 0 {
		ledger := map[string]semanticsnapshot.Completeness{}
		for _, entry := range partition.data.completeness {
			ledger[entry.Relation] = entry
		}
		bodies, concrete := ledger["generic-template-bodies"], ledger["generic-specializations"]
		if bodies.Status != semanticsnapshot.CompletenessUnavailable || bodies.Unknown < templates ||
			concrete.Status != semanticsnapshot.CompletenessExact || concrete.Resolved < len(implementations) {
			return fmt.Errorf("generic completeness omits template uncertainty or concrete implementations")
		}
	}
	return nil
}
