module module_unit_cfg_defer;
import module_unit_cfg_model as cfg;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_value_model as m;
import std::vec as v;

fn clean_ir::Problem fail(u32 code,usize module_index,usize start,usize end){return clean_ir::Problem{.code,.module_index,.start,.end};}
pub fn usize find(ref v::Vec<cfg::Deferred> sites,usize statement){var usize i=0;while(i<v::len<cfg::Deferred>(sites)){if(v::at<cfg::Deferred>(sites,i).statement==statement){return i+1;}i+=1;}return 0;}
fn clean_ir::Problem target(ref ir::Program p,usize frame,usize invocation,ref mut usize work){
    let b=ir::at(p,frame);let f=clean_ir::frame_at(ir::cleanup_ir(p),frame);
    if(invocation==0 || invocation>ir::call_count(b)){return fail(1001,f.module_index,0,0);}let call=ir::call_at(b,invocation-1);let e=ir::node_at(b,call.expression-1).value;let t=clean_ir::type_at(ir::cleanup_ir(p),e.type);
    if(e.kind!=6 || e.builtin.kind!=0 || t.kind!=1 || t.scalar!=7 || !call.checked){return fail(1001,f.module_index,e.start,e.end);}
    var bool found=false;var usize i=0;
    if(call.kind==1){while(i<ir::callable_count(p)){if(!m::charge(work,1)){return fail(1002,f.module_index,e.start,e.end);}let c=ir::callable_at(p,i);if(c.module_index==call.target_module && c.declaration==call.target_declaration){if(c.foreign || c.template || c.result!=e.type || c.parameter_count!=call.argument_count || (f.effects&c.effects)!=c.effects){return fail(1001,f.module_index,e.start,e.end);}found=true;break;}i+=1;}}
    else if(call.kind==2){i=1;while(i<=ir::count(p)){if(!m::charge(work,1)){return fail(1002,f.module_index,e.start,e.end);}let target=clean_ir::frame_at(ir::cleanup_ir(p),i);if(target.kind==1 && target.context==call.instance && target.module_index==call.target_module && target.declaration==call.target_declaration && target.result==e.type){if((f.effects&target.effects)!=target.effects){return fail(1001,f.module_index,e.start,e.end);}found=true;break;}i+=1;}}
    if(!found){return fail(1001,f.module_index,e.start,e.end);}i=0;while(i<call.argument_count){if(!m::charge(work,1)){return fail(1002,f.module_index,e.start,e.end);}let a=ir::argument_at(b,call.argument_start+i);if(!a.checked || a.actual!=a.expected || a.actual==0){return fail(1001,f.module_index,a.start,a.end);}let t=clean_ir::type_at(ir::cleanup_ir(p),a.actual);if(t.kind!=1 || t.scalar<1 || t.scalar>6){return fail(1001,f.module_index,a.start,a.end);}i+=1;}return fail(0,0,0,0);
}
pub fn Result<usize,clean_ir::Problem> prepare(ref ir::Program p,usize frame,usize statement,ref mut v::Vec<cfg::Deferred> sites,ref mut v::Vec<cfg::Capture> captures,ref mut usize work){
    let old=find(sites,statement);if(old!=0){return .ok(old);}let b=ir::at(p,frame);let f=clean_ir::frame_at(ir::cleanup_ir(p),frame);let s=ir::statement_at(b,statement-1).value;
    if(s.kind!=13){return .err(fail(1001,f.module_index,s.start,s.end));}let expr=lookup::node(b,s.expression);var usize invocation=0;var usize i=0;
    while(i<ir::call_count(b)){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,s.start,s.end));}if(ir::call_at(b,i).expression==expr){invocation=i+1;break;}i+=1;}let checked=target(p,frame,invocation,work);if(checked.code!=0){return .err(checked);}let call=ir::call_at(b,invocation-1);
    if(v::len<cfg::Deferred>(sites)>=256 || v::len<cfg::Capture>(captures)>4096 || call.argument_count>4096-v::len<cfg::Capture>(captures)){return .err(fail(1002,f.module_index,s.start,s.end));}
    let id=v::len<cfg::Deferred>(sites)+1;let start=v::len<cfg::Capture>(captures);i=0;
    while(i<call.argument_count){if(!m::charge(work,1)){return .err(fail(1002,f.module_index,s.start,s.end));}let a=ir::argument_at(b,call.argument_start+i);if(!v::push<cfg::Capture>(captures,cfg::Capture{.deferred=id,.position=i,.source=a.expression,.type=a.actual}).is_ok){return .err(fail(1003,f.module_index,a.start,a.end));}i+=1;}
    return switch(v::push<cfg::Deferred>(sites,cfg::Deferred{.statement,.invocation,.capture_start=start,.capture_count=call.argument_count})){case .ok(n):{.ok(n)}case .err(_):{.err(fail(1003,f.module_index,s.start,s.end))}};
}
pub fn clean_ir::Problem check(ref ir::Program p,ref cfg::Graph g,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));let f=clean_ir::frame_at(ir::cleanup_ir(p),cfg::frame(g));let nd=cfg::deferred_count(g);let nc=cfg::capture_count(g);
    if(nd>256 || nc>4096){return fail(1002,f.module_index,0,0);}var usize expected_start=0;var usize i=1;var v::Vec<u8> captures=v::empty<u8>();
    while(i<=nc){if(!v::push<u8>(ref mut captures,0).is_ok){return fail(1003,f.module_index,0,0);}i+=1;}i=1;
    while(i<=nd){if(!m::charge(work,1)){return fail(1002,f.module_index,0,0);}let d=cfg::deferred_at(g,i);
        if(d.statement==0 || d.statement>ir::statement_count(b) || d.capture_start!=expected_start || d.capture_count>nc-expected_start){return fail(1001,f.module_index,0,0);}let s=ir::statement_at(b,d.statement-1).value;let checked=target(p,cfg::frame(g),d.invocation,work);if(checked.code!=0){return checked;}let call=ir::call_at(b,d.invocation-1);
        if(s.kind!=13 || lookup::node(b,s.expression)!=call.expression || d.capture_count!=call.argument_count){return fail(1001,f.module_index,s.start,s.end);}
        var usize j=1;while(j<i){if(!m::charge(work,1)){return fail(1002,f.module_index,s.start,s.end);}if(cfg::deferred_at(g,j).statement==d.statement){return fail(1001,f.module_index,s.start,s.end);}j+=1;}
        j=0;while(j<d.capture_count){if(!m::charge(work,1)){return fail(1002,f.module_index,s.start,s.end);}let c=cfg::capture_at(g,d.capture_start+j+1);let a=ir::argument_at(b,call.argument_start+j);if(c.deferred!=i || c.position!=j || c.source!=a.expression || c.type!=a.actual){return fail(1001,f.module_index,a.start,a.end);}j+=1;}expected_start+=d.capture_count;i+=1;
    }
    if(expected_start!=nc){return fail(1001,f.module_index,0,0);}i=1;while(i<=cfg::count(g)){if(!m::charge(work,1)){return fail(1002,f.module_index,0,0);}let n=cfg::at(g,i);
        if(n.kind==13){let c=cfg::capture_at(g,n.target);if(n.source!=c.source || *v::at<u8>(ref captures,n.target-1)!=0){return fail(1001,f.module_index,n.start,n.end);}let e=ir::node_at(b,c.source-1).value;if(e.type!=c.type || n.start!=e.start || n.end!=e.end){return fail(1001,f.module_index,n.start,n.end);}let slot=v::at_mut<u8>(ref mut captures,n.target-1);*slot=1;}
        else if(n.kind==14){let d=cfg::deferred_at(g,n.target);let call=ir::call_at(b,d.invocation-1);if(n.source!=call.expression || n.start!=call.start || n.end!=call.end){return fail(1001,f.module_index,n.start,n.end);}}i+=1;
    }
    i=1;while(i<=nc){if(*v::at<u8>(ref captures,i-1)==0){return fail(1001,f.module_index,0,0);}i+=1;}return fail(0,0,0,0);
}
