module tests::closure_unit_body;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lower as lower;
import module_unit_body_ir_verify as proof;
import module_unit_body_ir_closures as closures;
import module_unit_cleanup_mir_model as clean;
import module_unit_checked_bodies as checked;
import module_unit_cfg_lower as flow;
import module_unit_cfg_model as cfg;
import module_unit_value_model as values;
import module_unit_region_model as regions;
import module_body_artifact_encode as old_body;
import module_cleanup_artifact_encode as old_cleanup;
import module_signatures as signatures;
import copied_closure_tables as private;
import syntax as syntax;
import scanner as scanner;
import std::vec as v;
import std::bytes as bytes;
import std::io as io;

pub fn Result<ir::Program,u32> from_source(ref u8[] text){
    let data=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{return .err(11);}};
    let stream=switch(scanner::collect(data,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{return .err(1001);}};
    let source=switch(syntax::parse(stream,512)){case .ok(s):{s}case .err(_):{return .err(1001);}};
    if(syntax::problem_count(ref source)!=0){return .err(syntax::problem_at(ref source,0).code);}var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,source).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    let owned=switch(checked::build(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{return .err(e.code);}};
    return switch(lower::lower_checked(ref sources,ref owned)){case .ok(p):{.ok(p)}case .err(e):{.err(e.code)}};
}
// Clone only the immutable body records. No mutation of the source Program can
// grant authority to the forged body supplied to the independent closure proof.
fn ir::Body changed(ref ir::Body b,u32 mode,usize closure_type){
    var nodes=v::empty<ir::Node>();var statements=v::empty<ir::Statement>();var locals=v::empty<values::Local>();var calls=v::empty<values::Invocation>();var args=v::empty<values::Argument>();var scopes=v::empty<regions::Scope>();var placements=v::empty<regions::Placement>();var resets=v::empty<regions::Exit>();var usize i=0;
    while(i<ir::node_count(b)){var n=ir::node_at(b,i);
        if(n.value.kind==36){if(mode==1){n.value.target=0;}else if(mode==2){n.value.target+=1;}else if(mode==3){n.value.state=2;}else if(mode==4){n.value.right=0;n.syntax.right=0;}}
        if(mode==5 && n.syntax.token>private::snapshot_flag()){n.syntax.token=private::snapshot_flag();}
        assert(v::push<ir::Node>(ref mut nodes,n).is_ok);i+=1;
    }
    i=0;while(i<ir::statement_count(b)){assert(v::push<ir::Statement>(ref mut statements,ir::statement_at(b,i)).is_ok);i+=1;}
    i=1;while(i<=ir::local_count(b)){var l=ir::local_at(b,i);if(mode==8 && l.kind==2 && l.type==closure_type){l.initializer=0;}assert(v::push<values::Local>(ref mut locals,l).is_ok);i+=1;}
    i=0;while(i<ir::call_count(b)){var c=ir::call_at(b,i);if(c.kind==4){if(mode==6){c.checked=false;}else if(mode==7){c.target_declaration=1;}}assert(v::push<values::Invocation>(ref mut calls,c).is_ok);i+=1;}
    i=0;while(i<ir::argument_count(b)){assert(v::push<values::Argument>(ref mut args,ir::argument_at(b,i)).is_ok);i+=1;}
    i=0;while(i<ir::region_count(b)){assert(v::push<regions::Scope>(ref mut scopes,ir::region_at(b,i)).is_ok);i+=1;}
    i=0;while(i<ir::placement_count(b)){assert(v::push<regions::Placement>(ref mut placements,ir::placement_at(b,i)).is_ok);i+=1;}
    i=0;while(i<ir::reset_count(b)){assert(v::push<regions::Exit>(ref mut resets,ir::reset_at(b,i)).is_ok);i+=1;}
    let text=switch(bytes::copy(ir::text_view(b))){case .ok(p):{p}case .err(_):{assert(false);return ir::assemble_body_regions(0,0,0,0,nodes,statements,locals,calls,args,scopes,placements,resets,bytes::empty());}};
    return ir::mark_inline(ir::assemble_body_regions(ir::body(b),ir::source_bytes(b),ir::start(b),ir::end(b),nodes,statements,locals,calls,args,scopes,placements,resets,text),ir::inline_hash(b));
}
@effects(console) pub fn void verify(ref u8[] name,ref u8[] text,usize constructs,usize invocations,usize captures){
    let p=switch(from_source(text)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(name);discard io::write_stderr(" lower code=");discard io::write_stderr_u32(e);discard io::write_stderr("\n");assert(false);return;}};
    assert(clean::closures(ir::cleanup_ir(ref p)) && proof::check(ref p).code==0);var usize made=0;var usize called=0;var usize observed=0;var usize tested=0;var usize frame=1;
    while(frame<=ir::count(ref p)){let b=ir::at(ref p,frame);var usize ctor=0;var usize calls=0;var bool snapshots=false;var usize closure_type=0;var usize i=1;
        while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(l.kind==2 && closures::leaf(ir::cleanup_ir(ref p),l.type)){closure_type=l.type;}i+=1;}i=0;
        while(i<ir::node_count(b)){let n=ir::node_at(b,i);if(n.value.kind==36){ctor+=1;snapshots=snapshots || n.value.right!=0;}if(n.value.kind==2 && n.value.state==1 && n.syntax.token>private::snapshot_flag()){observed+=1;}i+=1;}i=0;while(i<ir::call_count(b)){if(ir::call_at(b,i).kind==4){calls+=1;}i+=1;}
        made+=ctor;called+=calls;var u32 mode=1;while(mode<=8){let applicable=if(mode<=3){ctor!=0}else if(mode<=5){snapshots}else if(mode<=7){calls!=0}else{closure_type!=0};if(applicable){let forged=changed(b,mode,closure_type);var usize work=0;assert(closures::check_body(ref p,ref forged,frame,ref mut work).code==1001 && work<=1048576);tested+=1;}mode+=1;}
        let graph=switch(flow::lower(ref p,frame)){case .ok(g):{g}case .err(e):{discard io::write_stderr_bytes(name);discard io::write_stderr(" CFG code=");discard io::write_stderr_u32(e.code);discard io::write_stderr("\n");assert(false);return;}};assert(cfg::entry(ref graph)!=0);
        var usize work=0;assert(switch(old_body::encode(ref p,frame,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});frame+=1;
    }
    assert(made==constructs && called==invocations && observed==captures && tested!=0);var usize work=0;assert(switch(old_cleanup::encode(ir::cleanup_ir(ref p),ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});
}
