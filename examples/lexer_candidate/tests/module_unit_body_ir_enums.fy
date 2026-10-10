module tests::module_unit_body_ir_enums;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_verify as proof;
import module_unit_body_ir_enums as enums;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_moves as moves;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import module_types as types;
import module_unit_interfaces as interfaces;
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


@test @effects(console) fn void owned_enums_keep_nested_inert_payloads_provider_coordinates_and_tagged_fields(){
    let p=retained(b"module main;import provider as p;fn i32 examine(p::Outer value){return switch(value){case .Empty:{0}case .Data{.code=code,.result=result,.wrapped=wrapped,.item=item}:{discard code;discard result;discard wrapped;discard item;42}};}fn i32 main(){let p::Inner empty=.Empty;let p::Outer value=.Data{.item=.Data{.n=42},.wrapped=.some(.Data{.n=5}),.result=.ok(7),.code=null};discard empty;assert(examine(value)==42);return 0;}",b"module provider;pub enum Never{}pub enum Inner{Empty;Data{i32 n;};}pub enum Outer{Empty;Data{Inner item;Option<Inner> wrapped;Result<i32,bool> result;ptr i32 code;};}");assert(proof::check(ref p).code==0);var usize shapes=0;var usize i=0;while(i<ir::layout_count(ref p)){let shape=ir::layout_at(ref p,i).value;if(clean_ir::type_at(ir::cleanup_ir(ref p),shape.type).kind==4){assert(shape.module_index==1);shapes+=1;}i+=1;}assert(shapes==3);
}

fn clean_ir::Program copied_cleanup(ref clean_ir::Program p){
    var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<clean_ir::Frame> frames=v::empty<clean_ir::Frame>();var v::Vec<clean_ir::Glue> glues=v::empty<clean_ir::Glue>();var v::Vec<clean_ir::Field> fields=v::empty<clean_ir::Field>();var v::Vec<clean_ir::Operand> operands=v::empty<clean_ir::Operand>();var v::Vec<clean_ir::Drop> drops=v::empty<clean_ir::Drop>();var usize i=1;
    while(i<=clean_ir::type_count(p)){assert(v::push<types::Type>(ref mut graph,clean_ir::type_at(p,i)).is_ok && v::push<clean_ir::Glue>(ref mut glues,clean_ir::glue_at(p,i)).is_ok);i+=1;}i=1;while(i<=clean_ir::frame_count(p)){assert(v::push<clean_ir::Frame>(ref mut frames,clean_ir::frame_at(p,i)).is_ok);i+=1;}i=0;while(i<clean_ir::field_count(p)){assert(v::push<clean_ir::Field>(ref mut fields,clean_ir::field_at(p,i)).is_ok);i+=1;}i=1;while(i<=clean_ir::operand_count(p)){assert(v::push<clean_ir::Operand>(ref mut operands,clean_ir::operand_at(p,i)).is_ok);i+=1;}i=0;while(i<clean_ir::drop_count(p)){assert(v::push<clean_ir::Drop>(ref mut drops,clean_ir::drop_at(p,i)).is_ok);i+=1;}
    return switch(clean_ir::assemble(graph,frames,glues,fields,operands,drops,clean_ir::work(p))){case .ok(p):{p}case .err(_):{assert(false);copied_cleanup(p)}};
}
fn ir::Body copied_body(ref ir::Body b,u32 mode,usize enum_type,usize scalar){
    var v::Vec<ir::Node> nodes=v::empty<ir::Node>();var v::Vec<ir::Statement> statements=v::empty<ir::Statement>();var v::Vec<m::Local> locals=v::empty<m::Local>();var usize i=0;
    while(i<ir::node_count(b)){var n=ir::node_at(b,i);if(mode==7 && n.value.kind==10 && n.value.type==enum_type){n.value.field.owner=scalar;}if(mode==8 && n.value.kind==11 && n.value.field.id!=0){n.value.field.source_index+=1;}if(mode==9 && n.value.kind==1 && n.literal_kind==5){n.value.type=scalar;}assert(v::push<ir::Node>(ref mut nodes,n).is_ok);i+=1;}i=0;while(i<ir::statement_count(b)){assert(v::push<ir::Statement>(ref mut statements,ir::statement_at(b,i)).is_ok);i+=1;}i=1;while(i<=ir::local_count(b)){assert(v::push<m::Local>(ref mut locals,ir::local_at(b,i)).is_ok);i+=1;}
    assert(ir::call_count(b)==0 && ir::region_count(b)==0 && ir::reset_count(b)==0 && ir::placement_count(b)==0);let text=switch(bytes::copy(ir::text_view(b))){case .ok(t):{t}case .err(_):{assert(false);return copied_body(b,mode,enum_type,scalar);}};
    return ir::assemble_body(ir::body(b),ir::source_bytes(b),ir::start(b),ir::end(b),nodes,statements,locals,v::empty<m::Invocation>(),v::empty<m::Argument>(),v::empty<regions::Scope>(),v::empty<regions::Exit>(),text);
}
fn u32 mutated(ref ir::Program p,u32 mode){
    var usize shape=0;var usize scalar=0;var usize void_type=0;var usize i=0;while(i<ir::layout_count(p)){if(clean_ir::type_at(ir::cleanup_ir(p),ir::layout_at(p,i).value.type).kind==4){shape=i+1;}i+=1;}i=1;while(i<=clean_ir::type_count(ir::cleanup_ir(p))){let t=clean_ir::type_at(ir::cleanup_ir(p),i);if(t.kind==1 && t.scalar==6){scalar=i;}if(t.kind==1 && t.scalar==7){void_type=i;}i+=1;}assert(shape!=0 && scalar!=0 && void_type!=0);let s=ir::layout_at(p,shape-1).value;var usize variant=0;i=s.field_start;while(i<s.field_start+s.field_count){let f=ir::field_at(p,i);if(f.kind==3 && f.payload_count==2){variant=i;}i+=1;}assert(variant!=0);let payload=ir::field_at(p,variant).payload_start;
    var v::Vec<ir::Body> bodies=v::empty<ir::Body>();var v::Vec<ir::Layout> layouts=v::empty<ir::Layout>();var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();var v::Vec<usize> parameters=v::empty<usize>();i=1;
    while(i<=ir::count(p)){assert(v::push<ir::Body>(ref mut bodies,copied_body(ir::at(p,i),mode,s.type,scalar)).is_ok);i+=1;}i=0;while(i<ir::layout_count(p)){assert(v::push<ir::Layout>(ref mut layouts,ir::layout_at(p,i)).is_ok);i+=1;}if(mode==10){assert(v::push<ir::Layout>(ref mut layouts,ir::layout_at(p,shape-1)).is_ok);}i=0;
    while(i<ir::field_count(p)){var f=ir::field_at(p,i);if(i==variant && mode==1){f.payload_start+=1;}if(i==variant && mode==2){f.payload_count+=1;}if(i==payload && mode==3){f.kind=0;}if(i==payload && mode==4){f.type=s.type;}if(i==payload && mode==5){f.type=void_type;}if(i==payload+1 && mode==6){let original=ir::field_at(p,payload);f.name_start=original.name_start;f.name_end=original.name_end;}assert(v::push<interfaces::Field>(ref mut fields,f).is_ok);i+=1;}i=0;while(i<ir::callable_count(p)){assert(v::push<interfaces::Callable>(ref mut callables,ir::callable_at(p,i)).is_ok);i+=1;}i=0;while(i<ir::parameter_count(p)){assert(v::push<usize>(ref mut parameters,ir::parameter_at(p,i)).is_ok);i+=1;}let names=switch(bytes::copy(ir::names_view(p))){case .ok(n):{n}case .err(_):{assert(false);return 0;}};
    return switch(ir::assemble(copied_cleanup(ir::cleanup_ir(p)),bodies,layouts,fields,callables,parameters,names,ir::work(p))){case .ok(_):{0}case .err(e):{e.code}};
}
@test @effects(console) fn void owned_enums_reject_forged_ranges_cycles_names_and_constructor_fields(){
    let p=retained(b"module main;enum E{Empty;Data{i32 a;bool b;};}fn i32 main(){let E empty=.Empty;let E value=.Data{.a=7,.b=true};discard empty;discard value;return 0;}",b"");var usize total=0;assert(enums::check(ref p,ref mut total).code==0 && total>0 && total<1048576);var usize remaining=0;while(remaining<total){var usize work=1048576-remaining;assert(enums::check(ref p,ref mut work).code==1002);remaining+=1;}assert(mutated(ref p,0)==0);var u32 mode=1;while(mode<=10){assert(mutated(ref p,mode)==1001);mode+=1;}
}
