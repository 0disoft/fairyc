module tests::module_unit_iteration_types;
import module_unit_origin_diagnostics as diagnostics;
import module_unit_origin_returns as origins;
import module_unit_loan_live_model as live;
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



@effects(console) fn void checked(ref u8[] source,u32 expected){var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(source)).is_ok);switch(checker::check(ref sources)){case .ok(p):{assert(expected==0 && diagnostics::exit_code(ref p)==0);}case .err(p):{assert(expected==2 && p.phase==4 && p.code==2);}};}
@test @effects(console) fn void loop_slice_sources_are_typed_before_origin_analysis(){
    checked(b"module main;fn void f(ref mut u8[] p){for(let ref mut u8 item in p[..1]){*item=9;}}",0);
    checked(b"module main;fn void f(ref u8[] p){for(let u8 item in p[..1]){discard item;}}",0);
    checked(b"module main;fn void f(){var i64[2] x=[1,2];for(let ref mut i64 item in x){*item=9;}}",0);
    checked(b"module main;fn void f(){var u8[2] x=[1,2];for(let i32 item in x){discard item;}}",2);
    checked(b"module main;fn void f(){let u8[2] x=[1,2];for(let ref mut u8 item in x){*item=9;}}",2);
}
