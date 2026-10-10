module tests::module_unit_owner_reads;
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


// Frozen complete Go diagnostic coordinates, including move owner anchors.
fn void golden(ref diagnostics::Report p,ref usize[] wanted){assert(wanted.len!=0);assert(diagnostics::visible_count(p)==wanted[0]);if(wanted[0]==0){return;}assert(wanted[0]==1 && wanted.len>=6);assert(wanted.len==6+wanted[5]*4);var bool found=false;var usize i=0;while(i<diagnostics::count(p)){let d=diagnostics::at(p,i);if(diagnostics::visible_at(p,i) && d.code as usize==wanted[1] && d.module_index==wanted[2] && d.start==wanted[3] && d.end==wanted[4] && d.related_count==wanted[5]){var usize j=0;var bool complete=true;while(j<wanted[5]){var bool related=false;var usize k=0;while(k<d.related_count){let r=diagnostics::related_at(p,d,k);let at=6+j*4;if(r.role as usize==wanted[at] && r.module_index==wanted[at+1] && r.start==wanted[at+2] && r.end==wanted[at+3]){related=true;}k+=1;}complete=complete && related;j+=1;}found=found || complete;}i+=1;}assert(found);}
@effects(console) fn void check_case(usize index,u32 expected,ref u8[] text,ref u8[] provider,ref usize[] wanted){let p=checked(text,provider);emit(index,ref p);assert(diagnostics::exit_code(ref p)==expected);golden(ref p,wanted);}
@test @effects(console) fn void competing_owner_reads_follow_live_loans(){
    let usize[14] wanted0=[1,2101,0,72,73,2,5,0,83,84,4,0,54,63];
    check_case(0,3,b"module main;fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;discard x;discard *p;}",b"",ref wanted0);
    let usize[1] wanted1=[0];
    check_case(1,0,b"module main;fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;discard *p;discard x;}",b"",ref wanted1);
    let usize[1] wanted2=[0];
    check_case(2,0,b"module main;fn void f(){var i32 x=1;let ref i32 p=ref x;discard x;discard *p;}",b"",ref wanted2);
    let usize[1] wanted3=[0];
    check_case(3,0,b"module main;fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;discard *p;}",b"",ref wanted3);
    let usize[14] wanted4=[1,2101,0,92,93,2,5,0,103,104,4,0,54,63];
    check_case(4,3,b"module main;fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;let ref mut i32 q=p;discard x;discard *q;}",b"",ref wanted4);
    let usize[1] wanted5=[0];
    check_case(5,0,b"module main;fn void f(){var i32 x=1;var i32 y=2;let ref mut i32 p=ref mut x;discard y;discard *p;}",b"",ref wanted5);
    let usize[14] wanted6=[1,2101,0,87,88,2,5,0,96,97,4,0,68,77];
    check_case(6,3,b"module main;fn void f(){var i32 x=1;let Option<ref mut i32> p=.some(ref mut x);discard x;switch(p){case .some(q):{discard *q;}case .none:{}};}",b"",ref wanted6);
    let usize[14] wanted7=[1,2101,0,86,87,2,5,0,86,87,4,0,76,85];
    check_case(7,3,b"module main;fn void both(ref mut i32 p,i32 n){}fn void f(){var i32 x=1;both(ref mut x,x);}",b"",ref wanted7);
    let usize[1] wanted8=[0];
    check_case(8,0,b"module main;fn void both(ref i32 p,i32 n){}fn void f(){var i32 x=1;both(ref x,x);}",b"",ref wanted8);
    let usize[14] wanted9=[1,2101,0,79,80,2,5,0,92,93,4,0,61,70];
    check_case(9,3,b"module main;fn void f(){var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;discard x[0];discard p[0];}",b"",ref wanted9);
    let usize[14] wanted10=[1,2101,0,90,91,2,5,0,102,103,4,0,72,81];
    check_case(10,3,b"module main;struct S{i32 n;}fn void f(){var S x=S{.n=1};let ref mut S p=ref mut x;discard x.n;discard p.n;}",b"",ref wanted10);
    let usize[14] wanted11=[1,2101,0,101,102,2,5,0,113,114,4,0,83,92];
    check_case(11,3,b"module main;resource R{i32 n;cleanup{}}fn void f(){var R x=R{.n=1};let ref mut R p=ref mut x;discard x.n;discard p.n;}",b"",ref wanted11);
    let usize[18] wanted12=[1,2105,0,99,100,3,5,0,110,111,4,0,88,93,6,0,70,71];
    check_case(12,3,b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(){let R x=R{};let ref R p=ref x;take(x);discard p;}",b"",ref wanted12);
    let usize[18] wanted13=[1,2105,0,107,108,3,5,0,118,119,4,0,92,101,6,0,70,71];
    check_case(13,3,b"module main;resource R{cleanup{}}fn void take(R r){}fn void f(){var R x=R{};let ref mut R p=ref mut x;take(x);discard p;}",b"",ref wanted13);
    let usize[14] wanted14=[1,2101,0,82,85,2,5,0,96,97,4,0,72,81];
    check_case(14,3,b"module main;struct S{i32 n;}fn void f(){var S x=S{.n=1};let ref mut S p=ref mut x;x.n=2;discard p.n;}",b"",ref wanted14);
    let usize[14] wanted15=[1,2101,0,71,75,2,5,0,86,87,4,0,61,70];
    check_case(15,3,b"module main;fn void f(){var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;x[0]=3;discard p[0];}",b"",ref wanted15);
    let usize[14] wanted16=[1,2101,0,94,99,2,5,0,110,111,4,0,79,88];
    check_case(16,3,b"module main;fn void read(ref i32 p){}fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;read(ref x);discard *p;}",b"",ref wanted16);
    let usize[1] wanted17=[0];
    check_case(17,0,b"module main;fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;*p=*p+1;}",b"",ref wanted17);
    let usize[14] wanted18=[1,2101,0,86,96,2,5,0,105,106,4,0,61,70];
    check_case(18,3,b"module main;fn void f(){var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;let ref u8[] q=ref x[..1];discard p[0];discard q[0];}",b"",ref wanted18);
    let usize[14] wanted19=[1,2101,0,81,82,2,5,0,81,82,4,0,62,89];
    check_case(19,3,b"module main;fn void f(){var u8[2] x=[1,2];let ref mut u8 item=ref mut x[{discard x[0];0}];discard *item;}",b"",ref wanted19);
    let usize[1] wanted20=[0];
    check_case(20,0,b"module main;fn void f(){var u8[2] x=[1,2];let ref mut u8 item=ref mut x[x.len-1];discard *item;}",b"",ref wanted20);
    let usize[1] wanted21=[0];
    check_case(21,0,b"module main;resource R{ptr mut void data;usize length;cleanup{}}fn void f(){var R x=R{.data=null,.length=0};let p=unsafe{byte_slice::from_raw_parts_mut(ref mut x,x.data,x.length)};discard p;}",b"",ref wanted21);
    let usize[14] wanted22=[1,2101,0,178,179,2,5,0,178,179,4,0,152,161];
    check_case(22,3,b"module main;resource R{ptr mut void data;usize length;cleanup{}}fn void f(){var R x=R{.data=null,.length=0};let p=unsafe{byte_slice::from_raw_parts_mut(ref mut x,x.data,{discard x.length;0})};discard p;}",b"",ref wanted22);
    let usize[1] wanted23=[0];
    check_case(23,0,b"module main;resource R{usize length;cleanup{}}unsafe extern C fn u32 probe(ptr mut void data,usize length);fn void f(){var R x=R{.length=0};discard unsafe{probe(ref mut x as ptr mut void,x.length)};}",b"",ref wanted23);
    let usize[14] wanted24=[1,2101,0,179,188,2,5,0,224,225,4,0,152,157];
    check_case(24,3,b"module main;resource R{usize length;cleanup{}}unsafe extern C fn u32 probe(ptr mut void data,usize length);fn void f(){var R x=R{.length=0};let ref R p=ref x;discard unsafe{probe(ref mut x as ptr mut void,x.length)};discard p;}",b"",ref wanted24);
    let usize[14] wanted25=[1,2101,0,106,107,2,5,0,119,120,4,0,82,91];
    check_case(25,3,b"module main;fn void f(){var i32 x=1;var i32 y=2;var bool c=true;let ref mut i32 p=ref mut x;if(c){discard x;};discard *p;}",b"",ref wanted25);
    let usize[1] wanted26=[0];
    check_case(26,0,b"module main;fn void f(){var i32 x=1;var i32 y=2;var bool c=true;var ref mut i32 p=ref mut x;if(c){p=ref mut y;}else{p=ref mut y;};discard x;discard *p;}",b"",ref wanted26);
    let usize[14] wanted27=[1,2101,0,109,110,2,5,0,120,121,4,0,82,91];
    check_case(27,3,b"module main;fn void f(){var i32 x=1;var i32 y=2;var bool c=true;let ref mut i32 p=ref mut x;while(c){discard x;discard *p;break;}}",b"",ref wanted27);
    let usize[1] wanted28=[0];
    check_case(28,0,b"module main;fn void f(){var i32 x=1;var i32 y=2;var bool c=true;let ref mut i32 p=ref mut x;while(c){discard *p;discard x;break;}}",b"",ref wanted28);
    let usize[14] wanted29=[1,2101,0,120,121,2,5,0,110,111,4,0,82,91];
    check_case(29,3,b"module main;fn void f(){var i32 x=1;var i32 y=2;var bool c=true;let ref mut i32 p=ref mut x;while(c){discard *p;discard x;continue;}}",b"",ref wanted29);
    let usize[14] wanted30=[1,2101,0,107,108,2,5,0,118,119,4,0,88,97];
    check_case(30,3,b"module main;import provider as pr;fn void f(){var i32 x=1;let ref mut i32 p=pr::id<i32>(ref mut x);discard x;discard *p;}",b"module provider;pub fn ref mut T id<T>(ref mut T p){return p;}",ref wanted30);
}
