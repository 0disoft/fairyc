module tests::module_unit_cfg_defer;
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


@effects(console) fn cfg::Graph lowered(ref ir::Program p,usize frame){return switch(flow::lower(p,frame)){case .ok(g):{g}case .err(e):{discard io::write_u32(e.code);discard io::write(" cfg at ");discard io::write_usize(e.start);discard io::write(".. ");discard io::write_usize(e.end);discard io::write("\n");assert(false);lowered(p,frame)}};}
fn usize declared(ref ir::Program p,usize declaration){var usize i=1;while(i<=ir::count(p)){let f=clean_ir::frame_at(ir::cleanup_ir(p),i);if(f.kind==3 && f.declaration==declaration){return i;}i+=1;}return 0;}
fn usize containing(ref ir::Program p,u32 kind){var usize f=1;while(f<=ir::count(p)){let b=ir::at(p,f);var usize i=0;while(i<ir::node_count(b)){if(ir::node_at(b,i).value.kind==kind){return f;}i+=1;}f+=1;}return 0;}

fn usize with_defer(ref ir::Program p){var usize f=1;while(f<=ir::count(p)){let b=ir::at(p,f);var usize i=0;while(i<ir::statement_count(b)){if(ir::statement_at(b,i).value.kind==13){return f;}i+=1;}f+=1;}return 0;}
fn bool literal(ref ir::Program p,usize frame,usize source,ref u8[] text){return source!=0 && bytes::equal(ir::node_text(ir::at(p,frame),source-1),text);}
@test @effects(console) fn void deferred_capture_precedes_mutation_and_invocation_precedes_owner_drop(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn i64 side(i64 n){return n;}fn void done(i32 a,i64 b){}fn i32 f(){let R r=make();var i32 x=1;defer done(x,side(2));x=3;return 9;}",b"");let frame=with_defer(ref p);let g=lowered(ref p,frame);assert(cfg::deferred_count(ref g)==1 && cfg::capture_count(ref g)==2);var usize id=cfg::entry(ref g);var usize captured=0;var bool mutated=false;var bool result=false;var usize invoked=0;var usize dropped=0;var usize steps=0;
    while(id!=0){assert(steps<256);steps+=1;let n=cfg::at(ref g,id);if(n.kind==13){assert(!mutated && n.target==captured+1);captured+=1;}if(n.kind==2 && literal(ref p,frame,n.source,b"3")){assert(captured==2);mutated=true;}if(n.kind==1 && literal(ref p,frame,n.source,b"9")){result=true;}if(n.kind==14){assert(mutated && result && captured==2);invoked+=1;}if(n.kind==5){assert(invoked==1);dropped+=1;}id=n.first;}assert(invoked==1 && dropped==1 && captured==2);
    let first=cfg::capture_at(ref g,1);let second=cfg::capture_at(ref g,2);assert(clean_ir::type_at(ir::cleanup_ir(ref p),first.type).scalar==1 && clean_ir::type_at(ir::cleanup_ir(ref p),second.type).scalar==2);
}
@test @effects(console) fn void deferred_nested_fallthrough_uses_reverse_lexical_registration(){
    let p=retained(b"module main;fn void done(i32 n){}fn void f(){defer done(1);{defer done(2);defer done(3);}defer done(4);}",b"");let frame=with_defer(ref p);let g=lowered(ref p,frame);let usize[4] expected=[3,2,4,1];var usize index=0;var usize id=cfg::entry(ref g);var usize steps=0;while(id!=0){assert(steps<256);steps+=1;let n=cfg::at(ref g,id);if(n.kind==14){let d=cfg::deferred_at(ref g,n.target);let c=cfg::capture_at(ref g,d.capture_start+1);assert(index<expected.len);let text=ir::node_text(ir::at(ref p,frame),c.source-1);assert(text.len==1 && (text[0] as usize)==48+expected[index]);index+=1;}id=n.first;}assert(index==4);
}
@test @effects(console) fn void deferred_propagation_during_registration_excludes_unregistered_call(){
    let p=retained(b"module main;fn void done(i32 n){}fn Result<i32,i32> number(){return .err(1);}fn Result<i32,i32> f(){defer done(1);defer done(number()?);return .ok(3);}",b"");let frame=with_defer(ref p);let g=lowered(ref p,frame);var usize id=cfg::entry(ref g);var usize invoked=0;var usize captured=0;var bool returned=false;var usize steps=0;while(id!=0){assert(steps<256);steps+=1;let n=cfg::at(ref g,id);if(n.kind==13){captured+=1;}if(n.kind==14){assert(n.target==1);invoked+=1;}if(n.kind==9){returned=true;}id=if(n.kind==4){n.second}else{n.first};}assert(returned && captured==1 && invoked==1);
    id=cfg::entry(ref g);invoked=0;steps=0;while(id!=0){assert(steps<256);steps+=1;let n=cfg::at(ref g,id);if(n.kind==14){assert(n.target==2-invoked);invoked+=1;}id=n.first;}assert(invoked==2);
}
@test @effects(console) fn void deferred_loop_exit_runs_loop_suffix_and_preserves_outer_registration(){
    let p=retained(b"module main;fn void done(i32 n){}fn void f(bool c){defer done(1);while(c){defer done(2);if(c){continue;}else{break;}}}",b"");let frame=with_defer(ref p);let g=lowered(ref p,frame);var usize id=cfg::entry(ref g);var usize outer=0;var usize inner=0;var usize branches=0;var usize steps=0;while(id!=0){assert(steps<256);steps+=1;let n=cfg::at(ref g,id);if(n.kind==14){if(n.target==1){outer+=1;}else{assert(n.target==2);inner+=1;}}if(n.kind==3){branches+=1;id=if(branches==1){n.first}else{n.second};}else{id=n.first;}}assert(inner==1 && outer==1);
    id=cfg::entry(ref g);outer=0;inner=0;branches=0;steps=0;while(id!=0 && steps<128){steps+=1;let n=cfg::at(ref g,id);if(n.kind==14){assert(n.target==2);inner+=1;}if(n.kind==3){branches+=1;if(branches==3){break;}id=n.first;}else{id=n.first;}}assert(inner==1 && outer==0 && branches==3);
}

fn u32 malformed(ref ir::Program p,ref cfg::Graph g,u32 mode){var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();var v::Vec<cfg::Deferred> sites=v::empty<cfg::Deferred>();var v::Vec<cfg::Capture> captures=v::empty<cfg::Capture>();var usize i=1;var usize capture_block=0;
    while(i<=cfg::count(g)){var n=cfg::at(g,i);if(n.kind==13){capture_block=i;if(mode==5){n.source=999999;}if(mode==8){n.kind=7;n.source=0;n.target=0;}}if(n.kind==14 && mode==6){n.target=999999;}assert(v::push<cfg::Block>(ref mut blocks,n).is_ok);i+=1;}if(mode==7){assert(capture_block!=0 && v::push<cfg::Block>(ref mut blocks,cfg::at(g,capture_block)).is_ok);}
    i=1;while(i<=cfg::deferred_count(g)){var d=cfg::deferred_at(g,i);if(mode==3){d.invocation=0;}if(mode==4){d.capture_start=1;}if(mode==9){d.invocation=1;}assert(v::push<cfg::Deferred>(ref mut sites,d).is_ok);i+=1;}
    i=1;while(i<=cfg::capture_count(g)){var c=cfg::capture_at(g,i);if(mode==1){c.source=999999;}if(mode==2){c.type=0;}assert(v::push<cfg::Capture>(ref mut captures,c).is_ok);i+=1;}
    return switch(cfg::assemble_deferred(p,cfg::frame(g),cfg::entry(g),blocks,sites,captures,cfg::work(g))){case .ok(_):{0}case .err(e):{e.code}};
}
@test @effects(console) fn void deferred_handoff_rejects_invalid_targets_captures_and_missing_freezes(){let p=retained(b"module main;fn i32 side(i32 n){return n;}fn void done(i32 n){}fn void f(){defer done(side(1));}",b"");let g=lowered(ref p,with_defer(ref p));assert(malformed(ref p,ref g,0)==0);var u32 mode=1;while(mode<=9){assert(malformed(ref p,ref g,mode)==1001);mode+=1;}}
