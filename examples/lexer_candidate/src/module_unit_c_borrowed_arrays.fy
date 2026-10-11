module module_unit_c_borrowed_arrays;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cfg_model as cfg;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_aggregate as aggregate;
import module_unit_c_scalar as scalar;
import c_text as w;
import std::bytes as bytes;
import std::vec as v;

// Fuse only an adjacent field/index/read chain. Calls, stores and control-flow
// changes keep their by-value array snapshot and original evaluation order.
pub resource Plan {v::Vec<u8> omitted;cleanup{}}
pub fn bool omitted(ref Plan p,usize id){return id<v::len<u8>(ref p.omitted) && *v::at<u8>(ref p.omitted,id)!=0;}
fn bool bump(ref ir::Body b,usize original,ref mut v::Vec<usize> uses){let id=lookup::node(b,original);if(id!=0){let slot=v::at_mut<usize>(uses,id);*slot+=1;}return true;}
pub fn Result<Plan,u32> build(ref ir::Program p,ref ir::Body b,ref cfg::Graph g,ref mut usize work){
    var v::Vec<usize> uses=v::empty<usize>();var v::Vec<usize> evaluations=v::empty<usize>();var v::Vec<u8> flags=v::empty<u8>();var usize i=0;
    while(i<=ir::node_count(b)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut uses,0).is_ok || !v::push<usize>(ref mut evaluations,0).is_ok || !v::push<u8>(ref mut flags,0).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::node_count(b)){if(!budget::charge(work,1)){return .err(1002);}let e=ir::node_at(b,i).value;discard bump(b,e.left,ref mut uses);discard bump(b,e.right,ref mut uses);i+=1;}
    i=0;while(i<ir::statement_count(b)){if(!budget::charge(work,1)){return .err(1002);}let s=ir::statement_at(b,i).value;discard bump(b,s.expression,ref mut uses);discard bump(b,s.target,ref mut uses);i+=1;}
    i=1;while(i<=cfg::count(g)){if(!budget::charge(work,1)){return .err(1002);}let n=cfg::at(g,i);if(n.kind==1){let slot=v::at_mut<usize>(ref mut evaluations,n.source);*slot+=1;}i+=1;}
    i=1;while(i<=cfg::count(g)){if(!budget::charge(work,1)){return .err(1002);}let n=cfg::at(g,i);i+=1;if(n.kind!=1 || n.first==0){continue;}let e=ir::node_at(b,n.source-1).value;
        if(e.kind!=8 || e.field.id==0 || clean::type_at(ir::cleanup_ir(p),e.type).kind!=16 || *v::at<usize>(ref uses,n.source)!=1 || *v::at<usize>(ref evaluations,n.source)!=1){continue;}
        let receiver=lookup::node(b,e.left);if(receiver==0){continue;}let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,receiver-1).value.type);if(rt.kind!=7 && rt.kind!=8){continue;}
        let index=cfg::at(g,n.first);if(index.kind!=1 || index.first==0){continue;}let ie=ir::node_at(b,index.source-1).value;if((ie.kind!=1 && ie.kind!=2) || scalar::scalar(p,ie.type)!=5){continue;}
        let read=cfg::at(g,index.first);if(read.kind!=1){continue;}let re=ir::node_at(b,read.source-1).value;if(re.kind!=9 || re.left!=e.source_index || re.right!=ie.source_index){continue;}
        let slot=v::at_mut<u8>(ref mut flags,n.source);*slot=1;
    }return .ok(Plan{.omitted=flags});
}
pub fn bool read(ref ir::Program p,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s){
    let e=ir::node_at(b,id-1).value;let left=lookup::node(b,e.left);let right=lookup::node(b,e.right);let t=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);
    return w::text(out,s,b"if(") && scalar::value(out,s,right) && w::text(out,s,b">=") && w::number(out,s,t.second) && w::text(out,s,b")fy_runtime_trap(6);\n")
        && scalar::value(out,s,id) && w::text(out,s,b"=") && aggregate::place(p,b,left,out,s,0) && w::text(out,s,b".fy_items[") && scalar::value(out,s,right) && w::text(out,s,b"];\n");
}
