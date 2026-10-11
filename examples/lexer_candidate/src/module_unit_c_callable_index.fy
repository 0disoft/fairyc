module module_unit_c_callable_index;
import module_unit_body_ir_model as ir;
import module_unit_value_model as budget;
import std::vec as v;

pub resource Index {v::Vec<usize> slots;cleanup{}}
fn usize bucket(usize module_index,usize declaration,usize mask){return (module_index*104729 ^ declaration*3266489917)&mask;}
pub fn Result<Index,u32> build(ref ir::Program p,ref mut usize work){
    let count=ir::callable_count(p);if(count>4096){return .err(1002);}var usize size=1;while(size<count*2){size*=2;}
    var v::Vec<usize> slots=v::empty<usize>();var usize i=0;while(i<size){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut slots,0).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<count){let c=ir::callable_at(p,i);if(c.module_index>=256 || c.declaration==0 || c.declaration>1048576){return .err(1001);}var usize at=bucket(c.module_index,c.declaration,size-1);
        while(true){if(!budget::charge(work,1)){return .err(1002);}let id=*v::at<usize>(ref slots,at);if(id==0){let cell=v::at_mut<usize>(ref mut slots,at);*cell=i+1;break;}let old=ir::callable_at(p,id-1);if(old.module_index==c.module_index && old.declaration==c.declaration){return .err(1001);}at=(at+1)&(size-1);}i+=1;
    }return .ok(Index{.slots});
}
pub fn usize find(ref ir::Program p,ref Index index,usize module_index,usize declaration,ref mut usize work,ref mut u32 code){
    if(*code!=0){return 0;}if(module_index>=256 || declaration==0 || declaration>1048576){*code=1001;return 0;}let mask=v::len<usize>(ref index.slots)-1;var usize at=bucket(module_index,declaration,mask);
    while(true){if(!budget::charge(work,1)){*code=1002;return 0;}let id=*v::at<usize>(ref index.slots,at);if(id==0){return 0;}let c=ir::callable_at(p,id-1);if(c.module_index==module_index && c.declaration==declaration){return id;}at=(at+1)&mask;}return 0;
}
