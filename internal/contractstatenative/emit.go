// Package contractstatenative owns the private verified-state-MIR to C handoff.
package contractstatenative

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/target"
)

// Handoff can be created only by Prepare, which binds verified state MIR to its computed body
// identity before any target backend sees it.
type Handoff struct {
	program  contractstatemir.Program
	bodyHash contractstatemir.StateBodyHash
}

// Artifact is one compiler-private C unit tied to exact verified layout and body identities.
// It is not a stable runtime ABI or a supported target artifact.
type Artifact struct {
	Source     string
	LayoutHash contractstate.LayoutHash
	BodyHash   contractstatemir.StateBodyHash
}

// Prepare creates one exact compiler-to-native handoff from source-independent HIR, its stable
// package coordinate, and one transaction owner. Verification and body hashing share this boundary.
func Prepare(input hir.Package, packageID hir.PackageID, function hir.SymbolID, coordinate packagegraph.CoordinateID) (Handoff, error) {
	if packageID == 0 || function == 0 || coordinate == (packagegraph.CoordinateID{}) {
		return Handoff{}, fmt.Errorf("contract-state native handoff identity is incomplete")
	}
	lowered, err := contractstatemir.LowerHIRTransactions(input)
	if err != nil {
		return Handoff{}, fmt.Errorf("lower contract-state native handoff: %w", err)
	}
	hashes, err := contractstatemir.ComputeHIRStateBodyHashes(input, []hir.PackageID{packageID}, map[hir.PackageID]packagegraph.CoordinateID{packageID: coordinate})
	if err != nil {
		return Handoff{}, fmt.Errorf("identify contract-state native handoff: %w", err)
	}
	for _, transaction := range lowered {
		if transaction.Package == packageID && transaction.Function == function {
			return Handoff{program: transaction.Program, bodyHash: hashes[packageID]}, nil
		}
	}
	return Handoff{}, fmt.Errorf("contract-state native handoff owner %d:%d is absent", packageID, function)
}

// Emit lowers the first bounded native contract-state slice. It deliberately accepts only one
// straight-line u64 cell transaction and fails closed for every broader state-MIR form.
func Emit(handoff Handoff, targetID target.ID) (Artifact, error) {
	if targetID != target.X8664WindowsMSVC {
		return Artifact{}, fmt.Errorf("contract-state native transport does not admit target %q", targetID)
	}
	if handoff.bodyHash == (contractstatemir.StateBodyHash{}) {
		return Artifact{}, fmt.Errorf("contract-state native transport requires a prepared exact body handoff")
	}
	program, bodyHash := handoff.program, handoff.bodyHash
	if failures := contractstatemir.Verify(program); len(failures) != 0 {
		return Artifact{}, fmt.Errorf("contract-state native transport requires verified MIR: %s", failures[0])
	}
	if len(program.Blocks) != 0 || len(program.Helpers) != 0 || len(program.Layout.Fields) != 1 {
		return Artifact{}, fmt.Errorf("contract-state native transport admits one straight-line single-cell program")
	}
	field := program.Layout.Fields[0]
	if field.Index != 0 || field.Kind != contractstate.FieldCell || field.ValueTypeID != contractstate.U64.TypeID() {
		return Artifact{}, fmt.Errorf("contract-state native transport admits only field-zero u64 cell storage")
	}

	var source strings.Builder
	writePrelude(&source, program.Layout.Hash, bodyHash, program.TrapCategory)
	for index, operation := range program.Operations {
		if operation.FieldIndex != 0 {
			return Artifact{}, fmt.Errorf("contract-state native operation %d names unsupported field %d", index, operation.FieldIndex)
		}
		switch operation.Kind {
		case contractstatemir.OperationCellRead:
			fmt.Fprintf(&source, "    if (fycs_native_cell_read_v0(&transaction, fycs_native_last_read_v0, &fycs_native_last_read_length_v0, &fycs_native_last_read_found_v0) != 0) { fycs_native_rollback_v0(&transaction); return -20; }\n")
		case contractstatemir.OperationCellWrite:
			name := fmt.Sprintf("fycs_native_value_%d_v0", index)
			fmt.Fprintf(&source, "    static const uint8_t %s[] = {%s};\n", name, cBytes(operation.Value.Encoded))
			fmt.Fprintf(&source, "    if (fycs_native_cell_write_v0(&transaction, %s, %du) != 0) { fycs_native_rollback_v0(&transaction); return -21; }\n", name, len(operation.Value.Encoded))
		case contractstatemir.OperationCellDelete:
			fmt.Fprint(&source, "    if (fycs_native_cell_delete_v0(&transaction) != 0) { fycs_native_rollback_v0(&transaction); return -22; }\n")
		default:
			return Artifact{}, fmt.Errorf("contract-state native operation %d kind %q is not admitted", index, operation.Kind)
		}
	}
	switch program.Exit {
	case contractstatemir.ExitCommit:
		fmt.Fprint(&source, "    if (fycs_native_commit_v0(&transaction) != 0) { return -23; }\n    return 0;\n}\n")
	case contractstatemir.ExitReject:
		fmt.Fprint(&source, "    fycs_native_rollback_v0(&transaction);\n    return 1;\n}\n")
	case contractstatemir.ExitTrap:
		fmt.Fprint(&source, "    fycs_native_rollback_v0(&transaction);\n    return 2;\n}\n")
	default:
		return Artifact{}, fmt.Errorf("contract-state native exit %q is not admitted", program.Exit)
	}
	return Artifact{Source: source.String(), LayoutHash: program.Layout.Hash, BodyHash: bodyHash}, nil
}

func writePrelude(source *strings.Builder, layoutHash contractstate.LayoutHash, bodyHash contractstatemir.StateBodyHash, trapCategory string) {
	fmt.Fprint(source, `#include <stddef.h>
#include <stdint.h>

#define FYCS_NATIVE_VALUE_CAPACITY_V0 64u

typedef struct fycs_native_store_v0 {
    uint8_t value[FYCS_NATIVE_VALUE_CAPACITY_V0];
    size_t length;
    uint8_t present;
    uint8_t layout_hash[32];
    uint8_t body_hash[32];
} fycs_native_store_v0;

typedef struct fycs_native_transaction_v0 {
    fycs_native_store_v0 *owner;
    uint8_t value[FYCS_NATIVE_VALUE_CAPACITY_V0];
    size_t length;
    uint8_t present;
    uint8_t open;
} fycs_native_transaction_v0;

void fycs_native_store_initialize_v0(fycs_native_store_v0 *, const uint8_t[32], const uint8_t[32]);
int32_t fycs_native_begin_v0(fycs_native_store_v0 *, const uint8_t[32], const uint8_t[32], fycs_native_transaction_v0 *);
int32_t fycs_native_cell_read_v0(const fycs_native_transaction_v0 *, uint8_t[FYCS_NATIVE_VALUE_CAPACITY_V0], size_t *, uint8_t *);
int32_t fycs_native_cell_write_v0(fycs_native_transaction_v0 *, const uint8_t *, size_t);
int32_t fycs_native_cell_delete_v0(fycs_native_transaction_v0 *);
int32_t fycs_native_commit_v0(fycs_native_transaction_v0 *);
void fycs_native_rollback_v0(fycs_native_transaction_v0 *);
size_t fycs_native_live_resources_v0(void);

`)
	fmt.Fprintf(source, "const uint8_t fycs_native_layout_hash_v0[32] = {%s};\n", cBytes(layoutHash[:]))
	fmt.Fprintf(source, "const uint8_t fycs_native_body_hash_v0[32] = {%s};\n", cBytes(bodyHash[:]))
	fmt.Fprintf(source, "const uint8_t fycs_native_trap_category_v0[] = {%s, 0x00};\n", cBytes([]byte(trapCategory)))
	fmt.Fprint(source, `uint8_t fycs_native_last_read_v0[FYCS_NATIVE_VALUE_CAPACITY_V0] = {0};
size_t fycs_native_last_read_length_v0 = 0u;
uint8_t fycs_native_last_read_found_v0 = 0u;

int32_t fycs_native_entry_v0(fycs_native_store_v0 *store) {
    fycs_native_transaction_v0 transaction;
    if (store == NULL || (store->present != 0u && store->length != 8u)) {
        return -10;
    }
    if (fycs_native_begin_v0(store, fycs_native_layout_hash_v0, fycs_native_body_hash_v0, &transaction) != 0) {
        return -10;
    }
`)
}

func cBytes(values []byte) string {
	if len(values) == 0 {
		return "0x00"
	}
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = fmt.Sprintf("0x%02x", value)
	}
	return strings.Join(parts, ", ")
}
