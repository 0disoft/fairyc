module module_unit_c_borrowed_arrays;
import module_unit_body_ir_model as ir;
import module_unit_cfg_model as cfg;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_aggregate as aggregate;
import module_unit_c_scalar as scalar;
import c_text as w;
import std::bytes as bytes;
import std::vec as v;

// Fuse adjacent single-use projections. Calls, stores and control-flow changes
// retain snapshots. Marker 1 is a borrowed array read; 2 is an inert struct field.
pub resource Plan {v::Vec<u8> omitted;cleanup{}}
pub fn bool omitted(ref Plan p,usize id){return id<v::len<u8>(ref p.omitted) && *v::at<u8>(ref p.omitted,id)!=0;}
pub fn bool field_omitted(ref Plan p,usize id){return id<v::len<u8>(ref p.omitted) && *v::at<u8>(ref p.omitted,id)==2;}
fn Result<bool,u32> field_place(ref ir::Program p,ref ir::Body b,usize id,ref mut usize work){var usize current=id;var usize depth=0;while(current!=0 && depth<64){if(!budget::charge(work,1)){return .err(1002);}let e=ir::node_at(b,current-1).value;if(e.kind==2){return .ok(e.binding!=0);}if(e.kind==4){return .ok(bytes::equal(ir::node_text(b,current-1),b"*"));}if(e.kind!=8 || e.field.id==0){return .ok(false);}let left=ir::node(b,e.left);if(left==0){return .ok(false);}let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);if(rt.kind==7 || rt.kind==8){return .ok(true);}if(rt.kind!=2){return .ok(false);}current=left;depth+=1;}return .ok(false);}
fn bool bump(ref ir::Body b,usize original,ref mut v::Vec<usize> uses){let id=ir::node(b,original);if(id!=0){let slot=v::at_mut<usize>(uses,id);*slot+=1;}return true;}
pub fn Result<Plan,u32> build(ref ir::Program p,ref ir::Body b,ref cfg::Graph g,ref mut usize work){
    var v::Vec<usize> uses=v::empty<usize>();var v::Vec<usize> evaluations=v::empty<usize>();var v::Vec<u8> flags=v::empty<u8>();var usize i=0;var bool candidate=false;
    while(i<=ir::node_count(b)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut uses,0).is_ok || !v::push<usize>(ref mut evaluations,0).is_ok || !v::push<u8>(ref mut flags,0).is_ok){return .err(1003);}
        if(i<ir::node_count(b)){let e=ir::node_at(b,i).value;if(e.state==1 && (e.kind==2 || e.kind==8)){let t=clean::type_at(ir::cleanup_ir(p),e.type);if(t.kind==2 || e.kind==8 && e.field.id!=0 && t.kind==16){candidate=true;}}}i+=1;}
    // Neither fusion rule can match without an inert struct place or an array
    // field. Keep the zero markers and avoid unrelated use/statement/CFG scans.
    if(!candidate){return .ok(Plan{.omitted=flags});}
    i=0;while(i<ir::node_count(b)){if(!budget::charge(work,1)){return .err(1002);}let e=ir::node_at(b,i).value;discard bump(b,e.left,ref mut uses);discard bump(b,e.right,ref mut uses);i+=1;}
    i=0;while(i<ir::statement_count(b)){if(!budget::charge(work,1)){return .err(1002);}let s=ir::statement_at(b,i).value;discard bump(b,s.expression,ref mut uses);discard bump(b,s.target,ref mut uses);i+=1;}
    i=1;while(i<=cfg::count(g)){if(!budget::charge(work,1)){return .err(1002);}let n=cfg::at(g,i);if(n.kind==1){let slot=v::at_mut<usize>(ref mut evaluations,n.source);*slot+=1;}i+=1;}
    i=1;while(i<=cfg::count(g)){if(!budget::charge(work,1)){return .err(1002);}let n=cfg::at(g,i);i+=1;if(n.kind!=1 || n.first==0){continue;}let e=ir::node_at(b,n.source-1).value;
        if(*v::at<usize>(ref uses,n.source)!=1 || *v::at<usize>(ref evaluations,n.source)!=1){continue;}let type=clean::type_at(ir::cleanup_ir(p),e.type);
        let placeable=if(type.kind==2 && (e.kind==2 || e.kind==8)){switch(field_place(p,b,n.source,work)){case .ok(b):{b}case .err(code):{return .err(code);}}}else{false};
        if(placeable){let next=cfg::at(g,n.first);if(next.kind==1){let read=ir::node_at(b,next.source-1).value;if(read.kind==8 && read.field.id!=0 && read.left==e.source_index){let slot=v::at_mut<u8>(ref mut flags,n.source);*slot=2;}}continue;}
        if(e.kind!=8 || e.field.id==0 || type.kind!=16){continue;}
        let receiver=ir::node(b,e.left);if(receiver==0){continue;}let rt=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,receiver-1).value.type);if(rt.kind!=7 && rt.kind!=8){continue;}
        let index=cfg::at(g,n.first);if(index.kind!=1 || index.first==0){continue;}let ie=ir::node_at(b,index.source-1).value;if((ie.kind!=1 && ie.kind!=2) || scalar::scalar(p,ie.type)!=5){continue;}
        let read=cfg::at(g,index.first);if(read.kind!=1){continue;}let re=ir::node_at(b,read.source-1).value;if(re.kind!=9 || re.left!=e.source_index || re.right!=ie.source_index){continue;}
        let slot=v::at_mut<u8>(ref mut flags,n.source);*slot=1;
    }return .ok(Plan{.omitted=flags});
}
pub fn bool read(ref ir::Program p,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s){
    let e=ir::node_at(b,id-1).value;let left=ir::node(b,e.left);let right=ir::node(b,e.right);let t=clean::type_at(ir::cleanup_ir(p),ir::node_at(b,left-1).value.type);
    return w::text(out,s,b"if(") && scalar::value(out,s,right) && w::text(out,s,b">=") && w::number(out,s,t.second) && w::text(out,s,b")") && w::trap(out,s,6,e.start,e.end) && w::text(out,s,b";\n")
        && scalar::value(out,s,id) && w::text(out,s,b"=") && aggregate::place(p,b,left,out,s,0) && w::text(out,s,b".fy_items[") && scalar::value(out,s,right) && w::text(out,s,b"];\n");
}
// Preserve the opaque live-storage read required by the MSVC alias regression,
// but copy the selected field rather than its complete inert struct owner.
pub fn bool read_field(ref ir::Program p,ref ir::Body b,usize id,ref mut bytes::Buffer out,ref mut w::State s){return w::text(out,s,b"fy_unit_copy(&") && scalar::value(out,s,id) && w::text(out,s,b",&(") && aggregate::place(p,b,id,out,s,0) && w::text(out,s,b"),sizeof ") && scalar::value(out,s,id) && w::text(out,s,b");\n");}
