module tests::closure_project_link;
import module_project_compile as compile;
import module_project_link as link;
import module_project_calls as calls;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_body_ir_closures as closures;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_lower as flow;
import module_unit_cfg_model as cfg;
import module_unit_c_frame_index as targets;
import module_unit_c_callable_index as contracts;
import module_unit_c_reachability as reach;
import module_units as grouping;
import module_signatures as signatures;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import project_source_plan as planning;
import generic_aggregate_types as aggregates;
import sha256_stream as sha;
import copied_closure_tables as private;
import syntax as syntax;
import scanner as scanner;
import std::vec as v;
import std::bytes as bytes;
import std::io as io;

// Only original source bytes enter this producer. No source identity, codec,
// Complete owner or emitted C is used to grant project-link authority.
pub struct Receipt {u32 phase;u32 code;usize module_index;usize start;usize end;usize units;usize frames;usize made;usize called;usize helpers;usize drops;}
fn Receipt failed(u32 phase,u32 code,usize module_index,usize start,usize end){return Receipt{.phase,.code,.module_index,.start,.end,.units=0,.frames=0,.made=0,.called=0,.helpers=0,.drops=0};}
fn syntax::Parsed parsed(ref u8[] text){
    let input=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;
}
fn bool aggregate_source(ref v::Vec<syntax::Parsed> sources){var usize m=0;while(m<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,m);var usize i=1;while(i<syntax::declaration_count(p)){let d=syntax::declaration_at(p,i);if((d.kind==4 || d.kind==6) && d.generic_parameters!=0){return true;}i+=1;}m+=1;}return false;}
fn Result<signatures::Program,Receipt> prepared_signatures(ref v::Vec<syntax::Parsed> sources){
    if(!aggregate_source(sources)){return switch(signatures::build(sources)){case .ok(p):{.ok(p)}case .err(e):{.err(failed(1,e.code,e.module_index,e.start,e.end))}};}
    // The scalar aggregate neighbors are single original modules, without
    // literals/generic closure transport. Keep their sealed prerequisite.
    if(v::len<syntax::Parsed>(sources)!=1){return .err(failed(1,9,0,0,0));}
    var owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok);let imports=v::empty<catalog::Binding>();var names=bytes::empty();var digest=sha::empty();
    assert(sha::update(ref mut digest,syntax::source_view(v::at<syntax::Parsed>(sources,0))) && bytes::append(ref mut names,b"fypc1-sha256-").is_ok && sha::append_hex(ref digest,ref mut names));
    var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=bytes::len(ref names)}).is_ok);let table=coordinates::assemble(names,rows);var usize work=0;
    let original=switch(planning::build_aggregate_sources(sources,ref owners,ref imports,ref table,ref mut work)){case .ok(p):{p}case .err(e):{return .err(failed(1,e.code,e.module_index,e.start,e.end));}};
    let prepared=switch(aggregates::build(original,sources,ref mut work)){case .ok(p):{p}case .err(e):{return .err(failed(1,e.code,e.module_index,e.start,e.end));}};
    return switch(signatures::build_aggregates(sources,prepared,ref mut work)){case .ok(p):{.ok(p)}case .err(e):{.err(failed(1,e.code,e.module_index,e.start,e.end))}};
}
fn usize frame(ref ir::Program p,clean::Frame original){var usize id=0;var usize i=1;while(i<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.kind==original.kind && f.module_index==original.module_index && f.declaration==original.declaration){assert(id==0);id=i;}i+=1;}return id;}
// Compare source-qualified helper identities, not type IDs or project frame
// numbers. A padded provider deliberately has declaration IDs beyond its graph.
fn void preserved(ref ir::Program p,ref v::Vec<ir::Program> units,bool out_of_graph){var usize unit=0;var bool witnessed=false;
    while(unit<v::len<ir::Program>(units)){let input=v::at<ir::Program>(units,unit);var usize f=1;while(f<=ir::count(input)){let original=clean::frame_at(ir::cleanup_ir(input),f);let id=frame(p,original);assert(id!=0);let before=ir::at(input,f);let after=ir::at(p,id);assert(ir::body(before)==ir::body(after) && ir::source_bytes(before)==ir::source_bytes(after) && ir::start(before)==ir::start(after) && ir::end(before)==ir::end(after));var usize i=0;
        while(i<ir::node_count(before)){let n=ir::node_at(before,i);if(n.value.kind==36){let mapped=ir::node(after,n.value.source_index);assert(mapped!=0);let kept=ir::node_at(after,mapped-1);assert(kept.value.kind==36 && kept.value.target==n.value.target && kept.value.left==n.value.left && kept.value.right==n.value.right && kept.syntax.token==n.syntax.token && kept.syntax.start==n.syntax.start && kept.syntax.end==n.syntax.end);witnessed=witnessed || n.value.target>clean::type_count(ir::cleanup_ir(input));}i+=1;}f+=1;}unit+=1;
    }assert(!out_of_graph || witnessed);
}
fn Receipt inspect(ref ir::Program p,ref v::Vec<syntax::Parsed> sources,bool closure_mode,bool aggregate_mode){
    let cleaned=ir::cleanup_ir(p);assert(clean::project(cleaned) && clean::closures(cleaned)==closure_mode && clean::aggregates(cleaned)==aggregate_mode && proof::check(p).code==0);var usize linked_work=0;assert(calls::check(p,ref mut linked_work).code==0);
    var Receipt r=failed(0,0,0,0,0);r.frames=ir::count(p);r.drops=clean::drop_count(cleaned);var graphs=v::empty<cfg::Graph>();var usize entry=0;var usize f=1;
    while(f<=ir::count(p)){let b=ir::at(p,f);let owner=clean::frame_at(cleaned,f);assert(owner.module_index<v::len<syntax::Parsed>(sources));let source=v::at<syntax::Parsed>(sources,owner.module_index);assert(owner.declaration!=0 && owner.declaration<=syntax::declaration_count(source));let declaration=syntax::declaration_at(source,owner.declaration-1);
        if(owner.module_index==0 && owner.kind==3 && bytes::equal(syntax::declaration_name(source,owner.declaration),b"main")){assert(entry==0);entry=f;}assert(ir::source_bytes(b)==syntax::source_view(source).len && ir::start(b)>=declaration.start && ir::end(b)<=declaration.end);
        var usize i=0;while(i<ir::node_count(b)){let n=ir::node_at(b,i);let original=syntax::expression_at(source,n.value.source_index);assert(n.syntax.kind==original.kind && n.syntax.start==original.start && n.syntax.end==original.end);if(n.value.kind==36){var usize captures=0;var usize tail=n.value.right;while(tail!=0){let at=ir::node(b,tail);assert(at!=0);tail=ir::node_at(b,at-1).value.right;captures+=1;}var usize work=0;let helper=closures::helper(p,f,n.value.target,captures,ref mut work);assert(helper!=0 && clean::frame_at(cleaned,helper).module_index==owner.module_index && clean::frame_at(cleaned,helper).declaration==n.value.target);r.made+=1;r.helpers+=1;}i+=1;}
        i=0;while(i<ir::call_count(b)){let c=ir::call_at(b,i);if(c.kind==4){assert(c.checked && c.target_module==0 && c.target_declaration==0 && c.instance==0 && c.argument_count==1);let e=ir::node_at(b,c.expression-1).value;assert(e.target==0 && closures::i32(cleaned,e.type));let named=ir::node(b,e.left);assert(named!=0 && closures::leaf(cleaned,ir::node_at(b,named-1).value.type));r.called+=1;}i+=1;}
        let graph=switch(flow::lower(p,f)){case .ok(g):{g}case .err(_):{assert(false);return failed(4,1001,owner.module_index,0,0);}};assert(v::push<cfg::Graph>(ref mut graphs,graph).is_ok);f+=1;
    }
    assert(entry!=0);var usize work=0;let callable=switch(contracts::build(p,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return failed(4,1001,0,0,0);}};
    let indexed=switch(targets::build(p,ref callable,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return failed(4,1001,0,0,0);}};
    let selected=switch(reach::build(p,ref graphs,ref callable,ref indexed,entry,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return failed(4,1001,0,0,0);}};
    f=1;while(f<=ir::count(p)){if(reach::frame(ref selected,f)){let b=ir::at(p,f);var usize i=0;while(i<ir::node_count(b)){let n=ir::node_at(b,i);if(n.value.kind==36){let callee=ir::node(b,n.value.left);assert(callee!=0);let token=ir::node_at(b,callee-1).syntax.token;assert(token==private::helper_flag()+n.value.target);var usize captures=0;var usize tail=n.value.right;while(tail!=0){captures+=1;tail=ir::node_at(b,ir::node(b,tail)-1).value.right;}let helper=closures::helper(p,f,n.value.target,captures,ref mut work);assert(helper!=0 && reach::frame(ref selected,helper));}i+=1;}}f+=1;}
    return r;
}
fn Receipt linked(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,u32 mode,bool closure_mode,bool aggregate_mode,bool out_of_graph){
    if(mode==3){let grouped=switch(compile::build(sources,metadata)){case .ok(p):{p}case .err(e):{return failed(2,e.code,e.module_index,e.start,e.end);}};let p=switch(link::build(metadata,compile::programs(ref grouped))){case .ok(p):{p}case .err(e):{return failed(3,e.code,e.module_index,e.start,e.end);}};preserved(ref p,compile::programs(ref grouped),out_of_graph);var r=inspect(ref p,sources,closure_mode,aggregate_mode);r.units=compile::count(ref grouped);return r;}
    var units=v::empty<ir::Program>();var usize i=0;let usize total=if(mode==0){1}else{v::len<syntax::Parsed>(sources)};
    while(i<total){let usize first=if(mode==0){0}else if(mode==2){total-i-1}else{i};let usize stop=if(mode==0){v::len<syntax::Parsed>(sources)}else{first+1};let p=switch(compile::range(sources,metadata,first,stop)){case .ok(p):{p}case .err(e):{return failed(2,e.code,e.module_index,e.start,e.end);}};assert(v::push<ir::Program>(ref mut units,p).is_ok);i+=1;}
    let p=switch(link::build(metadata,ref units)){case .ok(p):{p}case .err(e):{return failed(3,e.code,e.module_index,e.start,e.end);}};preserved(ref p,ref units,out_of_graph);var r=inspect(ref p,sources,closure_mode,aggregate_mode);r.units=total;return r;
}
@effects(console) fn void require(ref u8[] name,u32 mode,Receipt r,bool accept){if((r.code==0)!=accept || !accept && (r.phase<1 || r.phase>3 || r.code==10 || r.code==11 || r.code==12 || r.code==1002 || r.code==1003 || r.code==4503)){discard io::write_stderr("closure-project-link case=");discard io::write_stderr_bytes(name);discard io::write_stderr(" mode=");discard io::write_stderr_u32(mode);discard io::write_stderr(" phase=");discard io::write_stderr_u32(r.phase);discard io::write_stderr(" code=");discard io::write_stderr_u32(r.code);discard io::write_stderr("\n");assert(false);}}
@effects(console) pub fn void verify(ref u8[] name,ref u8[] root,ref u8[] first,ref u8[] second,bool accept,bool closure_mode,bool aggregate_mode,usize made,usize called,usize batches,bool out_of_graph){
    var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(first.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(first)).is_ok);}if(second.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(second)).is_ok);}
    let metadata=switch(prepared_signatures(ref sources)){case .ok(p):{p}case .err(r):{require(name,0,r,accept);discard io::write("closure-project-link case=");discard io::write_bytes(name);discard io::write(" refused\n");return;}};
    let groups=switch(grouping::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(grouping::count(ref groups)==batches);var u32 mode=if(batches==1){0}else{1};var usize frames=0;var usize drops=0;
    while(mode<=3){let r=linked(ref sources,ref metadata,mode,closure_mode,aggregate_mode,out_of_graph);require(name,mode,r,accept);if(accept){assert(r.made==made && r.called==called && r.helpers==made);if(frames==0){frames=r.frames;drops=r.drops;}assert(r.frames==frames && r.drops==drops && r.units==if(mode==0){1}else if(mode==3){batches}else{v::len<syntax::Parsed>(ref sources)});}mode+=1;}
    discard io::write("closure-project-link case=");discard io::write_bytes(name);discard io::write(" modes=");discard io::write_usize(if(batches==1){4}else{3});discard io::write(" frames=");discard io::write_usize(frames);discard io::write(" made=");discard io::write_usize(made);discard io::write(" called=");discard io::write_usize(called);discard io::write("\n");
}
@test fn void closure_project_link_missing_duplicate_and_empty_inputs_stay_closed(){
    var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module main;import provider as p;fn i32 main(){let action=p::make();return action(2);}")).is_ok);assert(v::push<syntax::Parsed>(ref mut sources,parsed(b"module provider;pub fn fn i32(i32) make(){let fn i32(i32) action=fn [] i32(i32 x){return x;};return action;}")).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};var units=v::empty<ir::Program>();assert(switch(link::build(ref metadata,ref units)){case .ok(_):{false}case .err(e):{e.code==1001}});
    var usize i=0;while(i<2){let p=switch(compile::range(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(v::push<ir::Program>(ref mut units,p).is_ok);assert(switch(link::build(ref metadata,ref units)){case .ok(_):{false}case .err(e):{e.code==1001}});i+=1;}
}
