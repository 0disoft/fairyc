module module_unit_cfg_iteration;
import module_unit_cfg_model as cfg;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as m;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
fn bool named(ref ir::Body b,usize id,usize depth,ref mut usize work){if(id==0 || depth>=64 || !m::charge(work,1)){return false;}let e=ir::node_at(b,id-1).value;if(e.kind==8 || e.kind==26){return named(b,lookup::node(b,e.left),depth+1,work);}if(e.kind!=2 || e.binding==0 || e.binding>ir::local_count(b)){return false;}let l=ir::local_at(b,e.binding);return (l.kind==1 || l.kind==2) && l.type==e.type;}
// The original scanner/output path copies u8 values from a shared byte view.
// Other iteration carriers require their own borrow/element transport.
pub fn Result<usize,clean::Problem> binding(ref ir::Program p,ref ir::Body b,usize frame,usize statement,ref mut usize work){let module_index=clean::frame_at(ir::cleanup_ir(p),frame).module_index;
    if(statement==0 || statement>ir::statement_count(b)){return .err(fail(1001,module_index,0,0));}let s=ir::statement_at(b,statement-1).value;let source=lookup::node(b,s.expression);
    if(s.kind!=18 || source==0){return .err(fail(1001,module_index,s.start,s.end));}let e=ir::node_at(b,source-1).value;if(e.state!=1 || clean::type_at(ir::cleanup_ir(p),e.type).kind!=10 || !named(b,source,0,work)){return .err(fail(if(*work>=1048576){1002}else{1004},module_index,s.start,s.end));}
    var usize found=0;var usize i=1;while(i<=ir::local_count(b)){if(!m::charge(work,1)){return .err(fail(1002,module_index,s.start,s.end));}let l=ir::local_at(b,i);if(l.kind==5 && l.source_index==s.target){let t=clean::type_at(ir::cleanup_ir(p),l.type);if(found!=0 || t.kind!=1 || t.scalar!=3){return .err(fail(1004,module_index,s.start,s.end));}found=i;}i+=1;}return if(found==0){.err(fail(1001,module_index,s.start,s.end))}else{.ok(found)};
}
fn Result<bool,clean::Problem> before_capture(ref cfg::Graph g,usize capture,usize step,usize module_index,ref mut usize work){var v::Vec<u8> seen=v::empty<u8>();var v::Vec<usize> queue=v::empty<usize>();var usize i=0;
    while(i<cfg::count(g)){if(!m::charge(work,1)){return .err(fail(1002,module_index,0,0));}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(fail(1003,module_index,0,0));}i+=1;}
    if(!v::push<usize>(ref mut queue,cfg::entry(g)).is_ok){return .err(fail(1003,module_index,0,0));}{let slot=v::at_mut<u8>(ref mut seen,cfg::entry(g)-1);*slot=1;}i=0;
    while(i<v::len<usize>(ref queue)){if(!m::charge(work,1)){return .err(fail(1002,module_index,0,0));}let id=*v::at<usize>(ref queue,i);if(id==step){return .ok(true);}if(id!=capture){let n=cfg::at(g,id);var usize edge=0;while(edge<2){let next=if(edge==0){n.first}else{n.second};if(next!=0 && *v::at<u8>(ref seen,next-1)==0){{let slot=v::at_mut<u8>(ref mut seen,next-1);*slot=1;}if(!v::push<usize>(ref mut queue,next).is_ok){return .err(fail(1003,module_index,0,0));}}edge+=1;}}i+=1;}return .ok(false);
}
// Validate paired capture/step sites, source identity and capture dominance.
// Bounds are tested by the step before loading the iteration binding.
pub fn clean::Problem check(ref ir::Program p,ref cfg::Graph g,ref mut usize work){let b=ir::at(p,cfg::frame(g));let module_index=clean::frame_at(ir::cleanup_ir(p),cfg::frame(g)).module_index;var usize statement=1;
    while(statement<=ir::statement_count(b)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}let s=ir::statement_at(b,statement-1).value;if(s.kind==18){let valid=switch(binding(p,b,cfg::frame(g),statement,work)){case .ok(n):{n}case .err(e):{return e;}};var usize capture=0;var usize step=0;var usize i=1;
            while(i<=cfg::count(g)){if(!m::charge(work,1)){return fail(1002,module_index,s.start,s.end);}let n=cfg::at(g,i);if(n.target==statement && (n.kind==20 || n.kind==21)){if(n.source!=lookup::node(b,s.expression) || n.start!=s.start || n.end!=s.end || n.drop!=0){return fail(1001,module_index,s.start,s.end);}if(n.kind==20){if(capture!=0){return fail(1001,module_index,s.start,s.end);}capture=i;}else{if(step!=0){return fail(1001,module_index,s.start,s.end);}step=i;}}i+=1;}
            if(valid==0 || capture==0 || step==0 || cfg::at(g,capture).first!=step){return fail(1001,module_index,s.start,s.end);}let bypass=switch(before_capture(g,capture,step,module_index,work)){case .ok(b):{b}case .err(e):{return e;}};if(bypass){return fail(1001,module_index,s.start,s.end);}
        }statement+=1;
    }var usize i=1;while(i<=cfg::count(g)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}let n=cfg::at(g,i);if((n.kind==20 || n.kind==21) && ir::statement_at(b,n.target-1).value.kind!=18){return fail(1001,module_index,n.start,n.end);}i+=1;}return fail(0,module_index,0,0);
}
