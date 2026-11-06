package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type stateGraphAuthoredTrial struct {
	name, text, code string
}

func stateGraphAuthoredSource(effects, body, helpers string) string {
	attribute := ""
	if effects != "" {
		attribute = "@effects(" + effects + ")"
	}
	return "module ledger;" + stateGraphSourceSchema + attribute + "fn void transaction(){" + body + "}" + helpers
}

func stateGraphAuthoredTrials() []stateGraphAuthoredTrial {
	var out []stateGraphAuthoredTrial
	add := func(name, effects, body, helpers, code string) {
		out = append(out, stateGraphAuthoredTrial{name, stateGraphAuthoredSource(effects, body, helpers), code})
	}
	rw := "storage_read,storage_write"
	add("read-local", rw, `let x=storage::read("count");storage::write("count",x);storage::commit();`, "", "")
	add("typed-alias-found", rw, `let u32 x=storage::read("count");let y=((x));let bool exists=storage::found(y);storage::write("flag",exists);storage::write("values",y,x);storage::commit();`, "", "")
	for _, field := range []string{"flag", "count", "total", "account", "hash", "blob", "title"} {
		add("copy-codec-"+field, rw, fmt.Sprintf(`let x=storage::read(%q);storage::write(%q,x);storage::commit();`, field, field), "", "")
		add("borrow-codec-"+field, rw, fmt.Sprintf(`let b=storage::borrow(%q);let x=storage::copy(b);storage::write(%q,x);storage::commit();`, field, field), "", "")
	}
	add("map-local-key-value", rw, `let key=storage::read("count");let old=storage::read("values",key);storage::write("values",key,old);storage::delete("values",key);storage::commit();`, "", "")
	add("borrow-alias-view-copy", rw, `let raw=storage::borrow("blob");let alias=(raw);let view=storage::view(alias,0,1);let bytes copied=storage::copy(view);let bool exists=storage::found(view);storage::write("blob",copied);storage::write("flag",exists);storage::commit();`, "", "")
	add("map-other-key-preserves", rw, `let b=storage::borrow("values",1);storage::write("values",2,3);let x=storage::copy(b);storage::write("count",x);storage::commit();`, "", "")
	add("copy-survives-mutation", rw, `let b=storage::borrow("count");let x=storage::copy(b);storage::delete("count");storage::write("count",x);storage::commit();`, "", "")
	add("view-u32-max", "storage_read", `let b=storage::borrow("blob");let v=storage::view(b,4294967295,4294967295);let x=storage::copy(v);storage::commit();`, "", "")
	for _, body := range []struct{ name, text string }{
		{"branch-tail", `let flag=storage::read("flag");if(flag){storage::commit();}else{storage::reject();}`},
		{"branch-join", `let flag=storage::read("flag");let x=storage::read("count");if(flag){storage::write("count",x);}else{storage::delete("count");}storage::write("values",x,x);storage::commit();`},
		{"branch-no-else", `let flag=storage::read("flag");if(storage::found(flag)){storage::delete("count");}storage::commit();`},
		{"branch-one-closed", `let x=storage::read("flag");if((x)){storage::reject();}else{storage::write("flag",x);}storage::commit();`},
		{"branch-sibling-locals", `let flag=storage::read("flag");if(flag){let x=storage::read("count");storage::write("count",x);}else{let x=storage::read("count");storage::write("count",x);}storage::commit();`},
		{"nested-joins", `let flag=storage::read("flag");let x=storage::read("count");if(flag){if(storage::found(x)){storage::write("count",x);}}else{storage::delete("count");}storage::write("values",x,x);storage::commit();`},
		{"borrow-found-crosses-edge", `let b=storage::borrow("blob");if(storage::found(b)){storage::reject();}storage::commit();`},
		{"copied-value-crosses-edge", `let b=storage::borrow("count");let x=storage::copy(b);if(storage::found(b)){storage::write("count",x);}storage::write("count",x);storage::commit();`},
	} {
		add(body.name, rw, body.text, "", "")
	}
	write := `@effects(storage_write)fn void write(){storage::write("count",7);return;}`
	read := `@effects(storage_read)fn void read(){let x=storage::read("count");return;}`
	add("helper-write", "storage_write", `write();storage::commit();`, write, "")
	add("helper-read-preserves", "storage_read", `let b=storage::borrow("count");read();let x=storage::copy(b);storage::commit();`, read, "")
	add("helper-copy-survives-write", rw, `let b=storage::borrow("count");let x=storage::copy(b);write();storage::write("count",x);storage::commit();`, write, "")
	add("helper-transitive", "storage_write", `outer();storage::commit();`, `@effects(storage_write)fn void outer(){write();return;}`+write, "")
	out = append(out, stateGraphAuthoredTrial{"helper-before-root", "module ledger;" + stateGraphSourceSchema + write + `@effects(storage_write)fn void transaction(){write();storage::commit();}`, ""})
	add("helper-branch", rw, `route();storage::commit();`, `@effects(storage_read,storage_write)fn void route(){let flag=storage::read("flag");if(flag){write();return;}else{storage::reject();}}`+write, "")
	for _, body := range []struct{ name, text string }{
		{"mutable-local", `var x=storage::read("count");storage::commit();`},
		{"wrong-annotation", `let bool x=storage::read("count");storage::commit();`},
		{"duplicate-local", `let x=storage::read("count");let x=storage::read("count");storage::commit();`},
		{"unknown-local", `storage::write("count",x);storage::commit();`},
		{"wrong-local-type", `let x=storage::read("flag");storage::write("count",x);storage::commit();`},
		{"found-of-found", `let x=storage::read("count");let y=storage::found(x);let z=storage::found(y);storage::commit();`},
		{"borrow-as-value", `let b=storage::borrow("count");storage::write("count",b);storage::commit();`},
		{"borrow-annotation", `let u32 b=storage::borrow("count");storage::commit();`},
		{"borrow-dynamic-key", `let key=storage::read("count");let b=storage::borrow("values",key);storage::commit();`},
		{"view-empty", `let b=storage::borrow("blob");let v=storage::view(b,0,0);storage::commit();`},
		{"view-overflow", `let b=storage::borrow("blob");let v=storage::view(b,4294967296,1);storage::commit();`},
		{"borrow-write-invalidates", `let b=storage::borrow("count");storage::delete("count");let x=storage::copy(b);storage::commit();`},
		{"map-same-key-invalidates", `let b=storage::borrow("values",1);storage::delete("values",1);let x=storage::copy(b);storage::commit();`},
		{"map-dynamic-ambiguous", `let key=storage::read("count");let b=storage::borrow("values",1);storage::delete("values",key);storage::commit();`},
		{"branch-borrow-escape", `let b=storage::borrow("count");if(storage::found(b)){let x=storage::copy(b);storage::reject();}storage::commit();`},
		{"branch-local-escape", `let flag=storage::read("flag");if(flag){let x=storage::read("count");}storage::write("count",x);storage::commit();`},
		{"branch-local-shadow", `let x=storage::read("count");if(storage::found(x)){let x=storage::read("count");}storage::commit();`},
		{"branch-condition-type", `let x=storage::read("count");if(x){storage::reject();}storage::commit();`},
		{"after-closed-branches", `let x=storage::read("flag");if(x){storage::commit();}else{storage::reject();}storage::delete("count");`},
	} {
		add(body.name, rw, body.text, "", "1001")
	}
	add("helper-write-invalidates", rw, `let b=storage::borrow("count");write();let x=storage::copy(b);storage::commit();`, write, "1001")
	add("local-shadows-helper", rw, `let write=storage::read("count");write();storage::commit();`, write, "1001")
	add("helper-borrow", "storage_read", `read();storage::commit();`, strings.Replace(read, `let x=storage::read("count");`, `let x=storage::borrow("count");`, 1), "1001")
	add("helper-commit", "storage_write", `write();storage::commit();`, strings.Replace(write, "return;", "storage::commit();", 1), "1001")
	add("helper-recursion", "storage_write", `write();storage::commit();`, strings.Replace(write, `storage::write("count",7);`, "write();", 1), "1001")
	add("unused-helper", "storage_write", `storage::write("count",1);storage::commit();`, write, "1001")
	add("root-callee", "", `transaction();storage::commit();`, "", "1001")
	add("helper-effect-drift", rw, `let x=storage::read("flag");write();storage::commit();`, strings.Replace(write, "storage_write)", "storage_read,storage_write)", 1), "1001")
	for _, n := range []int{31, 32, 33} {
		var helpers strings.Builder
		for i := n - 1; i >= 0; i-- {
			fmt.Fprintf(&helpers, "@effects(storage_write)fn void h%02d(){", i)
			if i+1 == n {
				helpers.WriteString(`storage::delete("count");`)
			} else {
				fmt.Fprintf(&helpers, "h%02d();", i+1)
			}
			helpers.WriteString("return;}")
		}
		code := ""
		if n > 32 {
			code = "1002"
		}
		add(fmt.Sprintf("helper-count-%d", n), "storage_write", "h00();storage::commit();", helpers.String(), code)
	}
	for _, n := range []int{20, 21, 22} {
		code := ""
		if n > 21 {
			code = "1002"
		}
		add(fmt.Sprintf("block-bound-%d-branches", n), rw, `let flag=storage::read("flag");`+strings.Repeat(`if(flag){storage::delete("count");}`, n)+`storage::commit();`, "", code)
	}
	return out
}

func stateGraphAuthoredOracle(t *testing.T, trial stateGraphAuthoredTrial) (stateBodyFixture, string) {
	t.Helper()
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("ledger.fy", []byte(trial.text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
		t.Fatalf("invalid oracle fixture syntax: %v %+v", err, parsed.Tree.Diagnostics)
	}
	result, err := compiler.CheckExperimentalStateGraph(semantic.File{Package: 7, Path: "ledger.fy", Module: parsed.AST})
	valid := err == nil && result.Verified
	if valid != (trial.code == "") {
		t.Fatalf("Go admission=%v want=%v err=%v diagnostics=%+v unsupported=%+v", valid, trial.code == "", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
	}
	if !valid {
		return stateBodyFixture{}, trial.code
	}
	coordinate := packagegraph.CoordinateID(sha256.Sum256([]byte("authored state graph package coordinate")))
	return stateGraphBodyFixture(t, trial.name, result.Semantic.HIR, coordinate), ""
}

const stateGraphAuthoredProbe = `module main;
import module_state_graph_source as source;import module_state_graph_source_artifact as bound;import module_state_graph_body_hash as hash;import module_state_graph_hash_context as context;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::io as io;
@effects(console)fn i32 main(){var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] v=ref chunk;if(bytes::len(ref input)>1048576-n || !bytes::append(ref mut input,v[..n]).is_ok){return 2;}}
 let raw=bytes::view(ref input);if(raw.len<48){return 3;}var usize setup=0;var c=wire::Cursor{.offset=32,.code=0};let length=wire::count_from(raw,ref mut c,1048576,1,ref mut setup);let size=wire::count_from(raw,ref mut c,65536,1,ref mut setup);if(c.code!=0 || length>raw.len-c.offset || size!=raw.len-c.offset-length){return 4;}
 let descriptor=raw[c.offset..c.offset+length];let actual=raw[c.offset+length..];var usize work=0;let produced=switch(source::produce(actual,ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};let cost=work;
 work=16777216-cost;switch(source::produce(actual,ref mut work)){case .ok(_):{}case .err(_):{return 5;}};if(work!=16777216){return 6;}
 work=16777217-cost;switch(source::produce(actual,ref mut work)){case .ok(_):{return 7;}case .err(code):{if(code!=1002 || work>16777216){return 8;}}};
 work=16777217;switch(source::produce(actual,ref mut work)){case .ok(_):{return 9;}case .err(code):{if(code!=1002 || work!=16777217){return 10;}}};
 work=0;let identity=switch(context::validate(raw[..32],descriptor,ref mut work)){case .ok(p):{p}case .err(_):{return 11;}};
 let original=switch(hash::derive(source::program(ref produced),ref identity,ref mut work)){case .ok(p):{p}case .err(_):{return 12;}};
 work=0;let saved=switch(bound::encode(ref produced,ref mut work)){case .ok(p):{p}case .err(_):{return 13;}};work=0;
 let restored=switch(bound::restore(wire::content(ref saved),actual,ref mut work)){case .ok(p):{p}case .err(_):{return 14;}};work=0;
 let replay=switch(hash::derive(source::program(bound::current(ref restored)),ref identity,ref mut work)){case .ok(p):{p}case .err(_):{return 15;}};let decoded=switch(hash::derive(bound::graph(ref restored),ref identity,ref mut work)){case .ok(p):{p}case .err(_):{return 16;}};
 if(!bytes::equal(hash::canonical(ref original),hash::canonical(ref replay)) || !bytes::equal(hash::canonical(ref original),hash::canonical(ref decoded))){return 17;}
 return switch(io::write_bytes(hash::canonical(ref original))){case .ok(n):{if(n==hash::canonical(ref original).len){0}else{18}}case .err(_):{19}};
}`

func TestCandidateStateGraphAuthoredValuesOracle(t *testing.T) {
	for _, trial := range stateGraphAuthoredTrials() {
		t.Run(trial.name, func(t *testing.T) { stateGraphAuthoredOracle(t, trial) })
	}
}

func TestCandidateStateGraphAuthoredValues(t *testing.T) {
	trials := stateGraphAuthoredTrials()
	var packets, wants [][]byte
	for _, trial := range trials {
		fixture, code := stateGraphAuthoredOracle(t, trial)
		var b bytes.Buffer
		b.Write(fixture.coordinate[:])
		_ = binary.Write(&b, binary.BigEndian, uint64(len(fixture.descriptor)))
		_ = binary.Write(&b, binary.BigEndian, uint64(len(trial.text)))
		b.Write(fixture.descriptor)
		b.WriteString(trial.text)
		packets = append(packets, b.Bytes())
		want := fixture.canonical
		if code != "" {
			want = []byte(code)
		}
		wants = append(wants, want)
	}
	candidateAtlasProbeSize(t, "module_state_graph_source", stateGraphAuthoredProbe, 512<<10, func(t *testing.T, image string) {
		for i, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				code, output := candidateAtlasProcess(t, image, packets[i])
				if code != 0 || !bytes.Equal([]byte(output), wants[i]) {
					at := 0
					for at < len(output) && at < len(wants[i]) && output[at] == wants[i][at] {
						at++
					}
					t.Fatalf("native exit=%d lengths=%d/%d first mismatch=%d output-prefix=%x", code, len(output), len(wants[i]), at, []byte(output[:min(len(output), 64)]))
				}
			})
		}
	}, "module_state_graph_source_artifact", "module_state_graph_body_hash", "module_state_graph_hash_context")
}
