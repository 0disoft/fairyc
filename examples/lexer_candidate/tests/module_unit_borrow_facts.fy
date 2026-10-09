module tests::module_unit_borrow_facts;
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


@effects(console) fn origins::Plan owned(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);
    if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return owned(root,provider);}};
    let typed=primary(ref sources,ref metadata);
    return switch(origins::build(ref sources,ref metadata,ref typed)){case .ok(p):{p}case .err(p):{discard io::write_u32(p.code);assert(false);owned(root,provider)}};
}
@effects(console) fn void origin_name(ref origins::Plan p,usize frame,usize origin,ref u8[] root,ref u8[] provider){
    let input=origins::owned_inputs(p);if(origin==2){discard io::write("Static");return;}let id=origin-2;let owner=lifetime_inputs::owner(input,id);
    if(owner.kind==11){discard io::write("Static");return;}let f=lifetime_inputs::frame(input,frame);
    if((owner.kind==1 || owner.kind==13) && owner.borrowed){var usize i=0;while(i<f.parameter_count){if(lifetime_inputs::parameter(input,frame,i)==id){discard io::write(if(owner.kind==13){"RegionParam("}else{"Param("});discard io::write_usize(i);discard io::write(")");return;}i+=1;}assert(false);}
    else{discard io::write("Local(");let text=if(f.value.module_index==0){root}else{provider};discard io::write_bytes(text[owner.start..owner.end]);discard io::write(")");}
}
// Frozen Go BorrowFact source coordinates, kinds, last uses and escape flags.
fn void golden(ref origins::Plan p,ref usize[] wanted,usize count){assert(count*8<=wanted.len);var usize total=0;var usize frame=1;while(frame<=origins::count(p)){total+=origins::borrow_fact_count(p,frame);frame+=1;}assert(total==count);var usize i=0;while(i<count){var bool found=false;frame=1;while(frame<=origins::count(p)){let input=origins::owned_inputs(p);let f=lifetime_inputs::frame(input,frame);var usize j=0;while(j<origins::borrow_fact_count(p,frame)){let r=origins::borrow_fact_at(p,frame,j);let at=i*8;let usize mutable=if(r.mutable){1}else{0};let usize escapes=if(r.escapes){1}else{0};if(f.value.module_index==wanted[at] && live::start(lifetime_inputs::liveness(input,frame))==wanted[at+1] && mutable==wanted[at+2] && r.start==wanted[at+3] && r.end==wanted[at+4] && r.last_start==wanted[at+5] && r.last_end==wanted[at+6] && escapes==wanted[at+7]){found=true;}j+=1;}frame+=1;}assert(found);i+=1;}}
@effects(console) fn void facts(usize index,ref u8[] root,ref u8[] provider,ref usize[] wanted,usize count){let p=owned(root,provider);golden(ref p,wanted,count);discard io::write("case ");discard io::write_usize(index);discard io::write("\n");var usize frame=1;while(frame<=origins::count(ref p)){let input=origins::owned_inputs(ref p);let f=lifetime_inputs::frame(input,frame);var usize i=0;while(i<origins::borrow_fact_count(ref p,frame)){let r=origins::borrow_fact_at(ref p,frame,i);assert(r.frame==frame);discard io::write("fact ");discard io::write_usize(f.value.module_index);discard io::write(" ");discard io::write_usize(live::start(lifetime_inputs::liveness(input,frame)));discard io::write(" ");discard io::write_u32(if(r.mutable){1}else{0});discard io::write(" ");discard io::write_usize(r.start);discard io::write(" ");discard io::write_usize(r.end);discard io::write(" ");discard io::write_usize(r.last_start);discard io::write(" ");discard io::write_usize(r.last_end);discard io::write(" ");discard io::write_u32(if(r.escapes){1}else{0});discard io::write(" ");origin_name(ref p,frame,r.origin,root,provider);discard io::write("\n");i+=1;}frame+=1;}}
@test @effects(console) fn void owned_retained_borrow_facts_report_last_use_and_escape(){
    let usize[8] wanted0=[0,12,0,50,55,65,66,0];
    facts(0,b"module main;fn void f(){var i32 x=1;let ref i32 p=ref x;discard *p;}",b"",ref wanted0,1);
    let usize[8] wanted1=[0,12,0,50,55,0,0,0];
    facts(1,b"module main;fn void f(){var i32 x=1;let ref i32 p=ref x;}",b"",ref wanted1,1);
    let usize[1] wanted2=[0];
    facts(2,b"module main;fn void read(ref i32 p){}fn void f(){var i32 x=1;read(ref x);}",b"",ref wanted2,0);
    let usize[8] wanted3=[0,12,0,50,55,81,82,0];
    facts(3,b"module main;fn void f(){var i32 x=1;let ref i32 p=ref x;let ref i32 q=p;discard *q;}",b"",ref wanted3,1);
    let usize[16] wanted4=[0,12,0,62,67,77,78,0,0,12,0,81,86,96,97,0];
    facts(4,b"module main;fn void f(){var i32 x=1;var i32 y=2;var ref i32 p=ref x;discard *p;p=ref y;discard *p;}",b"",ref wanted4,2);
    let usize[8] wanted5=[0,12,0,64,69,0,0,0];
    facts(5,b"module main;fn void f(){var i32 x=1;var Option<ref i32> p=.some(ref x);p=.none;discard p;}",b"",ref wanted5,1);
    let usize[8] wanted6=[0,12,0,64,69,105,106,0];
    facts(6,b"module main;fn void f(){var i32 x=1;let Option<ref i32> p=.some(ref x);switch(p){case .some(q):{discard *q;}case .none:{}};}",b"",ref wanted6,1);
    let usize[16] wanted7=[0,12,0,54,62,54,62,1,0,64,0,100,105,132,133,0];
    facts(7,b"module main;fn Option<ref i32> wrap(ref i32 p){return .some(p);}fn void f(){var i32 x=1;switch(wrap(ref x)){case .some(q):{discard *q;}case .none:{}};}",b"",ref wanted7,2);
    let usize[8] wanted8=[0,12,1,54,63,78,79,0];
    facts(8,b"module main;fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;*p=2;discard *p;}",b"",ref wanted8,1);
    let usize[16] wanted9=[0,12,1,61,70,105,106,0,0,12,1,90,96,105,106,0];
    facts(9,b"module main;fn void f(){var u8[2] x=[1,2];let ref mut u8[] p=ref mut x;let ref mut u8[] q=p[..1];discard q[0];}",b"",ref wanted9,2);
    let usize[8] wanted10=[0,12,1,61,75,84,85,0];
    facts(10,b"module main;fn void f(){var u8[2] x=[1,2];let ref mut u8[] p=ref mut x[..1];discard p[0];}",b"",ref wanted10,1);
    let usize[8] wanted11=[0,12,0,43,44,43,44,1];
    facts(11,b"module main;fn ref i32 f(ref i32 p){return p;}",b"",ref wanted11,1);
    let usize[8] wanted12=[0,12,1,51,52,51,52,1];
    facts(12,b"module main;fn ref mut i32 f(ref mut i32 p){return p;}",b"",ref wanted12,1);
    let usize[8] wanted13=[0,12,0,51,59,51,59,1];
    facts(13,b"module main;fn Option<ref i32> f(ref i32 p){return .some(p);}",b"",ref wanted13,1);
    let usize[1] wanted14=[0];
    facts(14,b"module main;fn Option<ref i32> f(){return .none;}",b"",ref wanted14,0);
    let usize[8] wanted15=[0,12,1,53,59,53,59,1];
    facts(15,b"module main;fn ref mut u8[] f(ref mut u8[] p){return p[..1];}",b"",ref wanted15,1);
    let usize[8] wanted16=[0,12,1,65,71,79,80,1];
    facts(16,b"module main;fn ref mut u8[] f(ref mut u8[] p){let ref mut u8[] q=p[..1];return q;}",b"",ref wanted16,1);
    let usize[8] wanted17=[0,12,0,39,45,54,55,0];
    facts(17,b"module main;fn void f(){let ref u8[] p=b\"abc\";discard p[0];}",b"",ref wanted17,1);
    let usize[8] wanted18=[0,12,0,35,41,35,41,1];
    facts(18,b"module main;fn ref u8[] f(){return b\"abc\";}",b"",ref wanted18,1);
    let usize[8] wanted19=[0,12,0,43,49,57,58,1];
    facts(19,b"module main;fn ref u8[] f(){let ref u8[] p=b\"abc\";return p;}",b"",ref wanted19,1);
    let usize[8] wanted20=[0,12,0,53,58,66,67,0];
    facts(20,b"module main;fn ref i32 f(){var i32 x=1;let ref i32 p=ref x;return p;}",b"",ref wanted20,1);
    let usize[16] wanted21=[0,12,0,68,73,99,100,0,0,12,0,82,87,99,100,0];
    facts(21,b"module main;fn void f(bool c){var i32 x=1;var i32 y=2;var ref i32 p=ref x;if(c){p=ref y;};discard *p;}",b"",ref wanted21,2);
    let usize[8] wanted22=[0,12,0,56,61,80,81,0];
    facts(22,b"module main;fn void f(bool c){var i32 x=1;let ref i32 p=ref x;while(c){discard *p;break;}}",b"",ref wanted22,1);
    let usize[16] wanted23=[0,12,0,44,45,44,45,1,0,47,0,88,93,104,105,0];
    facts(23,b"module main;fn ref i32 id(ref i32 p){return p;}fn void f(){var i32 x=1;let ref i32 p=id(ref x);discard *p;}",b"",ref wanted23,2);
    let usize[24] wanted24=[0,12,0,44,45,44,45,1,0,47,1,89,98,128,129,0,0,47,0,116,117,128,129,0];
    facts(24,b"module main;fn ref i32 id(ref i32 p){return p;}fn void f(){var i32 x=1;let ref mut i32 p=ref mut x;let ref i32 q=id(p);discard *q;}",b"",ref wanted24,3);
    let usize[16] wanted25=[0,30,1,80,89,121,122,0,0,30,0,104,111,121,122,0];
    facts(25,b"module main;struct Box{i32 x;}fn void f(){var Box b=Box{.x=1};let ref mut Box p=ref mut b;let ref i32 q=ref p.x;discard *q;}",b"",ref wanted25,2);
    let usize[16] wanted26=[0,34,1,79,94,103,104,0,1,16,1,65,71,65,71,1];
    facts(26,b"module main;import provider as pr;fn void f(ref mut u8[] p){let ref mut u8[] q=pr::tail<u8>(p);discard q[0];}",b"module provider;pub fn ref mut T[] tail<T>(ref mut T[] p){return p[..1];}",ref wanted26,2);
}
