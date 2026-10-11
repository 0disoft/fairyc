module module_unit_interface_work;
import module_types as types;
import module_ast_join as join;
import std::bytes as bytes;
import std::vec as v;


pub struct Budget {usize work;}
pub fn join::Problem fail(u32 code,usize module_index,usize start,usize end){return join::Problem{.code,.module_index,.start,.end};}
pub fn bool charge(ref mut Budget work,usize amount){if(work.work>1048576 || amount>1048576-work.work){return false;}work.work+=amount;return true;}
pub fn u32 mark(ref mut v::Vec<types::Type> graph,ref mut v::Vec<usize> pending,ref mut bytes::Buffer seen,ref mut Budget work,usize id){
    if(id==0 || id>types::count(graph)){return 9;}if(!charge(work,1)){return 10;}
    if(bytes::len(seen)<=types::count(graph) && !bytes::resize_zeroed(seen,types::count(graph)+1).is_ok){return 11;}
    if(bytes::view(seen)[id]!=0){return 0;}bytes::view_mut(seen)[id]=1;return if(v::push<usize>(pending,id).is_ok){0}else{11};
}
pub fn Result<usize,u32> append(ref mut bytes::Buffer names,ref mut Budget work,ref u8[] text){if(!charge(work,text.len)){return .err(10);}let usize start=bytes::len(names);if(text.len>65536-start){return .err(10);}return if(bytes::append(names,text).is_ok){.ok(start)}else{.err(11)};}
