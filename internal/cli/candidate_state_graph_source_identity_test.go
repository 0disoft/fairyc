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
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// This transport carries coordinates and actual source, never a descriptor or
// expected hash. All expected identity bytes come from Go source compilation.
type stateSourceIdentityRow struct{ owner, start, end uint64 }

func stateSourceIdentityPacket(owner uint64, rows []stateSourceIdentityRow, names, source string) []byte {
	var out bytes.Buffer
	n := func(v uint64) { _ = binary.Write(&out, binary.BigEndian, v) }
	n(owner)
	n(uint64(len(rows)))
	for _, r := range rows {
		n(r.owner)
		n(r.start)
		n(r.end)
	}
	n(uint64(len(names)))
	out.WriteString(names)
	out.WriteString(source)
	return out.Bytes()
}

func stateSourceIdentityWant(f stateBodyFixture) []byte {
	out := append([]byte(nil), f.coordinate[:]...)
	out = append(out, f.descriptor...)
	out = append(out, []byte("\n"+f.hash+"\n")...)
	return append(out, f.canonical...)
}

const stateSourceIdentityProbe = `module main;
import module_state_graph_source_identity as bound;import module_state_graph_source as source;import module_state_graph_hash_context as context;import module_state_graph_body_hash as hash;import module_public_type_text as coordinates;import candidate_artifact_wire as wire;import std::bytes as bytes;import std::vec as v;import std::io as io;
fn Result<bound::Bound,u32> bind(ref u8[] raw){
 var usize setup=0;var wire::Cursor c=wire::Cursor{.offset=0,.code=0};let owner=wire::number_from(raw,ref mut c,ref mut setup);let count=wire::number_from(raw,ref mut c,ref mut setup);if(c.code!=0 || count>513){return .err(9001);}
 var v::Vec<coordinates::Coordinate> rows=v::empty<coordinates::Coordinate>();var usize i=0;while(i<count){let owner=wire::number_from(raw,ref mut c,ref mut setup);let start=wire::number_from(raw,ref mut c,ref mut setup);let end=wire::number_from(raw,ref mut c,ref mut setup);if(!v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner,.start,.end}).is_ok){return .err(9002);}i+=1;}
 let size=wire::number_from(raw,ref mut c,ref mut setup);if(c.code!=0 || size>raw.len-c.offset){return .err(9003);}let names=switch(bytes::copy(raw[c.offset..c.offset+size])){case .ok(p):{p}case .err(_):{return .err(9004);}};c.offset+=size;
 var table=coordinates::assemble(names,rows);var authored=switch(bytes::copy(raw[c.offset..])){case .ok(p):{p}case .err(_):{return .err(9005);}};var usize work=0;
 let produced=switch(source::produce(bytes::view(ref authored),ref mut work)){case .ok(p):{p}case .err(code):{return .err(code);}};
 // Mutating the caller's original source must not rename the retained Parsed.
 {let raw=bytes::view_mut(ref mut authored);if(raw.len!=0){raw[0]=99;}}
 work=0;let result=switch(bound::derive(ref produced,ref table,owner,ref mut work)){case .ok(p):{p}case .err(code):{return .err(code);}};let cost=work;if(cost==0 || cost>16777216){return .err(9006);}
 work=16777216-cost;let exact=switch(bound::derive(ref produced,ref table,owner,ref mut work)){case .ok(p):{p}case .err(_):{return .err(9007);}};
 if(work!=16777216 || !bytes::equal(context::coordinate(bound::context(ref result)),context::coordinate(bound::context(ref exact))) || !bytes::equal(context::descriptor(bound::context(ref result)),context::descriptor(bound::context(ref exact))) || !bytes::equal(hash::canonical(bound::identity(ref result)),hash::canonical(bound::identity(ref exact))) || !bytes::equal(hash::digest(bound::identity(ref result)),hash::digest(bound::identity(ref exact)))){return .err(9008);}
 work=16777217-cost;switch(bound::derive(ref produced,ref table,owner,ref mut work)){case .ok(_):{return .err(9009);}case .err(code):{if(code!=1002 || work>16777216){return .err(9010);}}};
 work=16777217;switch(bound::derive(ref produced,ref table,owner,ref mut work)){case .ok(_):{return .err(9011);}case .err(code):{if(code!=1002 || work!=16777217){return .err(9012);}}};
 // Also verify an invalid owner cannot bypass an already overdrawn meter.
 work=16777217;switch(bound::derive(ref produced,ref table,512,ref mut work)){case .ok(_):{return .err(9013);}case .err(code):{if(code!=1002 || work!=16777217){return .err(9014);}}};
 table=coordinates::assemble(bytes::empty(),v::empty<coordinates::Coordinate>());authored=bytes::empty();return .ok(result);
}
@effects(console)fn i32 main(){
 var input=bytes::empty();var u8[4096] chunk=[0;4096];while(true){let n=switch(io::read_stdin(ref mut chunk)){case .ok(n):{n}case .err(_):{return 1;}};if(n==0){break;}let ref u8[] raw=ref chunk;if(bytes::len(ref input)>2097152-n || !bytes::append(ref mut input,raw[..n]).is_ok){return 2;}}
 let result=switch(bind(bytes::view(ref input))){case .ok(p):{p}case .err(code):{discard io::write_u32(code);return 0;}};input=bytes::empty();
 if(!io::write_bytes(context::coordinate(bound::context(ref result))).is_ok || !io::write_bytes(context::descriptor(bound::context(ref result))).is_ok || !io::write("\n").is_ok || !io::write_bytes(hash::digest(bound::identity(ref result))).is_ok || !io::write("\n").is_ok || !io::write_bytes(hash::canonical(bound::identity(ref result))).is_ok){return 3;}return 0;
}`

func TestCandidateStateGraphSourceIdentity(t *testing.T) {
	type trial struct {
		name         string
		packet, want []byte
	}
	var trials []trial
	var plain, helpers stateGraphAuthoredTrial
	for _, authored := range stateGraphAuthoredTrials() {
		switch authored.name {
		case "read-local", "branch-join", "borrow-alias-view-copy", "helper-branch", "helper-before-root", "helper-transitive", "helper-read-preserves":
			f, _ := stateGraphAuthoredOracle(t, authored)
			for _, owner := range []uint64{0, 7, 511} {
				want := stateSourceIdentityWant(f)
				names := f.coordinate.String()
				trials = append(trials, trial{fmt.Sprintf("%s/owner-%d", authored.name, owner), stateSourceIdentityPacket(owner, []stateSourceIdentityRow{{owner, 0, uint64(len(names))}}, names, authored.text), want})
			}
			if authored.name == "read-local" {
				plain = authored
			}
			if authored.name == "helper-branch" {
				helpers = authored
			}
		}
	}
	addSource := func(authored stateGraphAuthoredTrial, modulePath ...string) stateBodyFixture {
		t.Helper()
		var f stateBodyFixture
		if len(modulePath) == 0 {
			f, _ = stateGraphAuthoredOracle(t, authored)
		} else {
			// Go enforces source-root-relative module/file correspondence. Compile
			// the renamed actual source under its real path, without editing HIR.
			path := strings.ReplaceAll(modulePath[0], "::", "/") + ".fy"
			parsed, err := frontend.AnalyzeSyntax(source.NewFile(path, []byte(authored.text)), source.DefaultLimits())
			if err != nil || parsed.AST == nil || len(parsed.Tree.Diagnostics) != 0 {
				t.Fatalf("renamed source syntax: %v %+v", err, parsed.Tree.Diagnostics)
			}
			result, err := compiler.CheckExperimentalStateGraph(semantic.File{Package: 7, Path: path, Module: parsed.AST})
			if err != nil || !result.Verified {
				t.Fatalf("renamed source compiler: %v %+v", err, result.Semantic.Diagnostics)
			}
			coordinate, _ := stateGraphAuthoredOracle(t, plain)
			f = stateGraphBodyFixture(t, authored.name, result.Semantic.HIR, coordinate.coordinate)
		}
		names := f.coordinate.String()
		want := stateSourceIdentityWant(f)
		trials = append(trials, trial{authored.name, stateSourceIdentityPacket(7, []stateSourceIdentityRow{{7, 0, uint64(len(names))}}, names, authored.text), want})
		return f
	}
	base := addSource(plain)
	for _, module := range []string{"renamed", "renamed::vault", "renamed /* retained trivia */ :: vault"} {
		changed := plain
		changed.name = "actual-module-" + module
		changed.text = strings.Replace(changed.text, "module ledger;", "module "+module+";", 1)
		path := strings.ReplaceAll(module, " /* retained trivia */ ", "")
		path = strings.ReplaceAll(path, " ", "")
		f := addSource(changed, path)
		if bytes.Equal(f.descriptor, base.descriptor) || f.hash == base.hash {
			t.Fatal("real source module rename preserved identity")
		}
	}
	reordered := plain
	reordered.name = "effect-order"
	reordered.text = strings.Replace(plain.text, "storage_read,storage_write", "storage_write,storage_read", 1)
	if f := addSource(reordered); f.hash != base.hash || !bytes.Equal(f.descriptor, base.descriptor) {
		t.Fatal("effect order leaked into canonical identity")
	}
	for _, effect := range []string{"", "storage_read", "storage_write"} {
		body := "storage::commit();"
		if effect == "storage_read" {
			body = `let x=storage::read("count");` + body
		}
		if effect == "storage_write" {
			body = `storage::delete("count");` + body
		}
		addSource(stateGraphAuthoredTrial{"effects-" + effect, stateGraphAuthoredSource(effect, body, ""), ""})
	}
	hf := addSource(helpers)
	// Move the actual helper declarations before the root, changing local IDs.
	root := strings.Index(helpers.text, "@effects(storage_read,storage_write)fn void transaction()")
	tail := strings.Index(helpers.text[root:], "}@effects(") + root + 1
	ordered := helpers
	ordered.name = "helper-declaration-order"
	ordered.text = helpers.text[:root] + helpers.text[tail:] + helpers.text[root:tail]
	if f := addSource(ordered); f.hash != hf.hash || !bytes.Equal(f.descriptor, hf.descriptor) {
		t.Fatal("helper declaration order leaked into identity")
	}
	changedBody := plain
	changedBody.name = "actual-body-change"
	changedBody.text = strings.Replace(plain.text, `storage::write("count",x);`, `storage::delete("count");`, 1)
	if f := addSource(changedBody); f.hash == base.hash || !bytes.Equal(f.descriptor, base.descriptor) {
		t.Fatal("actual graph change must affect only body identity")
	}
	other := base.coordinate
	other[0] ^= 1
	otherFixture := stateGraphBodyFixture(t, "coordinate-change", base.input, other)
	want := stateSourceIdentityWant(otherFixture)
	if otherFixture.hash == base.hash {
		t.Fatal("package coordinate did not bind body hash")
	}
	trials = append(trials, trial{"coordinate-change", stateSourceIdentityPacket(7, []stateSourceIdentityRow{{7, 0, 77}}, other.String(), plain.text), want})

	names := base.coordinate.String()
	valid := stateSourceIdentityRow{7, 0, uint64(len(names))}
	for _, bad := range []struct {
		name  string
		owner uint64
		rows  []stateSourceIdentityRow
		names string
	}{
		{"duplicate", 7, []stateSourceIdentityRow{valid, valid}, names},
		{"missing", 7, nil, names},
		{"wrong-owner", 8, []stateSourceIdentityRow{valid}, names},
		{"zero", 7, []stateSourceIdentityRow{valid}, "fypc1-sha256-" + strings.Repeat("0", 64)},
		{"malformed", 7, []stateSourceIdentityRow{valid}, names[:76] + "G"},
		{"wrong-prefix", 7, []stateSourceIdentityRow{valid}, "fypi1" + names[5:]},
		{"truncated", 7, []stateSourceIdentityRow{{7, 0, 76}}, names},
		{"reversed-span", 7, []stateSourceIdentityRow{{7, 77, 0}}, names},
		{"out-of-bounds-span", 7, []stateSourceIdentityRow{{7, 0, 78}}, names},
		{"owner-512", 512, []stateSourceIdentityRow{{512, 0, 77}}, names},
		{"owner-max", ^uint64(0), []stateSourceIdentityRow{{^uint64(0), 0, 77}}, names},
		{"too-many-rows", 7, make([]stateSourceIdentityRow, 513), names},
	} {
		trials = append(trials, trial{bad.name, stateSourceIdentityPacket(bad.owner, bad.rows, bad.names, plain.text), []byte("1001")})
	}
	// validate_owner ignores unrelated malformed, duplicate and out-of-range rows.
	ignored := []stateSourceIdentityRow{{512, 999, 1}, {99, 0, 999}, {99, 99, 0}, valid}
	trials = append(trials, trial{"unrelated-rows-ignored", stateSourceIdentityPacket(7, ignored, names, plain.text), stateSourceIdentityWant(base)})
	candidateAtlasProbeSize(t, "module_state_graph_source_identity", stateSourceIdentityProbe, 512<<10, func(t *testing.T, image string) {
		for _, trial := range trials {
			t.Run(trial.name, func(t *testing.T) {
				code, out := candidateAtlasProcess(t, image, trial.packet)
				if code != 0 || !bytes.Equal([]byte(out), trial.want) {
					t.Fatalf("exit=%d lengths=%d/%d mismatch=%d prefix=%x", code, len(out), len(trial.want), stateBodyFirstMismatch([]byte(out), trial.want), []byte(out[:min(len(out), 80)]))
				}
			})
		}
	}, "candidate_artifact_wire")
}

func TestCandidateStateGraphSourceIdentitySealedOwner(t *testing.T) {
	for _, trial := range []struct{ name, code, body string }{
		{"constructor", "E0103", `fn bound::Bound forge(){return bound::Bound{.context=0,.identity=0};}`},
		{"immutable-context", "E1008", `fn void forge(ref bound::Bound p){let ref mut context::Context c=ref mut bound::context(p);}`},
		{"immutable-identity", "E1008", `fn void forge(ref bound::Bound p){let ref mut hash::Identity h=ref mut bound::identity(p);}`},
		{"immutable-descriptor", "E1008", `fn void forge(ref bound::Bound p){let ref mut u8[] raw=ref mut context::descriptor(bound::context(p));}`},
		{"immutable-coordinate", "E1008", `fn void forge(ref bound::Bound p){let ref mut u8[] raw=ref mut context::coordinate(bound::context(p));}`},
		{"immutable-canonical", "E1008", `fn void forge(ref bound::Bound p){let ref mut u8[] raw=ref mut hash::canonical(bound::identity(p));}`},
		{"immutable-digest", "E1008", `fn void forge(ref bound::Bound p){let ref mut u8[] raw=ref mut hash::digest(bound::identity(p));}`},
	} {
		t.Run(trial.name, func(t *testing.T) {
			root := t.TempDir()
			copyLexerExample(t, root)
			copyCandidateParityModules(t, root, "module_state_graph_source_identity")
			writeTestSource(t, root, "src/main.fy", "module main;import module_state_graph_source_identity as bound;import module_state_graph_hash_context as context;import module_state_graph_body_hash as hash;"+trial.body+"fn i32 main(){return 0;}")
			workspace, err := workspaceloader.LoadVectorCandidate(root)
			if err != nil {
				t.Fatal(err)
			}
			var diagnostic bytes.Buffer
			_, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "sealed source identity", &diagnostic)
			if code == ExitSuccess || !strings.Contains(diagnostic.String(), "main.fy:") || !strings.Contains(diagnostic.String(), trial.code) {
				t.Fatalf("seal refusal code=%d diagnostics=%s", code, &diagnostic)
			}
		})
	}
}
