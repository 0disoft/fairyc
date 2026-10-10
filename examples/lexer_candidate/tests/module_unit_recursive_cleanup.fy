module tests::module_unit_recursive_cleanup;
import module_unit_origin_diagnostics as diagnostics;
import module_unit_moves as moves;
import module_unit_origin_returns as origins;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import module_unit_origin_check as checker;
import std::process as process;
import file_source as files;
import module_unit_origin_inputs as lifetime_inputs;
import module_unit_origin_flow_model as lifetime;
import module_unit_origin_sets as lifetime_sets;
import module_unit_origin_provenance as lifetime_sites;
fn syntax::Parsed parsed(ref u8[] text){let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn values::Plan primary(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata){let bound=switch(binding::bind(sources,metadata,0,1)){case .ok(b):{b}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};let ready=switch(closure::build(sources,metadata,ref bound)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);return primary(sources,metadata);}};return switch(values::build(sources,metadata,ref ready)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);discard io::write("\n");assert(false);primary(sources,metadata)}};}
@effects(console) fn diagnostics::Report checked(ref u8[] root,ref u8[] provider){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}return switch(checker::check(ref sources)){case .ok(p):{p}case .err(p):{discard io::write("analysis problem ");discard io::write_u32(p.code);assert(false);checked(root,provider)}};}
@effects(console) fn void emit(usize case_index,ref diagnostics::Report p){
    discard io::write("case ");discard io::write_usize(case_index);discard io::write(" ");discard io::write_u32(diagnostics::exit_code(p));discard io::write(" ");discard io::write_usize(diagnostics::visible_count(p));discard io::write("\n");var usize i=0;while(i<diagnostics::count(p)){if(!diagnostics::visible_at(p,i)){i+=1;continue;}let d=diagnostics::at(p,i);discard io::write("diagnostic ");discard io::write_u32(d.code);discard io::write(" ");discard io::write_usize(d.module_index);discard io::write(" ");discard io::write_usize(d.start);discard io::write(" ");discard io::write_usize(d.end);discard io::write(" ");discard io::write_usize(d.related_count);discard io::write("\n");var usize j=0;while(j<d.related_count){let r=diagnostics::related_at(p,d,j);discard io::write("related ");discard io::write_u32(r.role);discard io::write(" ");discard io::write_usize(r.module_index);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");j+=1;}i+=1;}
}


// Complete frozen reference coordinates; each row includes all related spans.
fn void golden(ref diagnostics::Report p,ref usize[] wanted){assert(wanted.len!=0 && diagnostics::visible_count(p)==wanted[0]);var usize row=0;var usize offset=1;
    while(row<wanted[0]){assert(offset+5<=wanted.len);let usize related=wanted[offset+4];assert(offset+5+related*4<=wanted.len);var bool found=false;var usize i=0;
        while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);if(diagnostics::visible_at(p,i) && d.code as usize==wanted[offset] && d.module_index==wanted[offset+1] && d.start==wanted[offset+2] && d.end==wanted[offset+3] && d.related_count==related){var usize j=0;var bool complete=true;while(j<related){var bool matched=false;var usize k=0;while(k<d.related_count){let r=diagnostics::related_at(p,d,k);let at=offset+5+j*4;matched=matched || r.role as usize==wanted[at] && r.module_index==wanted[at+1] && r.start==wanted[at+2] && r.end==wanted[at+3];k+=1;}complete=complete && matched;j+=1;}found=found || complete;}i+=1;}assert(found);offset+=5+related*4;row+=1;
    }assert(offset==wanted.len);
}
@effects(console) fn void check_case(usize index,u32 expected,ref u8[] text,ref u8[] provider,ref usize[] wanted){let p=checked(text,provider);emit(index,ref p);assert(diagnostics::exit_code(ref p)==expected);golden(ref p,wanted);}
@test @effects(console) fn void nested_fields_retain_cleanup_effect_authority(){
    // plain nested field
    let usize[10] wanted0=[1,7102,0,150,173,1,12,0,12,29];
    check_case(0,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(){let Box value=make();}",b"",ref wanted0);
    // nested authority
    let usize[1] wanted1=[0];
    check_case(1,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}@effects(console)fn void f(){let Box value=make();}",b"",ref wanted1);
    // whole root transfer
    let usize[1] wanted2=[0];
    check_case(2,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn Box f(Box value){return value;}",b"",ref wanted2);
    // two levels
    let usize[10] wanted3=[1,7102,0,185,224,1,12,0,12,29];
    check_case(3,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}resource Outer{Box inner;cleanup{}}fn void f(){let Outer value=Outer{.inner=make()};}",b"",ref wanted3);
    // repeated field requirement
    let usize[10] wanted4=[1,7102,0,186,229,1,12,0,12,29];
    check_case(4,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}resource Pair{Box a;Box b;cleanup{}}fn void f(){let Pair value=Pair{.a=make(),.b=make()};}",b"",ref wanted4);
    // repeated local requirement
    let usize[10] wanted5=[1,7102,0,150,186,1,12,0,12,29];
    check_case(5,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(){let Box a=make();let Box b=make();}",b"",ref wanted5);
    // nested Option
    let usize[10] wanted6=[1,7102,0,150,188,1,12,0,12,29];
    check_case(6,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(){let Option<Box> value=.some(make());}",b"",ref wanted6);
    // nested Result consumption
    let usize[10] wanted7=[1,7102,0,197,208,1,12,0,12,29];
    check_case(7,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(){let Result<Box,i32> value=.ok(make());discard value.is_ok;}",b"",ref wanted7);
    // live nested replacement
    let usize[10] wanted8=[1,7102,0,171,176,1,12,0,12,29];
    check_case(8,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn Box f(){var Box value=make();value=make();return value;}",b"",ref wanted8);
    // nested cleanup local
    let usize[10] wanted9=[1,7102,0,168,191,1,12,0,12,29];
    check_case(9,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}resource Holder{i32 n;cleanup{let Box value=make();}}",b"",ref wanted9);
    // nested cleanup authority
    let usize[1] wanted10=[0];
    check_case(10,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}@effects(console)resource Holder{i32 n;cleanup{let Box value=make();}}",b"",ref wanted10);
    // nested propagation exit
    let usize[19] wanted11=[2,7102,0,238,249,1,12,0,12,29,7102,0,250,264,1,12,0,12,29];
    check_case(11,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn Result<i32,i32> fallible(){return .ok(1);}fn Result<i32,i32> f(){let Box value=make();let i32 n=fallible()?;return .ok(n);}",b"",ref wanted11);
    // nested loop break
    let usize[10] wanted12=[1,7102,0,187,193,1,12,0,12,29];
    check_case(12,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(bool c){while(c){let Box value=make();break;}}",b"",ref wanted12);
    // nested loop continue
    let usize[10] wanted13=[1,7102,0,187,196,1,12,0,12,29];
    check_case(13,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(bool c){while(c){let Box value=make();continue;}}",b"",ref wanted13);
    // nested match capture
    let usize[10] wanted14=[1,7102,0,199,229,1,12,0,12,29];
    check_case(14,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(){let Option<Box> item=.some(make());switch(item){case .some(box):{discard box;}case .none:{}};}",b"",ref wanted14);
    // nested temporary
    let usize[10] wanted15=[1,7102,0,159,165,1,12,0,12,29];
    check_case(15,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(){discard make();}",b"",ref wanted15);
    // parent and child effects
    let usize[19] wanted16=[2,7102,0,180,203,1,12,0,66,86,7102,0,180,203,1,12,0,12,29];
    check_case(16,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}@effects(filesystem)resource Box{R inner;cleanup{discard 1;}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(){let Box value=make();}",b"",ref wanted16);
    // partial parent authority
    let usize[10] wanted17=[1,7102,0,200,223,1,12,0,12,29];
    check_case(17,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}@effects(filesystem)resource Box{R inner;cleanup{discard 1;}}fn Box make(){return Box{.inner=R{.n=1}};}@effects(filesystem)fn void f(){let Box value=make();}",b"",ref wanted17);
    // parent and child authority
    let usize[1] wanted18=[0];
    check_case(18,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}@effects(filesystem)resource Box{R inner;cleanup{discard 1;}}fn Box make(){return Box{.inner=R{.n=1}};}@effects(filesystem,console)fn void f(){let Box value=make();}",b"",ref wanted18);
    // no drop supersedes effects
    let usize[10] wanted19=[1,1108,0,167,190,1,11,0,66,83];
    check_case(19,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}@abilities(store)resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void f(){let Box value=make();}",b"",ref wanted19);
    // provider nested effect
    let usize[10] wanted20=[1,7102,0,44,73,1,12,1,16,33];
    check_case(20,3,b"module main;import provider as p;fn void f(){let p::Box value=p::make();}",b"module provider;@effects(console)resource R{i32 n;cleanup{discard 1;}}pub resource Box{R inner;cleanup{}}pub fn Box make(){return Box{.inner=R{.n=1}};}",ref wanted20);
    // generic nested effect
    let usize[10] wanted21=[1,7102,0,172,174,1,12,0,12,29];
    check_case(21,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}fn void dispose<T>(T n,Box value){}fn void f(){dispose<i32>(1,make());}",b"",ref wanted21);
    // generic nested authority
    let usize[1] wanted22=[0];
    check_case(22,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R inner;cleanup{}}fn Box make(){return Box{.inner=R{.n=1}};}@effects(console)fn void dispose<T>(T n,Box value){}@effects(console)fn void f(){dispose<i32>(1,make());}",b"",ref wanted22);
}
