module module_unit_c_layout_index;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import std::vec as v;

// Original one-based layout row IDs, independent of retained row ordering.
pub resource Index {v::Vec<usize> rows;cleanup{}}
pub fn Result<Index,u32> build(ref ir::Program p,ref mut usize work){
    let count=clean::type_count(ir::cleanup_ir(p));var v::Vec<usize> rows=v::empty<usize>();var usize i=0;
    while(i<count){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut rows,0).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<ir::layout_count(p)){if(!budget::charge(work,1)){return .err(1002);}let id=ir::layout_at(p,i).value.type;if(id==0 || id>count || *v::at<usize>(ref rows,id-1)!=0){return .err(1001);}{let cell=v::at_mut<usize>(ref mut rows,id-1);*cell=i+1;}i+=1;}
    return .ok(Index{.rows});
}
pub fn usize find(ref Index index,usize type,ref mut usize work,ref mut u32 code){
    if(*code!=0){return 0;}if(!budget::charge(work,1)){*code=1002;return 0;}if(type==0 || type>v::len<usize>(ref index.rows)){*code=1001;return 0;}return *v::at<usize>(ref index.rows,type-1);
}
