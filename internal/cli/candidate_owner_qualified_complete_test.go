package cli

import (
	"fmt"
	"strings"
	"testing"
)

// Exercise checking before Atlas projection. In particular, selecting a nominal
// method owner must not authorize an invalid borrow or consume a resource twice.
func TestCandidateOwnerQualifiedComplete(t *testing.T) {
	type fixture struct {
		name, source string
		code         int
	}
	var fixtures []fixture
	for _, f := range atlasOwnerQualifiedFixtures() {
		if f.member == "" {
			fixtures = append(fixtures, fixture{f.name, f.source, 0})
		}
	}
	for _, f := range atlasMoveImplicitBorrowCases() {
		fixtures = append(fixtures, fixture{f.name, f.source, 0})
	}
	const returned = `module main;resource Token{i32 value;cleanup{}}
fn ref Token Token::view(ref Token this){return this;}
fn ref mut Token Token::edit(ref mut Token this){return this;}
fn void Token::set(ref mut Token this,i32 n){this.value=n;}`
	fixtures = append(fixtures,
		fixture{"implicit returned shared borrow", implicitReceiverReturnedSource, 0},
		fixture{"implicit returned mutable borrow", returned + `fn i32 main(){var token=Token{.value=1};let view=token.edit();view.value=2;return token.value;}`, 0},
		fixture{"implicit forwarded shared parameter", returned + `fn ref Token forward(ref Token token){return token.view();}fn i32 main(){return 0;}`, 0},
		fixture{"implicit forwarded mutable parameter", returned + `fn ref mut Token forward(ref mut Token token){return token.edit();}fn i32 main(){return 0;}`, 0},
		fixture{"implicit local borrow cannot escape", returned + `fn ref Token escape(){let token=Token{.value=1};return token.view();}fn i32 main(){return 0;}`, 2102},
		fixture{"implicit mutable local borrow cannot escape", returned + `fn ref mut Token escape(){var token=Token{.value=1};return token.edit();}fn i32 main(){return 0;}`, 2102},
		fixture{"implicit returned shared conflict", returned + `fn i32 main(){var token=Token{.value=1};let view=token.view();token.set(2);return view.value;}`, 2101},
		fixture{"implicit returned mutable conflict", returned + `fn i32 main(){var token=Token{.value=1};let view=token.edit();let copy=token.value;return view.value+copy;}`, 2101},
	)
	fixtures = append(fixtures,
		fixture{"implicit immutable exclusive borrow", `module main;struct Cell{i32 value;}fn void Cell::set(ref mut Cell this,i32 n){this.value=n;}fn i32 main(){let cell=Cell{.value=1};cell.set(2);return 0;}`, 1008},
		fixture{"ordinary argument is not implicit", `module main;struct Cell{i32 value;}fn i32 read(ref Cell cell){return cell.value;}fn i32 main(){let cell=Cell{.value=1};return read(cell);}`, 1006},
		fixture{"static call is not implicit", `module main;struct Cell{i32 value;}fn i32 Cell::read(ref Cell this){return this.value;}fn i32 main(){let cell=Cell{.value=1};return Cell::read(cell);}`, 1006},
		fixture{"implicit consumed receiver", `module main;resource Token{i32 value;cleanup{}}fn i32 Token::read(ref Token this){return this.value;}fn i32 main(){let token=Token{.value=1};let other=token;return token.read();}`, 1101},
		fixture{"implicit active shared loan", `module main;struct Cell{i32 value;}fn void Cell::set(ref mut Cell this,i32 n){this.value=n;}fn i32 main(){var cell=Cell{.value=1};let view=ref cell;cell.set(2);return view.value;}`, 2101},
		fixture{"wrong receiver", `module main;struct Cell{i32 value;}struct Other{i32 value;}fn i32 Cell::bad(ref Other this){return this.value;}fn i32 main(){return 42;}`, 4501},
		fixture{"late receiver", `module main;struct Cell{i32 value;}fn i32 Cell::bad(i32 value,ref Cell this){return value+this.value;}fn i32 main(){return 42;}`, 4501},
		fixture{"generic owner remains closed", `module main;struct Cell<T>{T value;}fn i32 Cell::bad(ref Cell<i32> this){return this.value;}fn i32 main(){return 42;}`, 9},
		fixture{"immutable exclusive borrow", `module main;struct Cell{i32 value;}fn void Cell::set(ref mut Cell this,i32 value){this.value=value;}fn i32 main(){let cell=Cell{.value=0};let view=ref mut cell;view.set(42);return cell.value;}`, 1008},
		fixture{"consumed receiver", `module main;resource Token{i32 value;cleanup{}}fn i32 Token::take(Token this){return this.value;}fn i32 main(){let token=Token{.value=42};discard token.take();return token.take();}`, 1101},
		fixture{"associated function is not a method", `module main;struct Cell{i32 value;}fn i32 Cell::read(ref Cell other){return other.value;}fn i32 main(){let cell=Cell{.value=42};let view=ref cell;return view.read();}`, 5},
	)
	const main = `module main;
import project_compiler_core as compiler;import module_catalog as catalog;
import module_public_type_text as coordinates;import syntax as syntax;import scanner as scanner;
import std::vec as v;import std::bytes as bytes;import std::io as io;
fn syntax::Parsed parsed(ref u8[] raw){let b=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(raw);}};return switch(syntax::parse(s,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(raw)}};}
fn u32 check(ref u8[] raw){
 var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(raw)).is_ok);
 var owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok);let imports=v::empty<catalog::Binding>();
 let names=switch(bytes::copy(b"fypc1-sha256-1111111111111111111111111111111111111111111111111111111111111111")){case .ok(p):{p}case .err(_):{return 2;}};
 var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=bytes::len(ref names)}).is_ok);let table=coordinates::assemble(names,rows);
 var usize source_work=0;var usize link_work=0;return switch(compiler::build_complete_sources(ref sources,ref owners,ref imports,ref table,ref mut source_work,ref mut link_work)){case .ok(_):{0}case .err(e):{e.code}};
}
@effects(console) fn i32 main(){
`
	var program, expected strings.Builder
	program.WriteString(main)
	for _, f := range fixtures {
		root := writeWorkspaceSessionFixture(t, f.source)
		code, _, diagnostic := runForTest([]string{"check"}, root)
		if (code == ExitSuccess) != (f.code == 0) {
			t.Fatalf("Go owner admission %s: %d %s", f.name, code, diagnostic)
		}
		fmt.Fprintf(&program, "discard io::write_usize(check(b%q) as usize);discard io::write(\",\");\n", f.source)
		fmt.Fprintf(&expected, "%d,", f.code)
	}
	program.WriteString("return 0;}\n")
	candidateAtlasProbeSize(t, "project_compiler_core", program.String(), 2<<20, func(t *testing.T, image string) {
		t.Setenv("PATH", "")
		code, out := candidateAtlasProcess(t, image, nil)
		if code != 0 || out != expected.String() {
			t.Fatalf("owner Complete admission: %d %q, want %q; fixtures=%+v", code, out, expected.String(), fixtures)
		}
	})
}
