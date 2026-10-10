module tests::module_unit_c_enums;
import module_unit_c_emit as emit;
import module_unit_body_ir_model as ir;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_moves as moves;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import module_types as types;
import module_unit_value_model as m;
import module_unit_region_model as regions;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
@effects(console) fn ir::Program retained(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,v::len<syntax::Parsed>(ref sources))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" before lower\n");assert(false);return retained(root,provider);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{discard io::write_u32(e.code);discard io::write(" at ");discard io::write_usize(e.start);discard io::write(".. ");discard io::write_usize(e.end);discard io::write(" lower\n");assert(false);retained(root,provider)}};
}
fn usize entry(ref ir::Program p){var usize found=0;var usize i=1;while(i<=ir::count(p)){let f=clean_ir::frame_at(ir::cleanup_ir(p),i);let type=clean_ir::type_at(ir::cleanup_ir(p),f.result);if(f.module_index==0 && f.kind==3 && type.kind==1 && type.scalar==1){var usize j=1;var usize params=0;let b=ir::at(p,i);while(j<=ir::local_count(b)){if(ir::local_at(b,j).kind==1){params+=1;}j+=1;}if(params==0){assert(found==0);found=i;}}i+=1;}return found;}
@effects(console) fn void output(ref ir::Program p,usize frame){var bytes::Buffer c=bytes::empty();let e=emit::render(p,frame,ref mut c);if(e.code!=0){discard io::write_u32(e.code);discard io::write(" C at ");discard io::write_usize(e.start);discard io::write("\n");}assert(e.code==0);assert(bytes::view(ref c).len>100);assert(io::write_bytes(bytes::view(ref c)).is_ok);}




fn usize named_type(ref ir::Program p,usize module_index,ref u8[] name){var usize found=0;var usize i=0;while(i<ir::layout_count(p)){let layout=ir::layout_at(p,i);if(layout.value.module_index==module_index && bytes::equal(ir::layout_name(p,i),name)){assert(found==0);found=layout.value.type;}i+=1;}assert(found!=0);return found;}
fn usize named_member(ref ir::Program p,usize type,ref u8[] name){var usize i=0;while(i<ir::layout_count(p)){let s=ir::layout_at(p,i).value;if(s.type==type){var usize j=0;while(j<s.field_count){let id=s.field_start+j;if(bytes::equal(ir::field_name(p,id),name)){return id+1;}j+=1;}}i+=1;}assert(false);return 0;}
fn usize named_payload(ref ir::Program p,usize variant,ref u8[] name){let f=ir::field_at(p,variant-1);var usize i=0;while(i<f.payload_count){let id=f.payload_start+i;if(bytes::equal(ir::field_name(p,id),name)){return id+1;}i+=1;}assert(false);return 0;}
fn void text(ref mut bytes::Buffer out,ref u8[] value){assert(bytes::append(out,value).is_ok);}
fn void number(ref mut bytes::Buffer out,usize value){assert(bytes::append_usize(out,value).is_ok);}
fn void field(ref mut bytes::Buffer out,usize id){text(out,b"fy_f_");number(out,id);}
fn void variant(ref mut bytes::Buffer out,usize id){text(out,b"fy_variant_");number(out,id);}
fn void selected(ref mut bytes::Buffer out,usize member,usize child){text(out,b"v->fy_payload.");variant(out,member);text(out,b".");field(out,child);}
fn void begin_probe(ref mut bytes::Buffer out,ref u8[] name,usize type){text(out,b"\n/* private enum representation test companion, not a public ABI */\nint32_t ");text(out,name);text(out,b"(const void *raw){const struct fy_unit_t_");number(out,type);text(out,b" *v=(const struct fy_unit_t_");number(out,type);text(out,b" *)raw;if(v->fy_tag==0)return 0;if(v->fy_tag!=1)return -1;return ");}
@effects(console) fn void with_probe(ref ir::Program p,ref u8[] probe){var bytes::Buffer c=bytes::empty();assert(emit::render(p,entry(p),ref mut c).code==0);assert(bytes::append(ref mut c,probe).is_ok);assert(io::write_bytes(bytes::view(ref c)).is_ok);}

@test @effects(console) fn void enum_c_transports_provider_nested_and_copy_tagged_payloads(){
    let p=retained(b"module main;import provider as p;@link_name(\"probe_packet\")unsafe extern C fn i32 probe(ptr void value);fn p::Packet pass(p::Packet value){return value;}fn i32 main(){let p::Packet empty=.Empty;unsafe{assert(probe(ref empty as ptr void)==0);}let p::Packet original=p::make();let p::Packet copy=pass(original);let Result<p::Packet,bool> result=.ok(copy);let Option<p::Packet> optional=.some(pass(original));unsafe{assert(probe(ref original as ptr void)==42 && probe(ref copy as ptr void)==42);}discard result;discard optional;return 0;}",b"module provider;pub enum Never{}pub struct Flat{i32 n;bool valid;}pub enum Leaf{Empty;Data{i64 n;};}pub enum Packet{Empty;Full{Leaf nested;Option<Leaf> optional;Result<i32,bool> result;Flat pair;ptr i32 pointer;};}pub fn Packet make(){return .Full{.nested=.Data{.n=42},.optional=.some(.Data{.n=9}),.result=.err(false),.pair=Flat{.n=7,.valid=true},.pointer=null};}");
    let packet=named_type(ref p,1,b"Packet");let leaf=named_type(ref p,1,b"Leaf");let flat=named_type(ref p,1,b"Flat");let full=named_member(ref p,packet,b"Full");let data=named_member(ref p,leaf,b"Data");let leaf_n=named_payload(ref p,data,b"n");let nested=named_payload(ref p,full,b"nested");let optional=named_payload(ref p,full,b"optional");let result=named_payload(ref p,full,b"result");let pair=named_payload(ref p,full,b"pair");let pointer=named_payload(ref p,full,b"pointer");var bytes::Buffer probe=bytes::empty();begin_probe(ref mut probe,b"probe_packet",packet);
    selected(ref mut probe,full,nested);text(ref mut probe,b".fy_tag==1 && ");selected(ref mut probe,full,nested);text(ref mut probe,b".fy_payload.");variant(ref mut probe,data);text(ref mut probe,b".");field(ref mut probe,leaf_n);text(ref mut probe,b"==42 && ");
    selected(ref mut probe,full,optional);text(ref mut probe,b".fy_tag==1 && ");selected(ref mut probe,full,optional);text(ref mut probe,b".fy_payload.fy_ok.fy_tag==1 && ");selected(ref mut probe,full,optional);text(ref mut probe,b".fy_payload.fy_ok.fy_payload.");variant(ref mut probe,data);text(ref mut probe,b".");field(ref mut probe,leaf_n);text(ref mut probe,b"==9 && ");
    selected(ref mut probe,full,result);text(ref mut probe,b".fy_tag==0 && ");selected(ref mut probe,full,result);text(ref mut probe,b".fy_payload.fy_err==0 && ");selected(ref mut probe,full,pair);text(ref mut probe,b".");field(ref mut probe,named_member(ref p,flat,b"n"));text(ref mut probe,b"==7 && ");selected(ref mut probe,full,pair);text(ref mut probe,b".");field(ref mut probe,named_member(ref p,flat,b"valid"));text(ref mut probe,b"==1 && ");selected(ref mut probe,full,pointer);text(ref mut probe,b"==0 ? 42 : -1;}\n");with_probe(ref p,bytes::view(ref probe));
}
@test @effects(console) fn void enum_c_orders_payload_effects_copies_values_and_defers_pointer_cycles(){
    let p=retained(b"module main;enum Envelope{Empty;Node{i32 a;i32 b;ptr Link next;};}struct Link{Envelope value;}@link_name(\"probe_envelope\")unsafe extern C fn i32 probe(ptr void value);@link_name(\"observe_cleanup\")@effects(console)unsafe extern C fn void trace(i32 n);@effects(console)fn i32 next(ref mut i32 calls){*calls+=1;unsafe{trace(*calls);}return *calls;}fn Envelope pass(Envelope value){return value;}@effects(console)fn i32 main(){var i32 calls=0;var Envelope original=.Node{.b=next(ref mut calls),.a=next(ref mut calls),.next=null};let Envelope copy=pass(original);original=.Empty;assert(calls==2);unsafe{assert(probe(ref copy as ptr void)==42 && probe(ref original as ptr void)==0);trace(calls);}return 0;}",b"");
    let envelope=named_type(ref p,0,b"Envelope");let link=named_type(ref p,0,b"Link");let node=named_member(ref p,envelope,b"Node");var bytes::Buffer probe=bytes::empty();begin_probe(ref mut probe,b"probe_envelope",envelope);selected(ref mut probe,node,named_payload(ref p,node,b"a"));text(ref mut probe,b"==2 && ");selected(ref mut probe,node,named_payload(ref p,node,b"b"));text(ref mut probe,b"==1 && ");selected(ref mut probe,node,named_payload(ref p,node,b"next"));text(ref mut probe,b"==0 && sizeof(struct fy_unit_t_");number(ref mut probe,link);text(ref mut probe,b")==sizeof(struct fy_unit_t_");number(ref mut probe,envelope);text(ref mut probe,b") ? 42 : -1;}\n");with_probe(ref p,bytes::view(ref probe));
}
@test @effects(console) fn void enum_c_keeps_foreign_enum_abi_and_switch_execution_pending(){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;enum E{A;}@link_name(\"pending_enum_abi\")unsafe extern C fn void use(E n);")).is_ok);assert(switch(signatures::build(ref sources)){case .ok(_):{false}case .err(e):{e.code==5102}});
    let p=retained(b"module main;enum E{A;}fn i32 main(){let E value=.A;return switch(value){case .A:{0}};}",b"");var bytes::Buffer c=switch(bytes::copy(b"sentinel")){case .ok(c):{c}case .err(_):{assert(false);return;}};assert(emit::render(ref p,entry(ref p),ref mut c).code==1004);assert(bytes::equal(bytes::view(ref c),b"sentinel"));
}
