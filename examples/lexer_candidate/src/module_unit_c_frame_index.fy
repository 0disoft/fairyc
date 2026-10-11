module module_unit_c_frame_index;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_c_callable_index as contracts;
import module_unit_value_model as budget;
import candidate_limits as limits;
import std::vec as v;

struct Module {usize all;usize ordinary;usize first;usize last;usize count;usize declaration;bool ordered;}
struct Next {usize all;usize ordinary;}
pub resource Index {v::Vec<Module> modules;v::Vec<Next> next;bool direct;cleanup{}}
// Verify contiguous declaration order and specialization context identity while
// preparing module-local original frame lists. Never assume producer ordering.
pub fn Result<Index,u32> build(ref ir::Program p,ref contracts::Index callables,ref mut usize work){
    if(ir::count(p)>limits::project_frames()){return .err(1002);}var v::Vec<Module> modules=v::empty<Module>();var v::Vec<Next> next=v::empty<Next>();var bool direct=true;var usize i=0;
    while(i<limits::source_files()){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<Module>(ref mut modules,Module{.all=0,.ordinary=0,.first=0,.last=0,.count=0,.declaration=0,.ordered=true}).is_ok){return .err(1003);}i+=1;}
    i=1;while(i<=ir::count(p)){if(!budget::charge(work,1)){return .err(1002);}let f=clean::frame_at(ir::cleanup_ir(p),i);if(f.module_index>=limits::source_files() || f.declaration==0 || f.declaration>1048576 || f.context==0){return .err(1001);}var bool ordinary=f.kind==3;
        if(f.kind==1){if(f.context!=i){direct=false;}var u32 code=0;let id=contracts::find(p,callables,f.module_index,f.declaration,work,ref mut code);if(code!=0){return .err(code);}if(id!=0){let c=ir::callable_at(p,id-1);ordinary=!c.template && !c.foreign && c.result==f.result;}}
        let old=*v::at<Module>(ref modules,f.module_index);if(!v::push<Next>(ref mut next,Next{.all=old.all,.ordinary=if(ordinary){old.ordinary}else{0}}).is_ok){return .err(1003);}let cell=v::at_mut<Module>(ref mut modules,f.module_index);cell.all=i;
        if(ordinary){if(cell.count==0){cell.first=i;}else if(i!=cell.last+1 || f.declaration<=cell.declaration){cell.ordered=false;}cell.last=i;cell.count+=1;cell.declaration=f.declaration;cell.ordinary=i;}i+=1;
    }return .ok(Index{.modules,.next,.direct});
}
pub fn usize find(ref ir::Program p,ref Index index,usize module_index,usize declaration,usize instance,bool specialized,ref mut usize work,ref mut u32 code){
    if(*code!=0){return 0;}if(module_index>=limits::source_files() || declaration==0 || declaration>1048576){*code=1001;return 0;}if(!budget::charge(work,1)){*code=1002;return 0;}
    if(specialized && index.direct){if(instance==0 || instance>ir::count(p)){return 0;}let f=clean::frame_at(ir::cleanup_ir(p),instance);return if(f.kind==1 && f.context==instance && f.module_index==module_index && f.declaration==declaration){instance}else{0};}
    let m=*v::at<Module>(ref index.modules,module_index);
    if(!specialized && m.ordered){if(m.count==0){return 0;}var usize low=m.first;var usize high=m.last+1;while(low<high){if(!budget::charge(work,1)){*code=1002;return 0;}let mid=low+(high-low)/2;if(clean::frame_at(ir::cleanup_ir(p),mid).declaration<declaration){low=mid+1;}else{high=mid;}}if(low>m.last){return 0;}if(!budget::charge(work,1)){*code=1002;return 0;}return if(clean::frame_at(ir::cleanup_ir(p),low).declaration==declaration){low}else{0};}
    var usize row=if(specialized){m.all}else{m.ordinary};var usize found=0;while(row!=0){if(!budget::charge(work,1)){*code=1002;return 0;}let f=clean::frame_at(ir::cleanup_ir(p),row);if(f.declaration==declaration && (!specialized || f.kind==1 && f.context==instance)){if(found!=0){return 0;}found=row;}let next=*v::at<Next>(ref index.next,row-1);row=if(specialized){next.all}else{next.ordinary};}return found;
}
