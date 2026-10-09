module tests::module_unit_storage_generations;
import module_unit_origin_diagnostics as diagnostics;
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
@effects(console) fn diagnostics::Report checked(ref u8[] root,ref u8[] provider){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return checked(root,provider);}};let typed=primary(ref sources,ref metadata);let p=switch(origins::build(ref sources,ref metadata,ref typed)){case .ok(p):{p}case .err(p):{discard io::write("problem ");discard io::write_u32(p.code);discard io::write(" ");discard io::write_usize(p.start);discard io::write("\n");assert(false);return checked(root,provider);}};return switch(diagnostics::collect(ref p)){case .ok(p):{p}case .err(_):{assert(false);checked(root,provider)}};}
@effects(console) fn void emit(usize case_index,ref diagnostics::Report p){
    discard io::write("case ");discard io::write_usize(case_index);discard io::write(" ");discard io::write_u32(diagnostics::exit_code(p));discard io::write(" ");discard io::write_usize(diagnostics::visible_count(p));discard io::write("\n");var usize i=0;while(i<diagnostics::count(p)){if(!diagnostics::visible_at(p,i)){i+=1;continue;}let d=diagnostics::at(p,i);discard io::write("diagnostic ");discard io::write_u32(d.code);discard io::write(" ");discard io::write_usize(d.module_index);discard io::write(" ");discard io::write_usize(d.start);discard io::write(" ");discard io::write_usize(d.end);discard io::write(" ");discard io::write_usize(d.related_count);discard io::write("\n");var usize j=0;while(j<d.related_count){let r=diagnostics::related_at(p,d,j);discard io::write("related ");discard io::write_u32(r.role);discard io::write(" ");discard io::write_usize(r.module_index);discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write("\n");j+=1;}i+=1;}
}


@effects(filesystem) fn bytes::Buffer sdk(){var bytes::Buffer path=bytes::empty();assert(bytes::resize_zeroed(ref mut path,4096).is_ok);let n=switch(process::arg(1,bytes::view_mut(ref mut path))){case .ok(n):{n}case .err(_):{assert(false);return sdk();}};return switch(files::load(bytes::view(ref path)[..n])){case .ok(b):{b}case .err(_):{assert(false);sdk()}};}
// Exact Go diagnostic source anchors, independent of report insertion order.
fn void golden(ref diagnostics::Report p,ref usize[] wanted){assert(wanted.len==diagnostics::visible_count(p)*7);var usize at=0;while(at<wanted.len){var bool found=false;var usize i=0;while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);if(d.code==2101 && d.module_index==wanted[at] && d.start==wanted[at+1] && d.end==wanted[at+2] && d.related_count==2){let a=diagnostics::related_at(p,d,0);let b=diagnostics::related_at(p,d,1);let loan=if(a.role==4){a}else{b};let use=if(a.role==5){a}else{b};if(loan.role==4 && use.role==5 && loan.module_index==d.module_index && use.module_index==d.module_index && loan.start==wanted[at+3] && loan.end==wanted[at+4] && use.start==wanted[at+5] && use.end==wanted[at+6]){found=true;}}i+=1;}assert(found);at+=7;}}
@effects(console) fn void check_case(usize index,u32 expected,usize count,ref u8[] text,ref u8[] provider,ref usize[] wanted){let p=checked(text,provider);emit(index,ref p);assert(diagnostics::exit_code(ref p)==expected && diagnostics::visible_count(ref p)==count);if(count!=0){golden(ref p,wanted);}var usize i=0;while(i<diagnostics::count(ref p)){let d=diagnostics::at(ref p,i);assert(d.code==2101 && d.related_count==2);i+=1;}}
@test @effects(filesystem,console) fn void sdk_mutable_storage_views_freeze_original_parents(){let source=sdk();
    let usize[7] wanted0=[0,133,138,92,113,148,153];
    check_case(0,3,1,b"module main;import std::vec as v;fn void f(ref mut v::Vec<u8> value){let ref mut u8[] child=v::view_mut_u8(value);discard v::len<u8>(value);discard child[0];}",bytes::view(ref source),ref wanted0);
    let usize[1] wanted1=[0];
    check_case(1,0,0,b"module main;import std::vec as v;fn void f(ref mut v::Vec<u8> value){let ref mut u8[] child=v::view_mut_u8(value);discard child[0];discard v::len<u8>(value);}",bytes::view(ref source),ref wanted1);
    let usize[7] wanted2=[0,162,167,92,113,177,182];
    check_case(2,3,1,b"module main;import std::vec as v;fn void f(ref mut v::Vec<u8> value){let ref mut u8[] child=v::view_mut_u8(value);let ref mut u8[] alias=child;discard v::len<u8>(value);discard alias[0];}",bytes::view(ref source),ref wanted2);
    let usize[1] wanted3=[0];
    check_case(3,0,0,b"module main;import std::vec as v;fn void f(ref mut v::Vec<u8> value){let ref u8[] child=v::view_u8(value);discard v::len<u8>(value);discard child[0];}",bytes::view(ref source),ref wanted3);
    let usize[7] wanted4=[0,134,139,92,114,149,154];
    check_case(4,3,1,b"module main;import std::vec as v;fn void f(ref mut v::Vec<u8> value){let ref mut u8[] child=v::view_mut<u8>(value);discard v::len<u8>(value);discard child[0];}",bytes::view(ref source),ref wanted4);
    let usize[7] wanted5=[0,138,143,94,117,153,158];
    check_case(5,3,1,b"module main;import std::vec as v;fn void f(ref mut v::Vec<i64> value){let ref mut i64[] child=v::view_mut<i64>(value);discard v::len<i64>(value);discard child[0];}",bytes::view(ref source),ref wanted5);
    let usize[7] wanted6=[0,166,171,120,144,181,186];
    check_case(6,3,1,b"module main;import std::vec as v;struct Item{i32 number;}fn void f(ref mut v::Vec<Item> value){let ref mut Item[] child=v::view_mut<Item>(value);discard v::len<Item>(value);discard child[0].number;}",bytes::view(ref source),ref wanted6);
    let usize[7] wanted7=[0,148,153,106,127,162,167];
    check_case(7,3,1,b"module main;import std::vec as v;fn void f(ref mut v::Vec<u8> value){let Option<ref mut u8[]> child=.some(v::view_mut_u8(value));discard v::len<u8>(value);switch(child){case .some(p):{discard p[0];}case .none:{}};}",bytes::view(ref source),ref wanted7);
    let usize[1] wanted8=[0];
    check_case(8,0,0,b"module main;import std::vec as v;fn void f(ref mut v::Vec<u8> value){var Option<ref mut u8[]> child=.some(v::view_mut_u8(value));child=.none;discard v::len<u8>(value);discard child;}",bytes::view(ref source),ref wanted8);
}
