module module_state_graph_source_context;
import module_state_graph_model as m;
import project_cache_encoding as budget;
import std::vec as v;

pub struct Span{usize start;usize end;}
// Names remain immutable spans in the owned parser source. Value and borrow IDs
// occupy separate namespaces; aliases retain presence without inventing reads.
pub struct Binding{usize start;usize end;u32 codec;usize value;usize found;usize borrow;}
pub struct Counters{usize values;usize borrows;usize capability;usize remaining;bool helper;}
// Helpers are sorted by canonical logical name. Declaration is zero based;
// index is its nonzero whole-program callee index. Root is never in this table.
pub struct Function{usize declaration;usize index;u32 effects;Span name;}
pub fn Result<usize,u32> value(ref mut Counters ids,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(ids.values>=4294967295){return .err(1002);}ids.values+=1;return .ok(ids.values);
}
pub fn Result<usize,u32> borrow(ref mut Counters ids,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}if(ids.borrows>=4294967295){return .err(1002);}ids.borrows+=1;return .ok(ids.borrows);
}
pub fn u32 emit(m::Operation op,Span span,ref Counters ids,ref mut v::Vec<m::Operation> ops,ref mut v::Vec<Span> spans,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}if(ids.remaining>m::operation_limit() || v::len<m::Operation>(ops)>=ids.remaining){return 1002;}
    if(!v::push<m::Operation>(ops,op).is_ok || !v::push<Span>(spans,span).is_ok){return 1003;}return 0;
}
