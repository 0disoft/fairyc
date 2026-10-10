module module_unit_cfg_switches;
import module_unit_cfg_model as cfg;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as m;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
fn bool put(ref mut v::Vec<usize> map,usize id,usize value){if(id==0 || id>v::len<usize>(map) || *v::at<usize>(map,id-1)!=0){return false;}let slot=v::at_mut<usize>(map,id-1);*slot=value;return true;}
// Called after CFG value/successor bounds and the owning body proof. The arm
// IDs retain the exhaustive source table; the graph cannot omit or relabel a
// case, bypass its tag test, or enter another variant's payload seed.
pub fn clean::Problem check(ref ir::Program p,ref cfg::Graph g,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));let module_index=clean::frame_at(ir::cleanup_ir(p),cfg::frame(g)).module_index;
    var v::Vec<usize> owners=v::empty<usize>();var v::Vec<usize> cases=v::empty<usize>();var v::Vec<usize> seeds=v::empty<usize>();var usize i=0;
    while(i<ir::node_count(b)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}if(!v::push<usize>(ref mut owners,0).is_ok || !v::push<usize>(ref mut cases,0).is_ok || !v::push<usize>(ref mut seeds,0).is_ok){return fail(1003,module_index,0,0);}i+=1;}
    i=0;while(i<ir::node_count(b)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}let e=ir::node_at(b,i).value;if(e.kind==15){var usize link=e.right;while(link!=0){if(!m::charge(work,1)){return fail(1002,module_index,e.start,e.end);}let pair=lookup::node(b,link);if(pair==0){return fail(1001,module_index,e.start,e.end);}let row=ir::node_at(b,pair-1).value;let arm=lookup::node(b,row.left);if(row.kind!=19 || !put(ref mut owners,arm,i+1)){return fail(1001,module_index,e.start,e.end);}link=row.right;}}i+=1;}
    i=1;while(i<=cfg::count(g)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}let n=cfg::at(g,i);if(n.kind==18 || n.kind==19){
        if(n.source==0 || n.target==0 || ir::node_at(b,n.source-1).value.kind!=15 || *v::at<usize>(ref owners,n.target-1)!=n.source){return fail(1001,module_index,n.start,n.end);}let a=ir::node_at(b,n.target-1).value;if(a.kind!=18 || a.start!=n.start || a.end!=n.end || !(if(n.kind==18){put(ref mut cases,n.target,i)}else{put(ref mut seeds,n.target,i)})){return fail(1001,module_index,n.start,n.end);}
    }i+=1;}
    i=0;while(i<ir::node_count(b)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}let e=ir::node_at(b,i).value;if(e.kind==15){
        var usize link=e.right;var usize previous=0;while(link!=0){if(!m::charge(work,1)){return fail(1002,module_index,e.start,e.end);}let row=ir::node_at(b,lookup::node(b,link)-1).value;let arm=lookup::node(b,row.left);let branch=*v::at<usize>(ref cases,arm-1);let seed=*v::at<usize>(ref seeds,arm-1);if(branch==0 || seed==0){return fail(1001,module_index,e.start,e.end);}let n=cfg::at(g,branch);if(n.first!=seed || cfg::entry(g)==seed){return fail(1001,module_index,e.start,e.end);}
            if(previous==0){let trap=cfg::at(g,n.second);if(trap.kind!=12 || trap.start!=e.start || trap.end!=e.end){return fail(1001,module_index,e.start,e.end);}}else if(n.second!=previous){return fail(1001,module_index,e.start,e.end);}previous=branch;link=row.right;
        }
        // Every predecessor of a payload seed must be its own successful
        // case edge. Every non-root case is entered only by the previous test.
        var usize j=1;while(j<=cfg::count(g)){if(!m::charge(work,1)){return fail(1002,module_index,e.start,e.end);}let before=cfg::at(g,j);var usize edge=0;while(edge<2){let next=if(edge==0){before.first}else{before.second};if(next!=0){let dest=cfg::at(g,next);if(dest.source==i+1 && dest.kind==19 && (before.kind!=18 || before.source!=dest.source || before.target!=dest.target || edge!=0)){return fail(1001,module_index,e.start,e.end);}if(dest.source==i+1 && dest.kind==18 && next!=previous && (before.kind!=18 || before.source!=dest.source || edge!=1)){return fail(1001,module_index,e.start,e.end);}}edge+=1;}j+=1;}
        if(previous==0){return fail(1001,module_index,e.start,e.end);}let initial=cfg::at(g,cfg::entry(g));if(initial.kind==18 && initial.source==i+1 && cfg::entry(g)!=previous){return fail(1001,module_index,e.start,e.end);}
    }i+=1;}
    return fail(0,0,0,0);
}
