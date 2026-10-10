module module_unit_cfg_drops;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import std::vec as v;

// IDs stay in ascending original drop-table order. Private storage cannot be
// assembled or changed by callers; CFG verification builds its own table.
pub resource Table {usize frame;usize total;v::Vec<usize> ids;cleanup{}}
fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
pub fn Result<Table,clean::Problem> build(ref ir::Program p,usize frame,ref mut usize work){
    let cleaned=ir::cleanup_ir(p);if(frame==0 || frame>ir::count(p) || frame>clean::frame_count(cleaned)){return .err(fail(1001,0,0,0));}let module_index=clean::frame_at(cleaned,frame).module_index;var v::Vec<usize> ids=v::empty<usize>();var usize i=0;
    while(i<clean::drop_count(cleaned)){if(!budget::charge(work,1)){return .err(fail(1002,module_index,0,0));}let d=clean::drop_at(cleaned,i);if(d.operand==0 || d.operand>clean::operand_count(cleaned)){return .err(fail(1001,module_index,d.start,d.end));}let o=clean::operand_at(cleaned,d.operand);if(o.frame==0 || o.frame>ir::count(p)){return .err(fail(1001,module_index,d.start,d.end));}if(o.frame==frame && !v::push<usize>(ref mut ids,i+1).is_ok){return .err(fail(1003,module_index,d.start,d.end));}i+=1;}
    return .ok(Table{.frame,.total=clean::drop_count(cleaned),.ids});
}
pub fn usize count(ref Table p){return v::len<usize>(ref p.ids);}
pub fn usize at(ref Table p,usize row){return *v::at<usize>(ref p.ids,row);}
pub fn bool matches(ref Table p,usize frame,usize total){return p.frame==frame && p.total==total;}
pub fn usize row(ref Table p,usize id,ref mut usize work){var usize low=0;var usize high=count(p);while(low<high){if(!budget::charge(work,1)){return 0;}let mid=low+(high-low)/2;if(at(p,mid)<id){low=mid+1;}else{high=mid;}}return if(low<count(p) && at(p,low)==id){low+1}else{0};}
