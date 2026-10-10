module tests::module_unit_held_reads;
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
@test @effects(console) fn void exclusive_arguments_guard_referent_reads(){
    let usize[14] wanted0=[1,2101,0,206,208,2,5,0,207,208,4,0,204,205];
    check_case(0,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;take(p,*p);}",b"",ref wanted0);
    let usize[14] wanted1=[1,2101,0,179,181,2,5,0,180,181,4,0,177,178];
    check_case(1,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){take(p,*p);}",b"",ref wanted1);
    let usize[1] wanted2=[0];
    check_case(2,0,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){share(p,*p);}",b"",ref wanted2);
    let usize[14] wanted3=[1,2101,0,180,182,2,5,0,181,182,4,0,177,178];
    check_case(3,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){take(p,{*p=2;0});}",b"",ref wanted3);
    let usize[14] wanted4=[1,2101,0,181,183,2,5,0,182,183,4,0,178,179];
    check_case(4,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){share(p,{*p=2;0});}",b"",ref wanted4);
    let usize[1] wanted5=[0];
    check_case(5,0,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){take(p,0);discard *p;}",b"",ref wanted5);
    let usize[1] wanted6=[0];
    check_case(6,0,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){*p=*p+1;}",b"",ref wanted6);
    let usize[1] wanted7=[0];
    check_case(7,0,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){*p=read(p);}",b"",ref wanted7);
    let usize[14] wanted8=[1,2101,0,185,188,2,5,0,185,186,4,0,183,184];
    check_case(8,3,b"module main;struct S{i32 x;}fn void take(ref mut S p,i32 n){}fn void share(ref S p,i32 n){}fn void edit(ref mut S p){p.x=2;}fn void f(){var S x=S{.x=1};let ref mut S p=ref mut x;take(p,p.x);}",b"",ref wanted8);
    let usize[14] wanted9=[1,2101,0,154,157,2,5,0,154,155,4,0,152,153];
    check_case(9,3,b"module main;struct S{i32 x;}fn void take(ref mut S p,i32 n){}fn void share(ref S p,i32 n){}fn void edit(ref mut S p){p.x=2;}fn void f(ref mut S p){take(p,p.x);}",b"",ref wanted9);
    let usize[1] wanted10=[0];
    check_case(10,0,b"module main;struct S{i32 x;}fn void take(ref mut S p,i32 n){}fn void share(ref S p,i32 n){}fn void edit(ref mut S p){p.x=2;}fn void f(ref mut S p){share(p,p.x);}",b"",ref wanted10);
    let usize[14] wanted11=[1,2101,0,220,224,2,5,0,220,221,4,0,218,219];
    check_case(11,3,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(){var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;take(p,p[0]);}",b"",ref wanted11);
    let usize[14] wanted12=[1,2101,0,187,191,2,5,0,187,188,4,0,185,186];
    check_case(12,3,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(ref mut u8[] p){take(p,p[0]);}",b"",ref wanted12);
    let usize[1] wanted13=[0];
    check_case(13,0,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(ref mut u8[] p){share(p,p[0]);}",b"",ref wanted13);
    let usize[1] wanted14=[0];
    check_case(14,0,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(ref mut u8[] p){size(p,p.len);}",b"",ref wanted14);
    let usize[14] wanted15=[1,2101,0,196,197,2,5,0,196,197,4,0,188,202];
    check_case(15,3,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(ref mut u8[] p){discard p[{edit(p);0}];}",b"",ref wanted15);
    let usize[1] wanted16=[0];
    check_case(16,0,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(ref mut u8[] p){p[0]=p[1];}",b"",ref wanted16);
    let usize[1] wanted17=[0];
    check_case(17,0,b"module main;struct S{i32 x;}fn void take(ref mut S p,i32 n){}fn void share(ref S p,i32 n){}fn void edit(ref mut S p){p.x=2;}fn void f(ref mut S p){let ref i32 q=ref p.x;discard p.x;discard *q;}",b"",ref wanted17);
    let usize[14] wanted18=[1,2101,0,199,201,2,5,0,200,201,4,0,197,198];
    check_case(18,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){let ref mut i32 q=p;take(p,*q);}",b"",ref wanted18);
    let usize[14] wanted19=[1,2101,0,181,187,2,5,0,182,187,4,0,177,180];
    check_case(19,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){take((p),*((p)));}",b"",ref wanted19);
    let usize[14] wanted20=[1,2101,0,97,102,2,5,0,97,98,4,0,95,96];
    check_case(20,3,b"module main;struct S{usize len;}fn void take(ref mut S p,usize n){}fn void f(ref mut S p){take(p,p.len);}",b"",ref wanted20);
    let usize[1] wanted21=[0];
    check_case(21,0,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void reversed(i32 n,ref mut i32 p){}fn void f(ref mut i32 p){reversed(*p,p);}",b"",ref wanted21);
    let usize[1] wanted22=[0];
    check_case(22,0,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p,ref i32 q){take(p,*q);}",b"",ref wanted22);
    let usize[1] wanted23=[0];
    check_case(23,0,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p){take(p,{let ref mut i32 q=p;0});}",b"",ref wanted23);
    let usize[14] wanted24=[1,2101,0,192,194,2,5,0,193,194,4,0,184,185];
    check_case(24,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p,bool c){take(p,if(c){*p}else{0});}",b"",ref wanted24);
    let usize[14] wanted25=[1,2101,0,204,206,2,5,0,205,206,4,0,184,185];
    check_case(25,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn void f(ref mut i32 p,bool c){take(p,{while(c){discard *p;break;}0});}",b"",ref wanted25);
    let usize[14] wanted26=[1,2101,0,246,248,2,5,0,247,248,4,0,244,245];
    check_case(26,3,b"module main;fn void take(ref mut i32 p,i32 n){}fn void share(ref i32 p,i32 n){}fn void edit(ref mut i32 p){*p=2;}fn i32 read(ref i32 p){return *p;}fn ref mut i32 id(ref mut i32 p){return p;}fn void f(ref mut i32 q){let ref mut i32 p=id(q);take(p,*q);}",b"",ref wanted26);
    let usize[14] wanted27=[1,2101,0,75,77,2,5,0,76,77,4,0,73,74];
    check_case(27,3,b"module main;import provider as pr;fn void f(ref mut i32 p){pr::take<i32>(p,*p);}",b"module provider;pub fn void take<T>(ref mut T p,i32 n){}",ref wanted27);
    let usize[1] wanted28=[0];
    check_case(28,0,b"module main;import provider as pr;fn void f(ref mut i32 p){pr::take<i32>(p,*p);}",b"module provider;pub fn void take<T>(ref T p,i32 n){}",ref wanted28);
    let usize[14] wanted29=[1,2101,0,216,220,2,5,0,216,217,4,0,203,224];
    check_case(29,3,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(ref mut u8[] p){let ref mut u8[] child=p[..{discard p[0];1}];discard child[0];}",b"",ref wanted29);
    let usize[1] wanted30=[0];
    check_case(30,0,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(ref mut u8[] p){let ref mut u8[] child=p[..p.len];discard child[0];}",b"",ref wanted30);
    let usize[1] wanted31=[0];
    check_case(31,0,b"module main;fn void take(ref mut u8[] p,u8 n){}fn void share(ref u8[] p,u8 n){}fn void edit(ref mut u8[] p){p[0]=2;}fn void size(ref mut u8[] p,usize n){}fn void f(ref mut u8[] p){p[p[1] as usize]=2;}",b"",ref wanted31);
}
