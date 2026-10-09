module tests::module_unit_owner_arguments;
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


// Source anchors are frozen from the Go reference and independently anchored
// call-held cases. Shared aliases deterministically report the furthest use.
fn void golden(ref diagnostics::Report p,ref usize[] wanted){assert(wanted.len==diagnostics::visible_count(p)*9);var usize at=0;while(at<wanted.len){var bool found=false;var usize i=0;while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);if(d.code==2105 && d.module_index==wanted[at] && d.start==wanted[at+1] && d.end==wanted[at+2] && d.related_count==3){var bool owner=false;var bool loan=false;var bool use=false;var usize j=0;while(j<3){let r=diagnostics::related_at(p,d,j);assert(r.module_index==d.module_index);if(r.role==6 && r.start==wanted[at+3] && r.end==wanted[at+4]){owner=true;}if(r.role==4 && r.start==wanted[at+5] && r.end==wanted[at+6]){loan=true;}if(r.role==5 && r.start==wanted[at+7] && r.end==wanted[at+8]){use=true;}j+=1;}if(owner && loan && use){found=true;}}i+=1;}assert(found);at+=9;}}
@effects(console) fn void check_case(usize index,u32 expected,usize count,ref u8[] text,ref u8[] provider,ref usize[] wanted){let p=checked(text,provider);emit(index,ref p);assert(diagnostics::exit_code(ref p)==expected && diagnostics::visible_count(ref p)==count);if(count!=0){golden(ref p,wanted);}var usize i=0;while(i<diagnostics::count(ref p)){let d=diagnostics::at(ref p,i);assert(d.code==2105 && d.related_count==3);i+=1;}}
@test @effects(console) fn void owner_moves_preserve_later_argument_borrows(){
    let usize[9] wanted0=[0,340,345,283,288,322,331,346,347];
    check_case(0,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void reverse(Owner q,ref Owner p){}fn i32 number(Owner p){return p.value;}fn void numeric(i32 n,ref Owner p){}fn void tagged(Owner q,Option<ref Owner> p){}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;reverse(owner,p);}",b"",ref wanted0);
    let usize[9] wanted1=[0,369,374,283,288,322,331,375,376];
    check_case(1,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void reverse(Owner q,ref Owner p){}fn i32 number(Owner p){return p.value;}fn void numeric(i32 n,ref Owner p){}fn void tagged(Owner q,Option<ref Owner> p){}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;let ref Owner q=p;observe(p);reverse(owner,q);}",b"",ref wanted1);
    let usize[9] wanted2=[0,347,352,283,288,322,331,354,355];
    check_case(2,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void reverse(Owner q,ref Owner p){}fn i32 number(Owner p){return p.value;}fn void numeric(i32 n,ref Owner p){}fn void tagged(Owner q,Option<ref Owner> p){}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;numeric(number(owner),p);}",b"",ref wanted2);
    let usize[9] wanted3=[0,354,359,283,288,336,345,360,361];
    check_case(3,3,1,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void reverse(Owner q,ref Owner p){}fn i32 number(Owner p){return p.value;}fn void numeric(i32 n,ref Owner p){}fn void tagged(Owner q,Option<ref Owner> p){}fn void f(){var Owner owner=Owner{.value=42};let Option<ref Owner> p=.some(ref owner);tagged(owner,p);}",b"",ref wanted3);
    let usize[1] wanted4=[0];
    check_case(4,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void reverse(Owner q,ref Owner p){}fn i32 number(Owner p){return p.value;}fn void numeric(i32 n,ref Owner p){}fn void tagged(Owner q,Option<ref Owner> p){}fn void f(){var Owner owner=Owner{.value=42};var Owner other=Owner{.value=1};let ref Owner p=ref other;reverse(owner,p);}",b"",ref wanted4);
    let usize[1] wanted5=[0];
    check_case(5,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void reverse(Owner q,ref Owner p){}fn i32 number(Owner p){return p.value;}fn void numeric(i32 n,ref Owner p){}fn void tagged(Owner q,Option<ref Owner> p){}fn void f(){var Owner owner=Owner{.value=42};var Owner other=Owner{.value=1};var ref Owner p=ref owner;reverse(owner,{p=ref other;p});}",b"",ref wanted5);
    let usize[1] wanted6=[0];
    check_case(6,0,0,b"module main;resource Owner{i32 value;cleanup{}}fn void observe(ref Owner p){}fn void consume(Owner p){}fn void reverse(Owner q,ref Owner p){}fn i32 number(Owner p){return p.value;}fn void numeric(i32 n,ref Owner p){}fn void tagged(Owner q,Option<ref Owner> p){}fn void f(){var Owner owner=Owner{.value=42};let ref Owner p=ref owner;observe(p);consume(owner);}",b"",ref wanted6);
}
