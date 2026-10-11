module module_unit_c_reachability;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_model as cfg;
import module_unit_c_frame_index as targets;
import module_unit_c_callable_index as contracts;
import module_unit_value_model as budget;
import std::vec as v;

pub resource Plan {v::Vec<u8> frames;v::Vec<u8> drops;usize count;cleanup{}}
fn u32 mark(ref mut v::Vec<u8> marks,ref mut v::Vec<usize> pending,usize id,ref mut usize work){
    if(id==0 || id>v::len<u8>(marks)){return 1001;}if(!budget::charge(work,1)){return 1002;}if(*v::at<u8>(marks,id-1)!=0){return 0;}if(!v::push<usize>(pending,id).is_ok){return 1003;}let cell=v::at_mut<u8>(marks,id-1);*cell=2;return 0;
}
fn Result<v::Vec<u8>,u32> empty(usize count,ref mut usize work){var v::Vec<u8> marks=v::empty<u8>();var usize i=0;while(i<count){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<u8>(ref mut marks,0).is_ok){return .err(1003);}i+=1;}return .ok(marks);}
// Callers supply immutable checked IR, its checked CFGs and original indexes.
// Calls and cleanup bodies form one fixed point; resource fields and tagged
// success/error payloads retain cleanup dependencies in their declared order.
pub fn Result<Plan,u32> build(ref ir::Program p,ref v::Vec<cfg::Graph> graphs,ref contracts::Index callables,ref targets::Index indexed,usize entry,ref mut usize work){
    if(v::len<cfg::Graph>(graphs)!=ir::count(p)){return .err(1001);}var frames=switch(empty(ir::count(p),work)){case .ok(m):{m}case .err(e):{return .err(e);}};var drops=switch(empty(clean::type_count(ir::cleanup_ir(p)),work)){case .ok(m):{m}case .err(e):{return .err(e);}};
    var v::Vec<usize> pending_frames=v::empty<usize>();var v::Vec<usize> pending_types=v::empty<usize>();let seeded=mark(ref mut frames,ref mut pending_frames,entry,work);if(seeded!=0){return .err(seeded);}var usize frame_cursor=0;var usize type_cursor=0;
    while(frame_cursor<v::len<usize>(ref pending_frames) || type_cursor<v::len<usize>(ref pending_types)){
        while(frame_cursor<v::len<usize>(ref pending_frames)){let frame=*v::at<usize>(ref pending_frames,frame_cursor);let b=ir::at(p,frame);let graph=v::at<cfg::Graph>(graphs,frame-1);if(cfg::frame(graph)!=frame){return .err(1001);}var usize i=0;
            while(i<ir::call_count(b)){if(!budget::charge(work,1)){return .err(1002);}let call=ir::call_at(b,i);var u32 code=0;let found=targets::find(p,indexed,call.target_module,call.target_declaration,call.instance,call.kind==2,work,ref mut code);if(code!=0){return .err(code);}if(found!=0){let e=mark(ref mut frames,ref mut pending_frames,found,work);if(e!=0){return .err(e);}}else{let id=contracts::find(p,callables,call.target_module,call.target_declaration,work,ref mut code);if(code!=0){return .err(code);}if(id==0 || !ir::callable_at(p,id-1).foreign){return .err(1001);}}i+=1;}
            i=1;while(i<=cfg::count(graph)){if(!budget::charge(work,1)){return .err(1002);}let n=cfg::at(graph,i);if(n.kind==5){let d=clean::drop_at(ir::cleanup_ir(p),n.drop-1);let o=clean::operand_at(ir::cleanup_ir(p),d.operand);let e=mark(ref mut drops,ref mut pending_types,o.type,work);if(e!=0){return .err(e);}}i+=1;}frame_cursor+=1;
        }
        while(type_cursor<v::len<usize>(ref pending_types)){if(!budget::charge(work,1)){return .err(1002);}let id=*v::at<usize>(ref pending_types,type_cursor);let g=clean::glue_at(ir::cleanup_ir(p),id);if(g.kind==0){return .err(1001);}if(g.frame!=0){let e=mark(ref mut frames,ref mut pending_frames,g.frame,work);if(e!=0){return .err(e);}}
            if(g.kind==1){var usize i=0;while(i<g.field_count){if(!budget::charge(work,1)){return .err(1002);}let child=clean::field_at(ir::cleanup_ir(p),g.field_start+i).type;if(clean::glue_at(ir::cleanup_ir(p),child).kind!=0){let e=mark(ref mut drops,ref mut pending_types,child,work);if(e!=0){return .err(e);}}i+=1;}}
            else if(g.kind==2 || g.kind==3){let t=clean::type_at(ir::cleanup_ir(p),id);if(clean::glue_at(ir::cleanup_ir(p),t.first).kind!=0){let e=mark(ref mut drops,ref mut pending_types,t.first,work);if(e!=0){return .err(e);}}if(g.kind==3 && clean::glue_at(ir::cleanup_ir(p),t.second).kind!=0){let e=mark(ref mut drops,ref mut pending_types,t.second,work);if(e!=0){return .err(e);}}}else{return .err(1001);}type_cursor+=1;
        }
    }return .ok(Plan{.frames,.drops,.count=v::len<usize>(ref pending_frames)});
}
pub fn bool frame(ref Plan p,usize id){assert(id!=0 && id<=v::len<u8>(ref p.frames));return *v::at<u8>(ref p.frames,id-1)!=0;}
pub fn ref v::Vec<u8> drops(ref Plan p){return ref p.drops;}
pub fn usize count(ref Plan p){return p.count;}
pub fn Result<Plan,u32> all(ref ir::Program p,ref mut usize work){var frames=switch(empty(ir::count(p),work)){case .ok(m):{m}case .err(e):{return .err(e);}};var usize i=0;while(i<v::len<u8>(ref frames)){let cell=v::at_mut<u8>(ref mut frames,i);*cell=2;i+=1;}return .ok(Plan{.frames,.drops=v::empty<u8>(),.count=ir::count(p)});}
