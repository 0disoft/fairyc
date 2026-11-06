package contractstatenative

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

// GraphArtifact is verified operation C, not an input/output or persistence owner.
type GraphArtifact struct {
	source  string
	layout  contractstate.LayoutHash
	body    contractstatemir.StateBodyHash
	program [32]byte
	traps   []string
	valid   bool
}

func (artifact GraphArtifact) Source() string                           { return artifact.source }
func (artifact GraphArtifact) ProgramHash() [32]byte                    { return artifact.program }
func (artifact GraphArtifact) LayoutHash() contractstate.LayoutHash     { return artifact.layout }
func (artifact GraphArtifact) BodyHash() contractstatemir.StateBodyHash { return artifact.body }

type graphBody struct {
	operations []contractstatemir.Operation
	entry      contractstatemir.BlockID
	blocks     []contractstatemir.Block
	exit       contractstatemir.ExitKind
	trap       string
}

type graphEmitter struct {
	strings.Builder
	fields  []contractstatemir.LayoutField
	helpers map[string]int
	traps   map[string]int
	site    int
}

const graphNativeValueSlots = 4096

// EmitGraph preserves verified graph/legacy bodies and their shared helper
// capability. The separate literal emitter and its sealed protocol stay closed
// to these graphs until their own framing/CAS admission is implemented.
func EmitGraph(handoff Handoff, id target.ID) (GraphArtifact, error) {
	if id != target.X8664WindowsMSVC || handoff.bodyHash == (contractstatemir.StateBodyHash{}) {
		return GraphArtifact{}, fmt.Errorf("graph emission requires a prepared Windows owner")
	}
	p := handoff.program
	if failures := contractstatemir.Verify(p); len(failures) != 0 {
		return GraphArtifact{}, fmt.Errorf("graph emission requires verified MIR: %s", failures[0])
	}
	bodies := []graphBody{{p.Operations, p.Entry, p.Blocks, p.Exit, p.TrapCategory}}
	e := graphEmitter{fields: p.Layout.Fields, helpers: make(map[string]int), traps: make(map[string]int)}
	for index, helper := range p.Helpers {
		e.helpers[helper.Name] = index + 1
		bodies = append(bodies, graphBody{helper.Operations, helper.Entry, helper.Blocks, helper.Exit, helper.TrapCategory})
	}
	var traps []string
	totalSlots := 0
	for _, body := range bodies {
		values, borrows := graphSlots(body)
		totalSlots += len(values) + len(borrows)
		if body.trap != "" {
			traps = append(traps, body.trap)
		}
		for _, block := range body.blocks {
			if block.Terminator.TrapCategory != "" {
				traps = append(traps, block.Terminator.TrapCategory)
			}
		}
	}
	if totalSlots > graphNativeValueSlots {
		return GraphArtifact{}, fmt.Errorf("native graph exceeds %d value/borrow slots", graphNativeValueSlots)
	}
	slices.Sort(traps)
	traps = slices.Compact(traps)
	for index, trap := range traps {
		e.traps[trap] = index + 1
	}
	header, err := cstdlib.ContractStateGraphDeclarations()
	if err != nil {
		return GraphArtifact{}, err
	}
	e.WriteString(header)
	e.WriteString("\ntypedef int32_t (*fycs_graph_observer_v1)(void *, uint64_t, uint32_t, const uint8_t[32], uint8_t, const uint8_t *, size_t, uint8_t);\n")
	e.WriteString("static const uint8_t fygr_bool_v1[2] = {0u,1u};\n")
	for index := range bodies {
		fmt.Fprintf(&e, "static int32_t fygr_body_%d(fycs_graph_v1 *, size_t, fycs_graph_observer_v1, void *, uint8_t *, uint32_t *);\n", index)
	}
	for index, body := range bodies {
		e.body(index, body)
	}
	e.WriteString(`
int32_t fycs_graph_execute_v1(fycs_graph_v1 *context, size_t max_key,
    fycs_graph_observer_v1 observe, void *user, uint8_t *exit_kind, uint32_t *trap_kind) {
    int32_t status;
    if (context == NULL || context->transaction == NULL || observe == NULL ||
        exit_kind == NULL || trap_kind == NULL || max_key == 0u) { return FYCS_OV_INVALID; }
    if (context->transaction->state != 1u) { return FYCS_OV_CLOSED; }
    *exit_kind = 3u; *trap_kind = 0u;
    status = fygr_body_0(context,max_key,observe,user,exit_kind,trap_kind);
    if (status == 0 && *exit_kind == 0u) { return fycs_overlay_commit_v1(context->transaction); }
    (void)fycs_overlay_abort_v1(context->transaction);
    if (status != 0) { *exit_kind = 0u; *trap_kind = 0u; return status; }
    return *exit_kind == 1u || *exit_kind == 2u ? FYCS_OV_OK : FYCS_OV_INVALID;
}
`)
	artifact := GraphArtifact{source: e.String(), layout: p.Layout.Hash, body: handoff.bodyHash, traps: slices.Clone(traps), valid: true}
	metadata, _ := json.Marshal(traps)
	identity := append([]byte("fairyc.native-state-graph-entry/v1\x00"+string(id)+"\x00"), artifact.layout[:]...)
	identity = append(identity, artifact.body[:]...)
	identity = append(identity, metadata...)
	artifact.program = sha256.Sum256(append(identity, artifact.source...))
	return artifact, nil
}

func graphSlots(body graphBody) (map[contractstatemir.ValueID]int, map[contractstatemir.BorrowID]int) {
	values := make(map[contractstatemir.ValueID]int)
	borrows := make(map[contractstatemir.BorrowID]int)
	for _, block := range body.blocks {
		for _, parameter := range block.Parameters {
			values[parameter.ID] = len(values)
		}
		for _, op := range block.Operations {
			if op.Result != 0 {
				values[op.Result] = len(values)
			}
			if op.Found != 0 {
				values[op.Found] = len(values)
			}
			if op.BorrowResult != 0 {
				borrows[op.BorrowResult] = len(borrows)
			}
		}
	}
	return values, borrows
}

func (e *graphEmitter) checked(call string) {
	fmt.Fprintf(e, "    status = %s; if (status != 0) { goto finished; }\n", call)
}

func (e *graphEmitter) body(index int, body graphBody) {
	values, borrows := graphSlots(body)
	fmt.Fprintf(e, "static int32_t fygr_body_%d(fycs_graph_v1 *context, size_t max_key, fycs_graph_observer_v1 observe, void *user, uint8_t *exit_kind, uint32_t *trap_kind) {\n", index)
	e.WriteString("    int32_t status = 0; size_t mark = context->used;\n    (void)max_key; (void)observe; (void)user; (void)trap_kind; (void)fygr_bool_v1;\n")
	if len(values) != 0 {
		fmt.Fprintf(e, "    fycs_graph_value_v1 values[%d] = {{0}};\n", len(values))
	}
	if len(borrows) != 0 {
		fmt.Fprintf(e, "    fycs_graph_borrow_v1 borrows[%d] = {{0}};\n", len(borrows))
	}
	if len(body.blocks) == 0 {
		for _, op := range body.operations {
			e.operation(op, false, values, borrows)
		}
		e.exit(body.exit, body.trap)
	} else {
		fmt.Fprintf(e, "    goto block_%d;\n", body.entry)
		for _, block := range body.blocks {
			fmt.Fprintf(e, "block_%d:;\n", block.ID)
			for _, op := range block.Operations {
				e.operation(op, true, values, borrows)
			}
			e.checked("fycs_graph_step_v1(context,1u,0u)")
			term := block.Terminator
			switch term.Kind {
			case contractstatemir.TerminatorJump:
				e.edge(body.blocks[term.Target], term.Arguments, values)
			case contractstatemir.TerminatorBranch:
				fmt.Fprintf(e, "    if (values[%d].length != 1u || values[%d].bytes[0] > 1u) { status = FYCS_OV_CODEC; goto finished; }\n", values[term.Condition], values[term.Condition])
				fmt.Fprintf(e, "    if (values[%d].bytes[0] != 0u) {\n", values[term.Condition])
				e.edge(body.blocks[term.TrueTarget], term.TrueArguments, values)
				e.WriteString("    } else {\n")
				e.edge(body.blocks[term.FalseTarget], term.FalseArguments, values)
				e.WriteString("    }\n")
			case contractstatemir.TerminatorCommit:
				e.exit(contractstatemir.ExitCommit, "")
			case contractstatemir.TerminatorReject:
				e.exit(contractstatemir.ExitReject, "")
			case contractstatemir.TerminatorTrap:
				e.exit(contractstatemir.ExitTrap, term.TrapCategory)
			case contractstatemir.TerminatorReturn:
				e.exit(contractstatemir.ExitReturn, "")
			}
		}
	}
	e.WriteString("finished:\n    context->used = mark; return status;\n}\n")
}

func (e *graphEmitter) exit(kind contractstatemir.ExitKind, trap string) {
	exit := map[contractstatemir.ExitKind]int{contractstatemir.ExitCommit: 0, contractstatemir.ExitReject: 1, contractstatemir.ExitTrap: 2, contractstatemir.ExitReturn: 3}[kind]
	fmt.Fprintf(e, "    *exit_kind = %du; *trap_kind = %du; goto finished;\n", exit, e.traps[trap])
}

func (e *graphEmitter) edge(target contractstatemir.Block, arguments []contractstatemir.ValueID, values map[contractstatemir.ValueID]int) {
	// Verified DAG parameters have globally unique IDs unavailable in the
	// predecessor. Immutable slot assignment cannot overwrite an edge operand.
	for index, parameter := range target.Parameters {
		fmt.Fprintf(e, "    values[%d] = values[%d];\n", values[parameter.ID], values[arguments[index]])
	}
	fmt.Fprintf(e, "    goto block_%d;\n", target.ID)
}

func (e *graphEmitter) literal(name string, value contractstatemir.Value) {
	codec, _ := overlayCodec(value.TypeID)
	bytes := value.Encoded
	if len(bytes) == 0 {
		bytes = []byte{0}
	}
	fmt.Fprintf(e, "    static const uint8_t %s_bytes[%d] = {%s};\n    const fycs_graph_value_v1 %s = {%s_bytes,%du,%du};\n", name, len(bytes), cBytes(bytes), name, name, len(value.Encoded), codec)
}

func (e *graphEmitter) operation(op contractstatemir.Operation, graph bool, values map[contractstatemir.ValueID]int, borrows map[contractstatemir.BorrowID]int) {
	e.site++
	if op.Kind == contractstatemir.OperationCall {
		e.checked("fycs_graph_step_v1(context,1u,1u)")
		e.checked(fmt.Sprintf("fygr_body_%d(context,max_key,observe,user,exit_kind,trap_kind)", e.helpers[op.Callee]))
		e.WriteString("    if (*exit_kind != 3u) { goto finished; }\n")
		return
	}
	e.WriteString("    {\n")
	switch op.Kind {
	case contractstatemir.OperationBorrowView:
		e.checked(fmt.Sprintf("fycs_graph_borrow_view_v1(context,&borrows[%d],%du,%du,&borrows[%d])", borrows[op.BorrowInput], op.ViewOffset, op.ViewLength, borrows[op.BorrowResult]))
	case contractstatemir.OperationBorrowCopy:
		e.checked(fmt.Sprintf("fycs_graph_borrow_copy_v1(context,&borrows[%d],&values[%d])", borrows[op.BorrowInput], values[op.Result]))
	default:
		e.storage(op, graph, values, borrows)
	}
	charge := 0
	if graph {
		charge = 1
	}
	e.checked(fmt.Sprintf("fycs_graph_step_v1(context,%du,1u)", charge))
	e.WriteString("    }\n")
}

func (e *graphEmitter) storage(op contractstatemir.Operation, graph bool, values map[contractstatemir.ValueID]int, borrows map[contractstatemir.BorrowID]int) {
	field := e.fields[op.FieldIndex]
	codec, _ := overlayCodec(field.ValueTypeID)
	if field.Kind == contractstate.FieldCell {
		fmt.Fprintf(e, "    static const uint8_t key[32] = {%s};\n", cBytes(field.Prefix[:]))
	} else {
		fmt.Fprintf(e, "    static const uint8_t prefix[32] = {%s};\n    uint8_t key[32];\n", cBytes(field.Prefix[:]))
		key := "literal_key"
		if op.KeyInput != 0 {
			key = fmt.Sprintf("values[%d]", values[op.KeyInput])
		} else {
			e.literal(key, op.Key)
		}
		e.checked(fmt.Sprintf("fycs_graph_map_key_v1(prefix,&%s,max_key,key)", key))
	}
	switch op.Kind {
	case contractstatemir.OperationCellRead, contractstatemir.OperationMapRead, contractstatemir.OperationCellBorrow, contractstatemir.OperationMapBorrow:
		e.WriteString("    uint8_t found = 0u;\n")
		value := fmt.Sprintf("values[%d]", values[op.Result])
		borrow := op.Kind == contractstatemir.OperationCellBorrow || op.Kind == contractstatemir.OperationMapBorrow
		if borrow {
			value = fmt.Sprintf("borrows[%d].value", borrows[op.BorrowResult])
			e.checked(fmt.Sprintf("fycs_graph_borrow_v1_begin(context,key,%du,&borrows[%d],&found)", codec, borrows[op.BorrowResult]))
		} else {
			if !graph {
				e.WriteString("    fycs_graph_value_v1 read;\n")
				value = "read"
			}
			e.checked(fmt.Sprintf("fycs_graph_read_v1(context,key,%du,&%s,&found)", codec, value))
		}
		length := fmt.Sprintf("found != 0u ? %s.length : 0u", value)
		if borrow {
			length = value + ".length"
		}
		e.checked(fmt.Sprintf("observe(user,context->operation,%du,key,%du,%s.bytes,%s,found)", e.site, codec, value, length))
		if graph {
			fmt.Fprintf(e, "    values[%d].bytes = fygr_bool_v1+found; values[%d].length = 1u; values[%d].codec = FYCS_CODEC_BOOL;\n", values[op.Found], values[op.Found], values[op.Found])
		}
	case contractstatemir.OperationCellWrite, contractstatemir.OperationMapWrite:
		value := "literal_value"
		if op.ValueInput != 0 {
			value = fmt.Sprintf("values[%d]", values[op.ValueInput])
		} else {
			e.literal(value, op.Value)
		}
		e.checked(fmt.Sprintf("fycs_graph_write_v1(context,key,&%s,0u)", value))
	case contractstatemir.OperationCellDelete, contractstatemir.OperationMapDelete:
		e.checked("fycs_graph_write_v1(context,key,NULL,1u)")
	}
}
