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
@test @effects(console) fn void cfg_short_circuit_skips_right_call_and_accepts_defer(){
    let p=retained(b"module main;fn bool side(){return true;}fn bool f(){return false && side();}",b"");let g=lowered(ref p,2);var usize id=cfg::entry(ref g);var usize calls=0;var bool skipped=false;var usize steps=0;while(id!=0){assert(steps<128);steps+=1;let n=cfg::at(ref g,id);if(n.kind==1 && ir::node_at(ir::at(ref p,2),n.source-1).value.kind==6){calls+=1;}if(n.kind==10){skipped=true;}id=if(n.kind==3){n.second}else{n.first};}assert(skipped && calls==0);
    let q=retained(b"module main;fn void done(){}fn void f(){defer done();}",b"");let h=lowered(ref q,2);assert(cfg::deferred_count(ref h)==1 && cfg::capture_count(ref h)==0 && cfg::check(ref q,ref h).code==0);
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

@test @effects(console) fn void cfg_switches_freeze_one_subject_and_guard_all_provider_and_nested_payloads(){
    let p=retained(b"module main;import provider as p;fn p::E choose(){return .Data{.n=7,.flag=true};}fn i32 f(){return switch(choose()){case .Data{.flag=flag,.n=n}:{if(flag){n}else{0}}case .Empty:{1}};}fn i32 nested(Option<Result<i32,bool>> value){return switch(value){case .some(value):{switch(value){case .err(_):{3}case .ok(value):{value}}}case .none:{0}};}",b"module provider;pub enum E{Empty;Data{i32 n;bool flag;};}");
    var usize frame=1;var usize cases=0;var usize seeds=0;while(frame<=ir::count(ref p)){let g=lowered(ref p,frame);let b=ir::at(ref p,frame);var usize i=1;var usize calls=0;while(i<=cfg::count(ref g)){let n=cfg::at(ref g,i);if(n.kind==1 && ir::node_at(b,n.source-1).value.kind==6){calls+=1;}if(n.kind==18){cases+=1;let seed=cfg::at(ref g,n.first);assert(seed.kind==19 && seed.source==n.source && seed.target==n.target);}if(n.kind==19){seeds+=1;}i+=1;}if(frame==2){assert(calls==1);}frame+=1;}assert(cases==6 && seeds==6);
}

@test @effects(console) fn void cfg_switches_keep_active_owned_payload_cleanup_and_arm_defer_suffix(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}fn void done(i32 n){}fn void f(Result<R,i32> value){switch(value){case .ok(r):{defer done(1);discard r;}case .err(_):{defer done(2);}};}fn R take(Result<R,i32> value){return switch(value){case .ok(r):{r}case .err(_):{R{.n=3}}};}fn i32 terminal(Result<i32,bool> value){return switch(value){case .ok(n):{n}case .err(_):{return 4;}};}",b"");
    let g=lowered(ref p,declared(ref p,4));assert(cfg::deferred_count(ref g)==2);var usize i=1;var usize drops=0;var usize invokes=0;while(i<=cfg::count(ref g)){let n=cfg::at(ref g,i);if(n.kind==5){assert(clean_ir::drop_at(ir::cleanup_ir(ref p),n.drop-1).exit==9);drops+=1;}if(n.kind==14){invokes+=1;}i+=1;}assert(drops==1 && invokes==2);let moved=lowered(ref p,declared(ref p,5));i=1;drops=0;while(i<=cfg::count(ref moved)){if(cfg::at(ref moved,i).kind==5){drops+=1;}i+=1;}assert(drops==0);let returned=lowered(ref p,declared(ref p,6));assert(cfg::check(ref p,ref returned).code==0);
}

fn u32 switch_mutated(ref ir::Program p,ref cfg::Graph g,u32 mode){
    var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();var usize i=1;var usize branch=0;var usize seed=0;while(i<=cfg::count(g)){if(cfg::at(g,i).kind==18 && branch==0){branch=i;seed=cfg::at(g,i).first;}i+=1;}assert(branch!=0 && seed!=0);i=1;
    while(i<=cfg::count(g)){var n=cfg::at(g,i);if(i==branch){if(mode==1){n.target=0;}if(mode==2){n.source=lookup::node(ir::at(p,cfg::frame(g)),ir::node_at(ir::at(p,cfg::frame(g)),n.source-1).value.left);}if(mode==3){n.first=cfg::at(g,seed).first;}if(mode==4){n.second=n.first;}if(mode==5){n.start+=1;}if(mode==8){n.first=n.second;}if(mode==9){n.second=branch;}}
        if(i==seed && mode==6){n.kind=7;n.source=0;n.target=0;}if(i==seed && mode==7){n.target=cfg::at(g,branch).source;}assert(v::push<cfg::Block>(ref mut blocks,n).is_ok);i+=1;}
    if(mode==10){assert(v::push<cfg::Block>(ref mut blocks,cfg::at(g,branch)).is_ok);}return switch(cfg::assemble(p,cfg::frame(g),if(mode==11){seed}else if(mode==12){branch}else{cfg::entry(g)},blocks,cfg::work(g))){case .ok(_):{0}case .err(e):{e.code}};
}
@test @effects(console) fn void cfg_switches_reject_missing_duplicate_relabelled_and_unguarded_case_edges(){
    let p=retained(b"module main;enum E{A;B{i32 n;};}fn i32 f(E value){return switch(value){case .B{.n=n}:{n}case .A:{1}};}",b"");let g=lowered(ref p,1);assert(switch_mutated(ref p,ref g,0)==0);var u32 mode=1;while(mode<=12){assert(switch_mutated(ref p,ref g,mode)==1001);mode+=1;}
}

@test @effects(console) fn void cfg_anonymous_payloads_keep_distinct_slots_and_all_owned_exit_cleanup(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}fn Result<i32,i32> bad(){return .err(7);}fn void fall(Result<R,R> value){switch(value){case .ok(_):{let R inner=R{.n=3};}case .err(_):{}};}fn i32 early(Result<R,i32> value){return switch(value){case .ok(_):{return 4;}case .err(_):{0}};}fn Result<i32,i32> propagate(Result<R,i32> value){let i32 n=switch(value){case .ok(_):{bad()?}case .err(_):{0}};return .ok(n);}fn void leave(){while(true){let Result<R,i32> value=.ok(R{.n=5});switch(value){case .ok(_):{break;}case .err(_):{break;}};}}enum E{Data{i32 a;i32 b;};}fn i32 fields(E value){return switch(value){case .Data{.a=_,.b=_}:{0}};}",b"");
    var usize frame=1;var usize anonymous=0;var usize scoped=0;var usize returned=0;var usize propagated=0;var usize broken=0;while(frame<=ir::count(ref p)){let b=ir::at(ref p,frame);let g=lowered(ref p,frame);var usize i=1;while(i<=ir::local_count(b)){let local=ir::local_at(b,i);if(local.kind==4){let n=ir::node_at(b,lookup::node(b,local.source_index)-1);assert(bytes::equal(ir::node_text(b,lookup::node(b,local.source_index)-1),b"_") && n.value.place.root==i);anonymous+=1;}i+=1;}i=1;while(i<=cfg::count(ref g)){let n=cfg::at(ref g,i);if(n.kind==5){let d=clean_ir::drop_at(ir::cleanup_ir(ref p),n.drop-1);let o=clean_ir::operand_at(ir::cleanup_ir(ref p),d.operand);if(o.kind==1 && ir::local_at(b,o.id).kind==4){if(d.exit==9){scoped+=1;}if(d.exit==2){returned+=1;}if(d.exit==6){propagated+=1;}if(d.exit==3){broken+=1;}}}i+=1;}frame+=1;}
    assert(anonymous==10 && scoped==3 && returned==1 && propagated==1 && broken==1);
}
