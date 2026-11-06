package cli

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// Count graph operations, not statements: reads, deletes and helper calls each
// count once; aliases and commit/return terminators count zero. Neither helper
// approaches 1024 on its own, so a per-function limit cannot catch the overflow.
func stateGraphSourceLimitTrial(operations int, rootFirst, aliases bool) stateGraphAuthoredTrial {
	const rootOperations, alphaOperations = 19, 501
	aliasChain := func(prefix, seed string) string {
		if !aliases {
			return ""
		}
		var out strings.Builder
		for i := 0; i < 32; i++ {
			name := fmt.Sprintf("%s%02d", prefix, i)
			fmt.Fprintf(&out, "let %s=(%s);", name, seed)
			seed = name
		}
		return out.String()
	}
	root := `@effects(storage_read,storage_write)fn void transaction(){let seed=storage::read("count");alpha();zeta();` +
		strings.Repeat(`storage::delete("count");`, rootOperations-3) + aliasChain("r", "seed") + `storage::commit();}`
	alpha := `@effects(storage_write)fn void alpha(){` + strings.Repeat(`storage::delete("count");`, alphaOperations) + `return;}`
	zeta := `@effects(storage_read,storage_write)fn void zeta(){` +
		strings.Repeat(`storage::delete("count");`, operations-rootOperations-alphaOperations-1) +
		`let seed=storage::read("count");` + aliasChain("z", "seed") + `return;}`
	// Reverse helper declaration order relative to their canonical names. The
	// last canonical helper exhausts the remaining allowance before its aliases.
	body, order := zeta+alpha+root, "root-after"
	if rootFirst {
		body, order = root+zeta+alpha, "root-before"
	}
	name := fmt.Sprintf("combined-%d/%s", operations, order)
	if aliases {
		name += "/zero-operation-aliases"
	}
	code := ""
	if operations > 1024 {
		code = "1002"
	}
	return stateGraphAuthoredTrial{name, "module ledger;" + stateGraphSourceSchema + body, code}
}

func TestCandidateStateGraphSourceLimits(t *testing.T) {
	type limitCase struct {
		trial      stateGraphAuthoredTrial
		operations int
		packet     []byte
		want       []byte
	}
	var cases []limitCase
	for _, rootFirst := range []bool{true, false} {
		for _, operations := range []int{1023, 1024, 1025} {
			cases = append(cases, limitCase{trial: stateGraphSourceLimitTrial(operations, rootFirst, false), operations: operations})
		}
		withAliases := stateGraphSourceLimitTrial(1024, rootFirst, true)
		cases = append(cases, limitCase{trial: withAliases, operations: 1024})
		// The same full graph with an unknown alias is a semantic rejection,
		// not a policy overflow. Do not let a statement-count gate hide it.
		badAlias := withAliases
		badAlias.name += "/unknown-alias"
		badAlias.text = strings.Replace(badAlias.text, "let z31=(z30);", "let z31=(missing);", 1)
		badAlias.code = "1001"
		cases = append(cases, limitCase{trial: badAlias, operations: 1024})
	}

	for i := range cases {
		c := &cases[i]
		t.Run("oracle/"+c.trial.name, func(t *testing.T) {
			if len(c.trial.text) >= 65536 {
				t.Fatal("operation fixture must stay below the independent source byte limit")
			}
			var fixture stateBodyFixture
			if c.trial.code == "1002" {
				// Go's source producer itself refuses 1025 operations. Check
				// its actual limit error rather than pretending it supplied a
				// canonical body or mapping every Go refusal to policy 1002.
				parsed, err := frontend.AnalyzeSyntax(source.NewFile("ledger.fy", []byte(c.trial.text)), source.DefaultLimits())
				if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
					t.Fatalf("policy fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
				}
				result, err := compiler.CheckExperimentalStateGraph(semantic.File{Package: 7, Path: "ledger.fy", Module: parsed.AST})
				if result.Verified || err == nil || !strings.Contains(err.Error(), "exceeds 1024 operations") || len(result.Semantic.Diagnostics)+len(result.Semantic.Unsupported) != 0 {
					t.Fatalf("expected Go operation-bound refusal, verified=%v err=%v semantic=%+v unsupported=%+v", result.Verified, err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
				}
				c.want = []byte("1002")
			} else {
				var code string
				fixture, code = stateGraphAuthoredOracle(t, c.trial)
				c.want = fixture.canonical
				if code != "" {
					c.want = []byte(code)
				} else {
					operations := 0
					for _, transaction := range fixture.input.StateTransactions {
						operations += len(transaction.Accesses)
						for _, block := range transaction.Blocks {
							operations += len(block.Accesses)
							if len(block.Accesses) >= 1024 {
								t.Fatal("fixture must require aggregate rather than per-function accounting")
							}
						}
					}
					if len(fixture.input.StateTransactions) != 3 || operations != c.operations {
						t.Fatalf("Go graph has %d functions/%d operations, want 3/%d", len(fixture.input.StateTransactions), operations, c.operations)
					}
				}
			}
			var packet bytes.Buffer
			packet.Write(fixture.coordinate[:])
			_ = binary.Write(&packet, binary.BigEndian, uint64(len(fixture.descriptor)))
			_ = binary.Write(&packet, binary.BigEndian, uint64(len(c.trial.text)))
			packet.Write(fixture.descriptor)
			packet.WriteString(c.trial.text)
			c.packet = packet.Bytes()
		})
	}
	if t.Failed() {
		return
	}
	candidateAtlasProbeSize(t, "module_state_graph_source", stateGraphAuthoredProbe, 512<<10, func(t *testing.T, image string) {
		for _, c := range cases {
			t.Run(c.trial.name, func(t *testing.T) {
				code, output := candidateAtlasProcess(t, image, c.packet)
				if code != 0 || !bytes.Equal([]byte(output), c.want) {
					t.Fatalf("native exit=%d output=%d bytes want=%d bytes output-prefix=%x want-prefix=%x", code, len(output), len(c.want), []byte(output[:min(len(output), 64)]), c.want[:min(len(c.want), 64)])
				}
			})
		}
	}, "module_state_graph_source_artifact", "module_state_graph_body_hash", "module_state_graph_hash_context")
}
