module tests::module_unit_cfg;
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
@test @effects(console) fn void cfg_places_reverse_lexical_drops_after_return_value(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn i32 f(){let R a=make();let R b=make();return 9;}",b"");let frame=declared(ref p,4);let g=lowered(ref p,frame);assert(cfg::check(ref p,ref g).code==0);var usize id=cfg::entry(ref g);var usize count=0;var usize previous=99999;var bool value=false;var bool returned=false;var usize steps=0;
    while(id!=0){assert(steps<256);steps+=1;let n=cfg::at(ref g,id);if(n.kind==1){let e=ir::node_at(ir::at(ref p,frame),n.source-1).value;if(e.kind==1 && bytes::equal(ir::node_text(ir::at(ref p,frame),n.source-1),b"9")){value=true;}}if(n.kind==5){assert(value);let d=clean_ir::drop_at(ir::cleanup_ir(ref p),n.drop-1);let o=clean_ir::operand_at(ir::cleanup_ir(ref p),d.operand);assert(o.kind==1 && o.id<previous && d.exit==2);previous=o.id;count+=1;}if(n.kind==6){returned=true;assert(n.source!=0);}assert(n.kind!=3 && n.kind!=4);id=n.first;}
    assert(count==2 && returned);
}
@test @effects(console) fn void cfg_propagation_error_drops_only_completed_values_in_reverse_order(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}resource Pair{R a;R b;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Pair,i32> f(){return .ok(Pair{.a=make(),.b=make(),.n=number()?});}",b"");let frame=containing(ref p,30);let g=lowered(ref p,frame);assert(cfg::check(ref p,ref g).code==0);var usize id=cfg::entry(ref g);var usize previous=99999;var usize drops=0;var usize calls=0;var bool returned=false;var usize steps=0;
    while(id!=0){assert(steps<256);steps+=1;let n=cfg::at(ref g,id);if(n.kind==1 && ir::node_at(ir::at(ref p,frame),n.source-1).value.kind==6){calls+=1;}if(n.kind==5){let d=clean_ir::drop_at(ir::cleanup_ir(ref p),n.drop-1);let o=clean_ir::operand_at(ir::cleanup_ir(ref p),d.operand);assert(d.exit==6 && o.kind==2 && o.start<previous && calls==3);previous=o.start;drops+=1;}if(n.kind==9){returned=true;}id=if(n.kind==4){n.second}else{n.first};}
    assert(returned && drops==2);id=cfg::entry(ref g);drops=0;steps=0;while(id!=0){assert(steps<256);steps+=1;let n=cfg::at(ref g,id);if(n.kind==5){drops+=1;}id=n.first;}assert(drops==0);
}
@test @effects(console) fn void cfg_has_loop_back_edges_and_break_continue_cleanup(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void f(){while(true){let R inner=make();if(true){continue;}else{break;}}}",b"");let frame=declared(ref p,4);let g=lowered(ref p,frame);var usize i=1;var bool jump=false;var bool broken=false;var bool continued=false;while(i<=cfg::count(ref g)){let n=cfg::at(ref g,i);if(n.kind==7){jump=true;assert(n.first!=i);}if(n.kind==5){let d=clean_ir::drop_at(ir::cleanup_ir(ref p),n.drop-1);if(d.exit==3){broken=true;}if(d.exit==4){continued=true;}}i+=1;}assert(jump && broken && continued && cfg::check(ref p,ref g).code==0);
    let q=retained(b"module main;fn i32 f(){var i32 n=0;for(var i32 i=0;i<3;i+=1){n+=i;}return n;}",b"");let h=lowered(ref q,1);assert(cfg::check(ref q,ref h).code==0);i=1;var usize stores=0;while(i<=cfg::count(ref h)){if(cfg::at(ref h,i).kind==2){stores+=1;}i+=1;}assert(stores==4);
}
@test @effects(console) fn void cfg_short_circuit_skips_right_call_and_declines_defer(){
    let p=retained(b"module main;fn bool side(){return true;}fn bool f(){return false && side();}",b"");let g=lowered(ref p,2);var usize id=cfg::entry(ref g);var usize calls=0;var bool skipped=false;var usize steps=0;while(id!=0){assert(steps<128);steps+=1;let n=cfg::at(ref g,id);if(n.kind==1 && ir::node_at(ir::at(ref p,2),n.source-1).value.kind==6){calls+=1;}if(n.kind==10){skipped=true;}id=if(n.kind==3){n.second}else{n.first};}assert(skipped && calls==0);
    let q=retained(b"module main;fn void done(){}fn void f(){defer done();}",b"");switch(flow::lower(ref q,2)){case .ok(_):{assert(false);}case .err(e):{assert(e.code==1004);}};
}

fn u32 mutated(ref ir::Program p,ref cfg::Graph g,u32 mode){
    var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();var usize i=1;var bool changed=false;
    while(i<=cfg::count(g)){var n=cfg::at(g,i);if(!changed){if(mode==2 && n.first!=0){n.first=cfg::count(g)+1;changed=true;}if(mode==3 && n.kind==1){n.source=999999;changed=true;}if(mode==4 && n.kind==3){n.second=0;changed=true;}if(mode==5 && n.kind==5){n.drop=999999;changed=true;}if(mode==6 && n.kind==5){n.kind=7;n.drop=0;changed=true;}if(mode==7 && n.kind==6){n.first=1;changed=true;}if(mode==9 && n.kind==2){n.source=0;changed=true;}}assert(v::push<cfg::Block>(ref mut blocks,n).is_ok);i+=1;}
    assert(mode==0 || mode==1 || mode==8 || changed);return switch(cfg::assemble(p,cfg::frame(g),if(mode==1){cfg::count(g)+1}else{cfg::entry(g)},blocks,if(mode==8){1048577}else{cfg::work(g)})){case .ok(_):{0}case .err(e):{e.code}};
}
@test @effects(console) fn void cfg_rejects_invalid_successors_values_and_cleanup_coverage(){let p=retained(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn i32 f(){let R a=make();if(true){return 1;}else{return 2;}}",b"");let g=lowered(ref p,declared(ref p,4));assert(mutated(ref p,ref g,0)==0);var u32 mode=1;while(mode<=9){assert(mutated(ref p,ref g,mode)==if(mode==8){1002}else{1001});mode+=1;}}

@test @effects(console) fn void cfg_resource_replacement_evaluates_rhs_before_old_drop_without_loading_target(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}fn R make(){return R{.n=1};}fn void f(){var R r=make();r=make();}",b"");let frame=declared(ref p,4);let g=lowered(ref p,frame);var usize id=cfg::entry(ref g);var usize calls=0;var usize roots=0;var bool replaced=false;var usize steps=0;while(id!=0){assert(steps<128);steps+=1;let n=cfg::at(ref g,id);if(n.kind==1){let e=ir::node_at(ir::at(ref p,frame),n.source-1).value;if(e.kind==6){calls+=1;}if(e.kind==2 && e.binding!=0){roots+=1;}}if(n.kind==5 && clean_ir::drop_at(ir::cleanup_ir(ref p),n.drop-1).exit==5){assert(calls==2);replaced=true;assert(cfg::at(ref g,n.first).kind==2);}id=n.first;}assert(replaced && roots==0);
}

@test @effects(console) fn void cfg_loop_condition_revisits_share_one_runtime_drop_site(){let p=retained(b"module main;resource R{i32 n;cleanup{}}fn Result<R,i32> probe(){return .err(1);}fn void f(){while(probe().is_ok){}}",b"");let g=lowered(ref p,declared(ref p,4));var usize i=1;var usize drops=0;while(i<=cfg::count(ref g)){let n=cfg::at(ref g,i);if(n.kind==5){assert(clean_ir::drop_at(ir::cleanup_ir(ref p),n.drop-1).exit==7);drops+=1;}i+=1;}assert(drops==1 && cfg::check(ref p,ref g).code==0);}
