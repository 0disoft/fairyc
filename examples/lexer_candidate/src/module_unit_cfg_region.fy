module module_unit_cfg_region;
import module_unit_cfg_model as cfg;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_value_model as m;
import std::vec as v;
import module_unit_cfg_drops as drops;

fn clean_ir::Problem fail(u32 code,usize module_index,usize start,usize end){return clean_ir::Problem{.code,.module_index,.start,.end};}
fn Result<usize,clean_ir::Problem> emit(u32 kind,usize target,usize next,usize drop,usize start,usize end,usize module_index,ref mut v::Vec<cfg::Block> blocks,ref mut usize work){
    if(v::len<cfg::Block>(blocks)>=32768 || !m::charge(work,1)){return .err(fail(1002,module_index,start,end));}
    return switch(v::push<cfg::Block>(blocks,cfg::Block{.kind,.source=0,.target,.first=next,.second=0,.drop,.start,.end})){case .ok(n):{.ok(n)}case .err(_):{.err(fail(1003,module_index,start,end))}};
}
fn Result<usize,clean_ir::Problem> group(ref ir::Body b,clean_ir::Operand operand,ref v::Vec<usize> resets,usize module_index,ref mut usize work){
    var usize i=0;while(i<v::len<usize>(resets)){if(!m::charge(work,1)){return .err(fail(1002,module_index,operand.start,operand.end));}let e=ir::reset_at(b,*v::at<usize>(resets,i)-1);let r=ir::region_at(b,e.region_id-1);if(operand.start>=r.body_start && operand.end<=r.body_end){return .ok(i);}i+=1;}return .ok(v::len<usize>(resets));
}
pub fn Result<usize,clean_ir::Problem> exit_indexed(ref ir::Program p,ref drops::Table table,usize frame,u32 reason,usize start,usize end,u32 reset_kind,usize reset_start,usize reset_end,usize next,ref mut v::Vec<cfg::Block> blocks,ref mut usize work){
    let cleaned=ir::cleanup_ir(p);if(frame==0 || frame>ir::count(p) || !drops::matches(table,frame,clean_ir::drop_count(cleaned))){return .err(fail(1001,0,start,end));}let b=ir::at(p,frame);let module_index=clean_ir::frame_at(cleaned,frame).module_index;var v::Vec<usize> resets=v::empty<usize>();var v::Vec<usize> ordered=v::empty<usize>();var usize i=0;
    while(i<ir::reset_count(b)){if(!m::charge(work,1)){return .err(fail(1002,module_index,start,end));}let e=ir::reset_at(b,i);if(reset_kind!=0 && e.kind==reset_kind && e.start==reset_start && e.end==reset_end){var bool duplicate=false;var usize j=0;while(j<v::len<usize>(ref resets)){if(!m::charge(work,1)){return .err(fail(1002,module_index,start,end));}let old=ir::reset_at(b,*v::at<usize>(ref resets,j)-1);if(old.region_id==e.region_id){if(old.order!=e.order){return .err(fail(1001,module_index,start,end));}duplicate=true;}j+=1;}if(!duplicate && !v::push<usize>(ref mut resets,i+1).is_ok){return .err(fail(1003,module_index,start,end));}}i+=1;}
    i=1;while(i<v::len<usize>(ref resets)){var usize j=i;while(j>0){if(!m::charge(work,1)){return .err(fail(1002,module_index,start,end));}let before=*v::at<usize>(ref resets,j-1);let here=*v::at<usize>(ref resets,j);if(ir::reset_at(b,before-1).order<=ir::reset_at(b,here-1).order){break;}{let slot=v::at_mut<usize>(ref mut resets,j-1);*slot=here;}{let slot=v::at_mut<usize>(ref mut resets,j);*slot=before;}j-=1;}i+=1;}i=0;while(i<v::len<usize>(ref resets)){if(ir::reset_at(b,*v::at<usize>(ref resets,i)-1).order!=i){return .err(fail(1001,module_index,start,end));}i+=1;}
    i=0;while(i<drops::count(table)){if(!m::charge(work,1)){return .err(fail(1002,module_index,start,end));}let id=drops::at(table,i);if(id==0 || id>clean_ir::drop_count(cleaned)){return .err(fail(1001,module_index,start,end));}let d=clean_ir::drop_at(cleaned,id-1);if(d.operand==0 || d.operand>clean_ir::operand_count(cleaned)){return .err(fail(1001,module_index,start,end));}let o=clean_ir::operand_at(cleaned,d.operand);if(o.frame!=frame){return .err(fail(1001,module_index,start,end));}
        if(o.frame==frame && d.exit==reason && d.start==start && d.end==end){var bool duplicate=false;var usize j=0;while(j<v::len<usize>(ref ordered)){if(!m::charge(work,1)){return .err(fail(1002,module_index,start,end));}if(clean_ir::drop_at(cleaned,*v::at<usize>(ref ordered,j)-1).operand==d.operand){duplicate=true;}j+=1;}if(!duplicate && !v::push<usize>(ref mut ordered,id).is_ok){return .err(fail(1003,module_index,start,end));}}i+=1;
    }
    i=1;while(i<v::len<usize>(ref ordered)){var usize j=i;while(j>0){if(!m::charge(work,1)){return .err(fail(1002,module_index,start,end));}let before=*v::at<usize>(ref ordered,j-1);let here=*v::at<usize>(ref ordered,j);let a=clean_ir::operand_at(cleaned,clean_ir::drop_at(cleaned,before-1).operand);let b=clean_ir::operand_at(cleaned,clean_ir::drop_at(cleaned,here-1).operand);if(a.kind<b.kind || a.kind==b.kind && a.start<=b.start){break;}{let slot=v::at_mut<usize>(ref mut ordered,j-1);*slot=here;}{let slot=v::at_mut<usize>(ref mut ordered,j);*slot=before;}j-=1;}i+=1;}
    var usize entry=next;var usize selected=v::len<usize>(ref resets);
    while(true){if(selected<v::len<usize>(ref resets)){let id=*v::at<usize>(ref resets,selected);let e=ir::reset_at(b,id-1);entry=switch(emit(16,id,entry,0,e.start,e.end,module_index,blocks,work)){case .ok(n):{n}case .err(e):{return .err(e);}};}
        i=0;while(i<v::len<usize>(ref ordered)){let id=*v::at<usize>(ref ordered,i);let d=clean_ir::drop_at(cleaned,id-1);let o=clean_ir::operand_at(cleaned,d.operand);let assigned=switch(group(b,o,ref resets,module_index,work)){case .ok(n):{n}case .err(e):{return .err(e);}};if(assigned==selected){entry=switch(emit(5,0,entry,id,start,end,module_index,blocks,work)){case .ok(n):{n}case .err(e):{return .err(e);}};}i+=1;}
        if(selected==0){break;}selected-=1;
    }return .ok(entry);
}
pub fn Result<usize,clean_ir::Problem> exit(ref ir::Program p,usize frame,u32 reason,usize start,usize end,u32 reset_kind,usize reset_start,usize reset_end,usize next,ref mut v::Vec<cfg::Block> blocks,ref mut usize work){let table=switch(drops::build(p,frame,work)){case .ok(t):{t}case .err(e):{return .err(e);}};return exit_indexed(p,ref table,frame,reason,start,end,reset_kind,reset_start,reset_end,next,blocks,work);}
pub fn usize find_region(ref ir::Body b,usize statement){var usize i=0;while(i<ir::region_count(b)){if(ir::region_at(b,i).statement==statement){return i+1;}i+=1;}return 0;}
pub fn clean_ir::Problem check(ref ir::Program p,ref cfg::Graph g,ref mut usize work){
    let b=ir::at(p,cfg::frame(g));let module_index=clean_ir::frame_at(ir::cleanup_ir(p),cfg::frame(g)).module_index;var v::Vec<u8> begins=v::empty<u8>();var v::Vec<u8> resets=v::empty<u8>();var usize i=0;
    while(i<ir::region_count(b)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}if(!v::push<u8>(ref mut begins,0).is_ok){return fail(1003,module_index,0,0);}i+=1;}i=0;while(i<ir::reset_count(b)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}if(!v::push<u8>(ref mut resets,0).is_ok){return fail(1003,module_index,0,0);}i+=1;}
    i=1;while(i<=cfg::count(g)){if(!m::charge(work,1)){return fail(1002,module_index,0,0);}let n=cfg::at(g,i);
        if(n.kind==15){let r=ir::region_at(b,n.target-1);if(n.source!=lookup::node(b,r.source) || n.start!=r.start || n.end!=r.end || *v::at<u8>(ref begins,n.target-1)!=0){return fail(1001,module_index,n.start,n.end);}let e=ir::node_at(b,n.source-1).value;let t=clean_ir::type_at(ir::cleanup_ir(p),e.type);if(e.type!=r.source_type || t.kind!=8 || t.first!=r.arena_type){return fail(1001,module_index,n.start,n.end);}let slot=v::at_mut<u8>(ref mut begins,n.target-1);*slot=1;}
        else if(n.kind==16){let e=ir::reset_at(b,n.target-1);if(n.source!=0 || n.start!=e.start || n.end!=e.end){return fail(1001,module_index,n.start,n.end);}var usize j=0;while(j<ir::reset_count(b)){if(!m::charge(work,1)){return fail(1002,module_index,n.start,n.end);}let other=ir::reset_at(b,j);if(other.region_id==e.region_id && other.kind==e.kind && other.order==e.order && other.start==e.start && other.end==e.end){if(*v::at<u8>(ref resets,j)!=0){return fail(1001,module_index,n.start,n.end);}let slot=v::at_mut<u8>(ref mut resets,j);*slot=1;}j+=1;}}i+=1;
    }
    i=0;while(i<v::len<u8>(ref begins)){if(*v::at<u8>(ref begins,i)==0){return fail(1001,module_index,0,0);}i+=1;}i=0;while(i<v::len<u8>(ref resets)){if(*v::at<u8>(ref resets,i)==0){return fail(1001,module_index,0,0);}i+=1;}return fail(0,0,0,0);
}
