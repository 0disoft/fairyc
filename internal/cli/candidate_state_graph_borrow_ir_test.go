package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
)

type stateGraphBorrowTrial struct {
	name    string
	program contractstatemir.Program
	want    bool
	// A helper body is checked in its actual helper context; no helper proof is
	// manufactured by the root-only sealed factory used in stateGraphIRProbe.
	body int
}

func stateGraphBorrowBase(t *testing.T, mapped bool) contractstatemir.Program {
	t.Helper()
	p := stateGraphIRBase(t)
	borrow := contractstatemir.Operation{Kind: contractstatemir.OperationCellBorrow, Capability: p.Capability, FieldIndex: 5, BorrowResult: 1, Found: 1, Cost: 1}
	if mapped {
		borrow.Kind, borrow.FieldIndex = contractstatemir.OperationMapBorrow, 8
		borrow.Key = contractstatemir.Value{TypeID: contractstate.Text.TypeID(), Encoded: []byte("alpha")}
	}
	p.Blocks = []contractstatemir.Block{{Operations: []contractstatemir.Operation{
		borrow,
		{Kind: contractstatemir.OperationBorrowView, Capability: p.Capability, BorrowInput: 1, BorrowResult: 2, ViewOffset: 1, ViewLength: 3, Cost: 1},
		{Kind: contractstatemir.OperationBorrowCopy, Capability: p.Capability, BorrowInput: 2, Result: 2, Cost: 1},
		{Kind: contractstatemir.OperationCellWrite, Capability: p.Capability, FieldIndex: 5, ValueInput: 2, Cost: 1},
	}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}}}
	return p
}

func stateGraphBorrowInsert(p *contractstatemir.Program, at int, ops ...contractstatemir.Operation) {
	before := append([]contractstatemir.Operation(nil), p.Blocks[0].Operations[:at]...)
	before = append(before, ops...)
	p.Blocks[0].Operations = append(before, p.Blocks[0].Operations[at:]...)
}

func stateGraphBorrowTrials(t *testing.T) []stateGraphBorrowTrial {
	t.Helper()
	var trials []stateGraphBorrowTrial
	add := func(name string, mapped, want bool, mutate func(*contractstatemir.Program)) {
		p := stateGraphBorrowBase(t, mapped)
		if mutate != nil {
			mutate(&p)
		}
		trials = append(trials, stateGraphBorrowTrial{name: name, program: p, want: want})
	}
	add("cell-view-copy-separate-id-namespaces", false, true, nil)
	add("map-view-copy", true, true, nil)
	for codec := uint64(1); codec <= 7; codec++ {
		add(fmt.Sprintf("cell-codec-%d-runtime-range-deferred", codec), false, true, func(p *contractstatemir.Program) {
			p.Blocks[0].Operations[0].FieldIndex = uint32(codec - 1)
			p.Blocks[0].Operations[1].ViewOffset = ^uint32(0)
			p.Blocks[0].Operations[1].ViewLength = ^uint32(0)
			p.Blocks[0].Operations[3].FieldIndex = uint32(codec - 1)
		})
	}
	add("nested-view-keeps-origin", false, true, func(p *contractstatemir.Program) {
		stateGraphBorrowInsert(p, 2, contractstatemir.Operation{Kind: contractstatemir.OperationBorrowView, Capability: p.Capability, BorrowInput: 2, BorrowResult: 3, ViewOffset: 10, ViewLength: 99, Cost: 1})
		p.Blocks[0].Operations[3].BorrowInput = 3
	})
	add("read-only-root", false, true, func(p *contractstatemir.Program) {
		p.Effects = []hir.Effect{hir.EffectStorageRead}
		p.Blocks[0].Operations = p.Blocks[0].Operations[:3]
	})
	add("handles-at-1024-operation-limit", false, true, func(p *contractstatemir.Program) {
		p.Effects = []hir.Effect{hir.EffectStorageRead}
		p.Blocks[0].Operations = p.Blocks[0].Operations[:1]
		for id := contractstatemir.BorrowID(2); id <= 1024; id++ {
			p.Blocks[0].Operations = append(p.Blocks[0].Operations, contractstatemir.Operation{Kind: contractstatemir.OperationBorrowView, Capability: p.Capability, BorrowInput: id - 1, BorrowResult: id, ViewLength: 1, Cost: 1})
		}
	})
	add("map-empty-literal-key", true, true, func(p *contractstatemir.Program) {
		p.Blocks[0].Operations[0].Key.Encoded = []byte{}
	})
	add("map-u32-key", true, true, func(p *contractstatemir.Program) {
		p.Blocks[0].Operations[0].FieldIndex = 7
		p.Blocks[0].Operations[0].Key = contractstatemir.Value{TypeID: contractstate.U32.TypeID(), Encoded: []byte{0, 0, 0, 7}}
		p.Blocks[0].Operations[3].FieldIndex = 1
	})
	for name, mutate := range map[string]func(*contractstatemir.Operation){
		"zero-borrow-id":      func(op *contractstatemir.Operation) { op.BorrowResult = 0 },
		"missing-found":       func(op *contractstatemir.Operation) { op.Found = 0 },
		"cell-carries-result": func(op *contractstatemir.Operation) { op.Result = 9 },
		"cell-carries-key": func(op *contractstatemir.Operation) {
			op.Key = contractstatemir.Value{TypeID: contractstate.Text.TypeID(), Encoded: []byte("a")}
		},
		"cell-carries-input":  func(op *contractstatemir.Operation) { op.BorrowInput = 1 },
		"cell-carries-offset": func(op *contractstatemir.Operation) { op.ViewOffset = 1 },
		"cell-carries-length": func(op *contractstatemir.Operation) { op.ViewLength = 1 },
		"cell-names-map":      func(op *contractstatemir.Operation) { op.FieldIndex = 8 },
		"cell-field-outside":  func(op *contractstatemir.Operation) { op.FieldIndex = 99 },
		"borrow-zero-cap":     func(op *contractstatemir.Operation) { op.Capability = contractstatemir.TransactionCapability{} },
		"borrow-cost-zero":    func(op *contractstatemir.Operation) { op.Cost = 0 },
		"borrow-cost-two":     func(op *contractstatemir.Operation) { op.Cost = 2 },
		"borrow-carries-write": func(op *contractstatemir.Operation) {
			op.Value = contractstatemir.Value{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte{}}
		},
		"borrow-value-input":  func(op *contractstatemir.Operation) { op.ValueInput = 9 },
		"borrow-carries-call": func(op *contractstatemir.Operation) { op.Callee = "main::unused" },
	} {
		add(name, false, false, func(p *contractstatemir.Program) { mutate(&p.Blocks[0].Operations[0]) })
	}
	add("missing-read-effect", false, false, func(p *contractstatemir.Program) { p.Effects = []hir.Effect{hir.EffectStorageWrite} })
	for name, mutate := range map[string]func(*contractstatemir.Operation){
		"map-names-cell":   func(op *contractstatemir.Operation) { op.FieldIndex = 5 },
		"map-missing-key":  func(op *contractstatemir.Operation) { op.Key = contractstatemir.Value{} },
		"map-key-type":     func(op *contractstatemir.Operation) { op.Key.TypeID = contractstate.Bytes.TypeID() },
		"map-key-encoding": func(op *contractstatemir.Operation) { op.Key.Encoded = []byte{0xff} },
		"map-dynamic-key":  func(op *contractstatemir.Operation) { op.Key = contractstatemir.Value{}; op.KeyInput = 1 },
	} {
		add(name, true, false, func(p *contractstatemir.Program) { mutate(&p.Blocks[0].Operations[0]) })
	}
	for index, label := range map[int]string{1: "view", 2: "copy"} {
		for name, mutate := range map[string]func(*contractstatemir.Operation){
			"missing-input": func(op *contractstatemir.Operation) { op.BorrowInput = 0 },
			"future-input":  func(op *contractstatemir.Operation) { op.BorrowInput = 77 },
			"field":         func(op *contractstatemir.Operation) { op.FieldIndex = 1 },
			"key":           func(op *contractstatemir.Operation) { op.KeyInput = 1 },
			"found":         func(op *contractstatemir.Operation) { op.Found = 5 },
			"write":         func(op *contractstatemir.Operation) { op.ValueInput = 2 },
			"call":          func(op *contractstatemir.Operation) { op.Callee = "main::unused" },
			"cost":          func(op *contractstatemir.Operation) { op.Cost = 0 },
		} {
			add(label+"-"+name, false, false, func(p *contractstatemir.Program) { mutate(&p.Blocks[0].Operations[index]) })
		}
	}
	add("view-empty", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[1].ViewLength = 0 })
	add("view-missing-result", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[1].BorrowResult = 0 })
	add("view-duplicate-borrow-id", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[1].BorrowResult = 1 })
	add("view-ordinary-result", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[1].Result = 8 })
	add("copy-missing-result", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[2].Result = 0 })
	add("copy-duplicate-ordinary-id", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[2].Result = 1 })
	add("copy-borrow-result", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[2].BorrowResult = 3 })
	add("copy-offset", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[2].ViewOffset = 1 })
	add("copy-length", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[2].ViewLength = 1 })
	add("copy-used-with-wrong-codec", false, false, func(p *contractstatemir.Program) { p.Blocks[0].Operations[3].FieldIndex = 1 })
	add("bare-borrow-is-not-a-value", false, false, func(p *contractstatemir.Program) {
		p.Blocks[0].Operations = append(p.Blocks[0].Operations[:2], p.Blocks[0].Operations[3])
	})

	for _, mapped := range []bool{false, true} {
		for _, write := range []bool{false, true} {
			for _, same := range []bool{false, true} {
				add(fmt.Sprintf("invalidate-map-%v-write-%v-same-%v", mapped, write, same), mapped, !same, func(p *contractstatemir.Program) {
					op := contractstatemir.Operation{Kind: contractstatemir.OperationCellDelete, Capability: p.Capability, FieldIndex: 5, Cost: 1}
					if !same {
						op.FieldIndex = 6
					}
					if write {
						op.Kind = contractstatemir.OperationCellWrite
						op.Value = contractstatemir.Value{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte("next")}
						if !same {
							op.Value.TypeID = contractstate.Text.TypeID()
						}
					}
					if mapped {
						op.Kind, op.FieldIndex = contractstatemir.OperationMapDelete, 8
						op.Key = contractstatemir.Value{TypeID: contractstate.Text.TypeID(), Encoded: []byte("alpha")}
						if !same {
							op.Key.Encoded = []byte("beta")
						}
						if write {
							op.Kind = contractstatemir.OperationMapWrite
							op.Value = contractstatemir.Value{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte("next")}
						}
					}
					stateGraphBorrowInsert(p, 2, op)
				})
			}
		}
	}
	add("root-invalidated-before-view", false, false, func(p *contractstatemir.Program) {
		stateGraphBorrowInsert(p, 1, contractstatemir.Operation{Kind: contractstatemir.OperationCellDelete, Capability: p.Capability, FieldIndex: 5, Cost: 1})
	})
	add("copy-detached-after-exact-write", false, true, func(p *contractstatemir.Program) {
		stateGraphBorrowInsert(p, 3, contractstatemir.Operation{Kind: contractstatemir.OperationCellDelete, Capability: p.Capability, FieldIndex: 5, Cost: 1})
	})
	add("fresh-borrow-after-invalidation", false, true, func(p *contractstatemir.Program) {
		stateGraphBorrowInsert(p, 2,
			contractstatemir.Operation{Kind: contractstatemir.OperationCellDelete, Capability: p.Capability, FieldIndex: 5, Cost: 1},
			contractstatemir.Operation{Kind: contractstatemir.OperationCellBorrow, Capability: p.Capability, FieldIndex: 5, BorrowResult: 3, Found: 3, Cost: 1})
		p.Blocks[0].Operations[4].BorrowInput = 3
	})
	for _, live := range []bool{false, true} {
		for _, write := range []bool{false, true} {
			add(fmt.Sprintf("dynamic-map-live-%v-write-%v", live, write), true, !live, func(p *contractstatemir.Program) {
				p.Blocks[0].Operations = p.Blocks[0].Operations[:2]
				if !live {
					p.Blocks[0].Operations = append(p.Blocks[0].Operations, contractstatemir.Operation{Kind: contractstatemir.OperationMapDelete, Capability: p.Capability, FieldIndex: 8, Key: p.Blocks[0].Operations[0].Key, Cost: 1})
				}
				p.Blocks[0].Operations = append(p.Blocks[0].Operations,
					contractstatemir.Operation{Kind: contractstatemir.OperationCellRead, Capability: p.Capability, FieldIndex: 6, Result: 3, Found: 4, Cost: 1},
					contractstatemir.Operation{Kind: contractstatemir.OperationMapDelete, Capability: p.Capability, FieldIndex: 8, KeyInput: 3, Cost: 1})
				if write {
					op := &p.Blocks[0].Operations[len(p.Blocks[0].Operations)-1]
					op.Kind = contractstatemir.OperationMapWrite
					op.Value = contractstatemir.Value{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte("next")}
				}
			})
		}
	}
	add("dynamic-map-other-field-preserves", true, true, func(p *contractstatemir.Program) {
		stateGraphBorrowInsert(p, 2,
			contractstatemir.Operation{Kind: contractstatemir.OperationCellRead, Capability: p.Capability, FieldIndex: 1, Result: 3, Found: 4, Cost: 1},
			contractstatemir.Operation{Kind: contractstatemir.OperationMapDelete, Capability: p.Capability, FieldIndex: 7, KeyInput: 3, Cost: 1})
	})
	for _, escape := range []bool{false, true} {
		add(fmt.Sprintf("edge-bare-borrow-%v", escape), false, !escape, func(p *contractstatemir.Program) {
			id := contractstatemir.ValueID(2)
			if escape {
				p.Blocks[0].Operations = p.Blocks[0].Operations[:2]
			}
			p.Blocks[0].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorJump, Target: 1, Arguments: []contractstatemir.ValueID{id}, Cost: 1}
			p.Blocks = append(p.Blocks, contractstatemir.Block{ID: 1, Parameters: []contractstatemir.BlockParameter{{ID: 8, TypeID: contractstate.Bytes.TypeID()}}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}})
		})
	}
	add("block-local-borrow-unavailable", false, false, func(p *contractstatemir.Program) {
		copy := p.Blocks[0].Operations[2]
		p.Blocks[0].Operations = p.Blocks[0].Operations[:2]
		p.Blocks[0].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorJump, Target: 1, Cost: 1}
		p.Blocks = append(p.Blocks, contractstatemir.Block{ID: 1, Operations: []contractstatemir.Operation{copy}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}})
	})
	add("borrow-id-global-across-blocks", false, false, func(p *contractstatemir.Program) {
		op := p.Blocks[0].Operations[0]
		op.Found = 8
		p.Blocks[0].Terminator = contractstatemir.Terminator{Kind: contractstatemir.TerminatorJump, Target: 1, Cost: 1}
		p.Blocks = append(p.Blocks, contractstatemir.Block{ID: 1, Operations: []contractstatemir.Operation{op}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}})
	})
	for _, writes := range []bool{false, true} {
		for _, transitive := range []bool{false, true} {
			add(fmt.Sprintf("helper-write-%v-transitive-%v", writes, transitive), false, !writes, func(p *contractstatemir.Program) {
				effect, op := hir.EffectStorageRead, contractstatemir.Operation{Kind: contractstatemir.OperationCellRead, Capability: p.Capability, FieldIndex: 1, Result: 10, Found: 11, Cost: 1}
				if writes {
					effect = hir.EffectStorageWrite
					op = contractstatemir.Operation{Kind: contractstatemir.OperationCellDelete, Capability: p.Capability, FieldIndex: 1, Cost: 1}
				}
				p.Helpers = []contractstatemir.Helper{{Name: "main::a", Effects: []hir.Effect{effect}, Blocks: []contractstatemir.Block{{Operations: []contractstatemir.Operation{op}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorReturn, Cost: 1}}}}}
				if transitive {
					p.Helpers[0].Name = "main::z"
					p.Helpers = append([]contractstatemir.Helper{{Name: "main::a", Effects: []hir.Effect{effect}, Blocks: []contractstatemir.Block{{Operations: []contractstatemir.Operation{{Kind: contractstatemir.OperationCall, Capability: p.Capability, Callee: "main::z", Cost: 1}}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorReturn, Cost: 1}}}}}, p.Helpers...)
				}
				stateGraphBorrowInsert(p, 2, contractstatemir.Operation{Kind: contractstatemir.OperationCall, Capability: p.Capability, Callee: "main::a", Cost: 1})
			})
		}
	}
	p := stateGraphBorrowBase(t, false)
	helper := contractstatemir.Helper{Name: "main::borrow", Effects: p.Effects, Blocks: p.Blocks}
	helper.Blocks[0].Terminator.Kind = contractstatemir.TerminatorReturn
	p.Helpers = []contractstatemir.Helper{helper}
	p.Blocks = []contractstatemir.Block{{Operations: []contractstatemir.Operation{{Kind: contractstatemir.OperationCall, Capability: p.Capability, Callee: helper.Name, Cost: 1}}, Terminator: contractstatemir.Terminator{Kind: contractstatemir.TerminatorCommit, Cost: 1}}}
	trials = append(trials, stateGraphBorrowTrial{name: "helper-cannot-own-borrow", program: p, want: false, body: 1})
	return trials
}

func TestCandidateStateGraphBorrowIROracle(t *testing.T) {
	for _, trial := range stateGraphBorrowTrials(t) {
		t.Run(trial.name, func(t *testing.T) {
			failures := contractstatemir.Verify(trial.program)
			if got := len(failures) == 0; got != trial.want {
				t.Fatalf("Go accepted=%v want=%v: %v", got, trial.want, failures)
			}
		})
	}
}

// The context envelope is private test transport. Helpers' transitive effects
// are supplied only to the context validator, never to a sealed proof factory.
func stateGraphBorrowPacket(trial stateGraphBorrowTrial) []byte {
	p := trial.program
	var out bytes.Buffer
	out.WriteString("FYSGBCTX")
	number := func(n uint64) { _ = binary.Write(&out, binary.BigEndian, n) }
	mode := uint64(0)
	if len(p.Helpers) != 0 {
		mode = 1
	}
	if trial.body != 0 {
		mode = 2
	}
	number(mode)
	number(uint64(len(p.Helpers)))
	for _, helper := range p.Helpers {
		var effects uint64
		for _, effect := range helper.Effects {
			if effect == hir.EffectStorageRead {
				effects |= 1
			}
			if effect == hir.EffectStorageWrite {
				effects |= 2
			}
		}
		number(effects)
	}
	if trial.body != 0 {
		helper := p.Helpers[trial.body-1]
		p.Blocks, p.Entry, p.Effects = helper.Blocks, helper.Entry, helper.Effects
	}
	packet := stateGraphIRPacket(p)
	// The shared helper maps only registered names to numeric IDs. Preserve a
	// nonempty unknown name as an out-of-range sentinel in these negative cases.
	var parameters, arguments int
	for _, block := range p.Blocks {
		parameters += len(block.Parameters)
		arguments += len(block.Terminator.Arguments) + len(block.Terminator.TrueArguments) + len(block.Terminator.FalseArguments)
	}
	operation := 80 + parameters*16 + arguments*8
	for _, block := range p.Blocks {
		for _, op := range block.Operations {
			if op.Callee != "" && binary.BigEndian.Uint64(packet[operation+14*8:]) == 0 {
				binary.BigEndian.PutUint64(packet[operation+14*8:], uint64(len(p.Helpers)+1))
			}
			operation += 19 * 8
		}
	}
	out.Write(packet)
	return out.Bytes()
}

func stateGraphBorrowProbe(t *testing.T) string {
	t.Helper()
	probe := stateGraphIRProbe
	replace := func(before, after string) {
		if strings.Count(probe, before) != 1 {
			t.Fatalf("borrow probe seam not unique: %s", before)
		}
		probe = strings.Replace(probe, before, after, 1)
	}
	replace(`let g=switch(read(bytes::view(ref input),ref mut work))`, `let raw=bytes::view(ref input);var ctx=switch(wire::cursor(raw,b"FYSGBCTX",ref mut work)){case .ok(c):{c}case .err(_):{return 40;}};
 let mode=number(raw,ref mut ctx,ref mut work);let count=wire::count_from(raw,ref mut ctx,33,8,ref mut work);var v::Vec<u32> helper_effects=v::empty<u32>();var usize h=0;
 while(h<count){let effect=wire::word_from(raw,ref mut ctx,ref mut work);if(!v::push<u32>(ref mut helper_effects,effect).is_ok){return 41;}h+=1;}
 if(wire::status(ref ctx)!=0 || mode>2){return 42;}let g=switch(read(raw[ctx.offset..],ref mut work))`)
	probe = strings.ReplaceAll(probe, `verify::validate(ref g,ref p,ref mut work)`, `verify::validate_function(ref g,ref p,ref helper_effects,mode!=0,mode==2,ref mut work)`)
	replace(`work=0;let proof=switch(verify::verify`, `if(mode!=0){discard io::write("0");return 0;}work=0;let proof=switch(verify::verify`)
	replace(`b"`+stateGraphIRSource+`"`, `b"`+strings.ReplaceAll(stateGraphIRSource, `"`, `\"`)+`"`)
	return probe
}

func TestCandidateStateGraphBorrowIR(t *testing.T) {
	trials := stateGraphBorrowTrials(t)
	// Fail fast on an incorrectly authored oracle before building native code.
	for _, trial := range trials {
		if got := len(contractstatemir.Verify(trial.program)) == 0; got != trial.want {
			t.Fatalf("bad Go oracle %s", trial.name)
		}
	}
	candidateAtlasProbeSize(t, "module_state_graph_verify", stateGraphBorrowProbe(t), 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				code, output := candidateAtlasProcess(t, image, stateGraphBorrowPacket(trial))
				if code != 0 || output != "0" && output != "1001" || (output == "0") != trial.want {
					t.Fatalf("Go accepted=%v, Fairy exit=%d output=%q; Go failures=%v", trial.want, code, output, contractstatemir.Verify(trial.program))
				}
			})
		}
		// usize private transport must not widen the Go u32 view fields. These
		// cannot be represented in Operation and therefore have no Go fixture.
		for _, field := range []int{17, 18} {
			t.Run(fmt.Sprintf("view-wire-u32-overflow-%d", field), func(t *testing.T) {
				packet := stateGraphBorrowPacket(trials[0])
				// Context 24 + graph header 80 + first operation 19*8.
				binary.BigEndian.PutUint64(packet[24+80+19*8+field*8:], 1<<32)
				if code, output := candidateAtlasProcess(t, image, packet); code != 0 || output != "1001" {
					t.Fatalf("wide view field: %d %q", code, output)
				}
			})
		}
	}, "module_state_declarations", "candidate_artifact_wire")
}
