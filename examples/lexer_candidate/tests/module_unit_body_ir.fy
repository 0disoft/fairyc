module tests::module_unit_body_ir;
import module_unit_body_ir_model as ir;
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

@test @effects(console) fn void body_ir_retains_original_blocks_calls_and_cleanup_frames(){
    let p=retained(b"module main;import provider as p;@effects(console)resource R{i32 n;cleanup{discard 2;}}@effects(console)fn void f(){let p::R x=p::make();let R y=R{.n=1};}",b"module provider;@effects(console)pub resource R{i32 n;cleanup{discard 1;}}pub fn R make(){return R{.n=3};}");
    assert(proof::check(ref p).code==0 && ir::count(ref p)==clean_ir::frame_count(ir::cleanup_ir(ref p)));
    var usize frame=1;var usize calls=0;var usize receivers=0;
    while(frame<=ir::count(ref p)){let b=ir::at(ref p,frame);let root=ir::statement_at(b,lookup::statement(b,ir::body(b))-1).value;assert(root.kind==1);if(clean_ir::frame_at(ir::cleanup_ir(ref p),frame).kind==2){receivers+=1;assert(root.start>=ir::start(b) && root.end==ir::end(b));}calls+=ir::call_count(b);frame+=1;}
    assert(calls==1 && receivers==2);var bool provider=false;var bool root=false;var usize i=0;while(i<ir::layout_count(ref p)){if(bytes::equal(ir::layout_name(ref p,i),b"R")){if(ir::layout_at(ref p,i).value.module_index==0){root=true;}else{provider=true;}}i+=1;}assert(root && provider);
}
@test @effects(console) fn void body_ir_preserves_literal_cooking_assignment_and_for_links(){
    let p=retained(b"module main;fn i32 f(){let s=\"a\\n\";var i32 n=0;for(var i32 i=0;i<3;i+=1){n+=i;}return n;}",b"");assert(proof::check(ref p).code==0);let b=ir::at(ref p,1);var bool literal=false;var bool assignment=false;var bool loop=false;var usize i=0;
    while(i<ir::node_count(b)){let n=ir::node_at(b,i);if(n.literal_kind==7){assert(bytes::equal(ir::node_cooked(b,i),b"a\n"));literal=true;}i+=1;}
    i=0;while(i<ir::statement_count(b)){let s=ir::statement_at(b,i).value;if(s.kind==4){assert(bytes::equal(ir::assignment_operator(b,i),b"+="));assignment=true;}if(s.kind==14){let h=ir::statement_at(b,lookup::statement(b,s.second)-1).value;assert(h.kind==15 && lookup::statement(b,h.first)!=0 && lookup::statement(b,h.second)!=0);loop=true;}i+=1;}assert(literal && assignment && loop);
}
@test @effects(console) fn void body_ir_retains_specialized_cleanup_and_partial_values(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}resource Pair{R a;R b;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Pair,i32> f(){return .ok(Pair{.a=make(),.b=make(),.n=number()?});}",b"");assert(proof::check(ref p).code==0 && clean_ir::operand_count(ir::cleanup_ir(ref p))==2);var usize i=1;while(i<=2){let o=clean_ir::operand_at(ir::cleanup_ir(ref p),i);let b=ir::at(ref p,o.frame);let n=ir::node_at(b,lookup::node(b,o.id)-1).value;assert(n.type==o.type && n.start==o.start && n.end==o.end);i+=1;}
}

fn u32 mutated(ref ir::Program p,u32 mode){
    let b=ir::at(p,1);var v::Vec<ir::Node> nodes=v::empty<ir::Node>();var v::Vec<ir::Statement> statements=v::empty<ir::Statement>();var v::Vec<m::Local> locals=v::empty<m::Local>();var v::Vec<m::Invocation> calls=v::empty<m::Invocation>();var v::Vec<m::Argument> arguments=v::empty<m::Argument>();var v::Vec<regions::Scope> regions=v::empty<regions::Scope>();var v::Vec<regions::Exit> resets=v::empty<regions::Exit>();var usize i=0;
    while(i<ir::node_count(b)){var n=ir::node_at(b,i);if(i==0){if(mode==1){n.value.type=clean_ir::type_count(ir::cleanup_ir(p))+1;}if(mode==2){n.syntax.kind=0;}if(mode==3){n.text_end=ir::text_bytes(b)+1;}if(mode==4){n.value.left=999999;n.syntax.left=999999;}}if(mode==5 && i==1){n.value.source_index=ir::node_at(b,0).value.source_index;}assert(v::push<ir::Node>(ref mut nodes,n).is_ok);i+=1;}
    i=0;while(i<ir::statement_count(b)){var row=ir::statement_at(b,i);if(mode==6 && row.source_index==ir::body(b)){row.value.second+=1;}assert(v::push<ir::Statement>(ref mut statements,row).is_ok);i+=1;}
    i=1;while(i<=ir::local_count(b)){var n=ir::local_at(b,i);if(mode==7 && i==1){n.initializer=999999;}assert(v::push<m::Local>(ref mut locals,n).is_ok);i+=1;}
    i=0;while(i<ir::call_count(b)){var c=ir::call_at(b,i);if(mode==8 && i==0){c.expression=999999;}assert(v::push<m::Invocation>(ref mut calls,c).is_ok);i+=1;}
    i=0;while(i<ir::argument_count(b)){var a=ir::argument_at(b,i);if(mode==9 && i==0){a.source_index=999999;}assert(v::push<m::Argument>(ref mut arguments,a).is_ok);i+=1;}
    i=0;while(i<ir::region_count(b)){assert(v::push<regions::Scope>(ref mut regions,ir::region_at(b,i)).is_ok);i+=1;}i=0;while(i<ir::reset_count(b)){assert(v::push<regions::Exit>(ref mut resets,ir::reset_at(b,i)).is_ok);i+=1;}
    let text=switch(bytes::copy(ir::text_view(b))){case .ok(p):{p}case .err(_):{assert(false);return 0;}};
    let changed=ir::assemble_body(if(mode==10){999999}else{ir::body(b)},ir::source_bytes(b),ir::start(b),ir::end(b),nodes,statements,locals,calls,arguments,regions,resets,text);var usize work=if(mode==11){1048577}else{0};return proof::check_body(ref changed,ir::cleanup_ir(p),1,ref mut work).code;
}
@test @effects(console) fn void body_ir_rejects_malformed_owned_body_tables(){let p=retained(b"module main;fn i32 f(){let i32 n=g(1);return n;}fn i32 g(i32 n){return n;}",b"");assert(mutated(ref p,0)==0);var u32 mode=1;while(mode<=10){assert(mutated(ref p,mode)==1001);mode+=1;}assert(mutated(ref p,11)==1002);}
