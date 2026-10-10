module tests::module_unit_partial_cleanup;
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
@test @effects(console) fn void partial_fields_and_arguments_keep_cleanup_obligations(){
    // field propagation
    let usize[10] wanted0=[1,7102,0,230,239,1,12,0,12,29];
    check_case(0,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n=number()?});}",b"",ref wanted0);
    // field authority
    let usize[1] wanted1=[0];
    check_case(1,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}@effects(console)fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n=number()?});}",b"",ref wanted1);
    // field transferred local
    let usize[10] wanted2=[1,7102,0,248,257,1,12,0,12,29];
    check_case(2,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){let R value=make();return .ok(Box{.first=value,.n=number()?});}",b"",ref wanted2);
    // argument propagation
    let usize[10] wanted3=[1,7102,0,279,288,1,12,0,12,29];
    check_case(3,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Box take(R value,i32 n){return Box{.first=value,.n=n};}fn Result<Box,i32> f(){return .ok(take(make(),number()?));}",b"",ref wanted3);
    // argument authority
    let usize[1] wanted4=[0];
    check_case(4,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Box take(R value,i32 n){return Box{.first=value,.n=n};}@effects(console)fn Result<Box,i32> f(){return .ok(take(make(),number()?));}",b"",ref wanted4);
    // completed transfer
    let usize[1] wanted5=[0];
    check_case(5,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n=1});}",b"",ref wanted5);
    // later owner after propagation
    let usize[1] wanted6=[0];
    check_case(6,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){return .ok(Box{.n=number()?,.first=make()});}",b"",ref wanted6);
    // completed nested call
    let usize[1] wanted7=[0];
    check_case(7,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn R pass(R value){return value;}fn Result<Box,i32> f(){return .ok(Box{.n=number()?,.first=pass(make())});}",b"",ref wanted7);
    // field explicit return
    let usize[10] wanted8=[1,7102,0,231,246,1,12,0,12,29];
    check_case(8,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n={return .err(1);0}});}",b"",ref wanted8);
    // aborted binding
    let usize[10] wanted9=[1,7102,0,232,247,1,12,0,12,29];
    check_case(9,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){let Box box=Box{.first=make(),.n={return .err(1);0}};return .ok(box);}",b"",ref wanted9);
    // loop break
    let usize[10] wanted10=[1,7102,0,247,253,1,12,0,12,29];
    check_case(10,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(bool c){while(c){let Box box=Box{.first=make(),.n={break;0}};}return .err(1);}",b"",ref wanted10);
    // loop continue
    let usize[10] wanted11=[1,7102,0,247,256,1,12,0,12,29];
    check_case(11,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(bool c){while(c){let Box box=Box{.first=make(),.n={continue;0}};}return .err(1);}",b"",ref wanted11);
    // inner loop preserves outer partial
    let usize[1] wanted12=[0];
    check_case(12,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n={while(true){break;}0}});}",b"",ref wanted12);
    // nested pending call
    let usize[10] wanted13=[1,7102,0,318,327,1,12,0,12,29];
    check_case(13,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn R pass(R value){return value;}fn Box take(R value,i32 n){return Box{.first=value,.n=n};}fn Result<Box,i32> f(){return .ok(take(pass(make()),number()?));}",b"",ref wanted13);
    // partial and live dedup
    let usize[19] wanted14=[2,7102,0,217,261,1,12,0,12,29,7102,0,249,258,1,12,0,12,29];
    check_case(14,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){let R older=make();return .ok(Box{.first=make(),.n=number()?});}",b"",ref wanted14);
    // two partial values dedup
    let usize[10] wanted15=[1,7102,0,276,285,1,12,0,12,29];
    check_case(15,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}resource Pair{R a;R b;i32 n;cleanup{}}fn Result<Pair,i32> f(){return .ok(Pair{.a=make(),.b=make(),.n=number()?});}",b"",ref wanted15);
    // tagged partial
    let usize[10] wanted16=[1,7102,0,291,300,1,12,0,12,29];
    check_case(16,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}resource Tagged{Option<R> value;i32 n;cleanup{}}fn Result<Tagged,i32> f(){return .ok(Tagged{.value=.some(make()),.n=number()?});}",b"",ref wanted16);
    // nested root partial
    let usize[10] wanted17=[1,7102,0,329,338,1,12,0,12,29];
    check_case(17,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Box make_box(){return Box{.first=make(),.n=1};}resource Outer{Box value;i32 n;cleanup{}}fn Result<Outer,i32> f(){return .ok(Outer{.value=make_box(),.n=number()?});}",b"",ref wanted17);
    // branch partial propagation
    let usize[10] wanted18=[1,7102,0,242,251,1,12,0,12,29];
    check_case(18,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(bool c){return .ok(Box{.first=make(),.n=if(c){number()?}else{1}});}",b"",ref wanted18);
    // no drop partial
    let usize[10] wanted19=[1,1108,0,366,375,1,11,0,175,192];
    check_case(19,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}@abilities(store)resource Linear{i32 n;cleanup{}}fn Linear linear(){return Linear{.n=1};}resource Pair{Linear first;i32 n;cleanup{}}fn Result<Pair,i32> f(){return .ok(Pair{.first=linear(),.n=number()?});}",b"",ref wanted19);
    // pending survives scalar declaration
    let usize[10] wanted20=[1,7102,0,262,264,1,12,0,12,29];
    check_case(20,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n={let Result<i32,i32> r=number();r?}});}",b"",ref wanted20);
    // loop authority
    let usize[1] wanted21=[0];
    check_case(21,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}@effects(console)fn Result<Box,i32> f(bool c){while(c){let Box box=Box{.first=make(),.n={break;0}};}return .err(1);}",b"",ref wanted21);
    // pending survives successful block
    let usize[1] wanted22=[0];
    check_case(22,0,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f(){return .ok(Box{.first=make(),.n={let i32 n=1;n}});}",b"",ref wanted22);
    // generic partial
    let usize[10] wanted23=[1,7102,0,233,242,1,12,0,12,29];
    check_case(23,3,b"module main;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{R first;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Box,i32> f<T>(){return .ok(Box{.first=make(),.n=number()?});}fn Result<Box,i32> use(){return f<i32>();}",b"",ref wanted23);
}
