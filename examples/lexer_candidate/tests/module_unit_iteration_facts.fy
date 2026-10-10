module tests::module_unit_iteration_facts;
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
@test @effects(console) fn void implicit_iteration_loans_have_owned_facts(){
    let usize[8] wanted0=[0,12,0,64,68,64,68,0];
    facts(0,b"module main;fn void f(){var u8[2] data=[1,2];for(let u8 item in data){discard item;}}",b"",ref wanted0,1);
    let usize[8] wanted1=[0,12,0,64,68,64,68,0];
    facts(1,b"module main;fn void f(){var u8[2] data=[1,2];for(let u8 item in data){break;}}",b"",ref wanted1,1);
    let usize[8] wanted2=[0,12,0,60,68,88,89,0];
    facts(2,b"module main;fn void f(){var u8[2] data=[1,2];let ref u8[] p=ref data;for(let u8 item in p){discard item;}}",b"",ref wanted2,1);
    let usize[1] wanted3=[0];
    facts(3,b"module main;fn void f(ref u8[] p){for(let u8 item in p){discard item;}}",b"",ref wanted3,0);
    let usize[16] wanted4=[0,12,1,72,76,79,83,0,0,12,1,72,76,79,83,0];
    facts(4,b"module main;fn void f(){var u8[2] data=[1,2];for(let ref mut u8 item in data){*item=9;}}",b"",ref wanted4,2);
    let usize[8] wanted5=[0,12,1,65,66,69,73,0];
    facts(5,b"module main;fn void f(ref mut u8[] p){for(let ref mut u8 item in p){*item=9;}}",b"",ref wanted5,1);
    let usize[16] wanted6=[0,12,1,65,71,74,78,0,0,12,1,65,71,74,78,0];
    facts(6,b"module main;fn void f(ref mut u8[] p){for(let ref mut u8 item in p[..1]){*item=9;}}",b"",ref wanted6,2);
    let usize[16] wanted7=[0,12,0,68,72,83,87,0,0,12,0,68,72,83,87,0];
    facts(7,b"module main;fn void f(){var u8[2] data=[1,2];for(let ref u8 item in data){discard *item;}}",b"",ref wanted7,2);
    let usize[8] wanted8=[0,12,0,65,70,80,81,0];
    facts(8,b"module main;fn void f(bool c){var i32 x=1;while(c){let ref i32 p=ref x;discard *p;break;}}",b"",ref wanted8,1);
    let usize[8] wanted9=[0,12,0,50,55,91,92,0];
    facts(9,b"module main;fn void f(){var i32 x=1;let ref i32 p=ref x;for(var i32 i=0;i<2;i+=1){discard *p;continue;}}",b"",ref wanted9,1);
    let usize[16] wanted10=[0,12,0,58,59,58,59,0,0,12,0,77,78,77,78,0];
    facts(10,b"module main;fn void f(){var u8[2] x=[1,2];for(let u8 a in x){for(let u8 b in x){discard a;discard b;}}}",b"",ref wanted10,2);
    let usize[8] wanted11=[0,12,0,58,59,58,59,0];
    facts(11,b"module main;fn void f(){var u8[2] x=[1,2];for(let u8 a in x){return;}}",b"",ref wanted11,1);
    let usize[16] wanted12=[0,12,1,72,76,72,76,0,0,12,1,72,76,0,0,0];
    facts(12,b"module main;fn void f(){var u8[2] data=[1,2];for(let ref mut u8 item in data){break;}}",b"",ref wanted12,2);
    let usize[16] wanted13=[0,12,1,74,78,81,85,0,0,12,1,74,78,81,85,0];
    facts(13,b"module main;fn void f(){var i64[2] data=[1,2];for(let ref mut i64 item in data){*item=9;}}",b"",ref wanted13,2);
    let usize[8] wanted14=[0,12,1,71,72,81,85,1];
    facts(14,b"module main;fn ref mut u8 f(ref mut u8[] p){for(let ref mut u8 item in p){return item;}assert(false);}",b"",ref wanted14,1);
    let usize[8] wanted15=[0,35,0,95,103,95,103,0];
    facts(15,b"module main;struct Bag{u8[2] data;}fn void f(){var Bag bag=Bag{.data=[1,2]};for(let u8 item in bag.data){discard item;}}",b"",ref wanted15,1);
    let usize[1] wanted16=[0];
    facts(16,b"module main;struct Bag{u8[2] data;}fn void f(ref Bag bag){for(let u8 item in bag.data){discard item;}}",b"",ref wanted16,0);
    let usize[16] wanted17=[0,35,1,89,97,100,104,0,0,35,1,89,97,100,104,0];
    facts(17,b"module main;struct Bag{u8[2] data;}fn void f(ref mut Bag bag){for(let ref mut u8 item in bag.data){*item=9;}}",b"",ref wanted17,2);
}
