module module_state_graph_hash_encoding;
import module_state_graph_model as m;
import module_state_graph_program as program;
import module_state_layout as state;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

pub struct Status{u32 code;}
fn void save(u32 code,ref mut Status status){if(code!=0){status.code=if(code==4){1003}else{1002};}}
pub fn void raw(ref mut bytes::Buffer out,ref u8[] data,ref mut Status status,ref mut usize work){if(status.code==0){save(budget::raw(out,data,work),status);}}
pub fn void number(ref mut bytes::Buffer out,usize value,ref mut Status status,ref mut usize work){if(status.code==0){save(budget::length(out,value,work),status);}}
pub fn void text(ref mut bytes::Buffer out,ref u8[] data,ref mut Status status,ref mut usize work){number(out,data.len,status,work);raw(out,data,status,work);}
fn ref u8[] segment(ref u8[] pool,u32 k){var usize start=0;var usize end=0;var u32 index=0;while(end<pool.len){if(pool[end]==124){if(index==k){break;}index+=1;start=end+1;}end+=1;}return pool[start..end];}
fn ref u8[] kind(u32 k){return segment(b"|cell-read|cell-write|cell-delete|map-read|map-write|map-delete|cell-borrow|map-borrow|borrow-view|borrow-copy|call|",k);}
fn ref u8[] exit(u32 k){return segment(b"|jump|branch|commit|reject|trap|return|",k);}
fn void effects(ref mut bytes::Buffer out,u32 mask,ref mut Status status,ref mut usize work){number(out,if(mask==3){2}else if(mask==0){0}else{1},status,work);if((mask&1)!=0){text(out,b"storage_read",status,work);}if((mask&2)!=0){text(out,b"storage_write",status,work);}}
fn void operand(ref mut bytes::Buffer out,ref m::Graph graph,m::Operand value,ref mut Status status,ref mut usize work){text(out,state::codec(value.codec),status,work);text(out,m::data(graph)[value.start..value.end],status,work);}
fn void operation(ref mut bytes::Buffer out,ref program::Verified proof,ref m::Graph graph,m::Operation op,u32 mode,ref mut Status status,ref mut usize work){
    text(out,kind(op.kind),status,work);number(out,op.field,status,work);operand(out,graph,op.key,status,work);operand(out,graph,op.value,status,work);
    number(out,op.key.input,status,work);number(out,op.value.input,status,work);number(out,op.result,status,work);number(out,op.found,status,work);number(out,op.cost,status,work);
    if(mode!=0){text(out,if(op.callee==0){b""}else{program::helper_name(proof,op.callee-1)},status,work);}if(mode==2){number(out,op.borrow_input,status,work);number(out,op.borrow_result,status,work);number(out,op.view_offset,status,work);number(out,op.view_length,status,work);}
}
fn void arguments(ref mut bytes::Buffer out,ref m::Graph graph,usize start,usize count,ref mut Status status,ref mut usize work){number(out,count,status,work);var usize i=0;while(i<count && status.code==0){number(out,*v::at<usize>(m::arguments(graph),start+i),status,work);i+=1;}}
fn void terminator(ref mut bytes::Buffer out,ref m::Graph graph,m::Terminator term,ref mut Status status,ref mut usize work){
    text(out,exit(term.kind),status,work);number(out,term.condition,status,work);number(out,if(term.kind==1){term.target}else{0},status,work);arguments(out,graph,if(term.kind==1){term.argument_start}else{0},if(term.kind==1){term.argument_count}else{0},status,work);
    number(out,if(term.kind==2){term.target}else{0},status,work);arguments(out,graph,if(term.kind==2){term.argument_start}else{0},if(term.kind==2){term.argument_count}else{0},status,work);
    number(out,term.alternative,status,work);arguments(out,graph,term.alternate_start,term.alternate_count,status,work);text(out,m::data(graph)[term.category_start..term.category_end],status,work);number(out,term.cost,status,work);
}
pub fn bool has_borrows(ref m::Graph graph){var usize i=0;while(i<v::len<m::Operation>(m::operations(graph))){let op=*v::at<m::Operation>(m::operations(graph),i);if(op.kind>=7 && op.kind<=10){return true;}i+=1;}return false;}
// mode 0 graph-v1, 1 helpers-v1, 2 borrows-v1. Only sealed graphs enter here.
pub fn void executable(ref mut bytes::Buffer out,ref program::Verified proof,ref m::Graph graph,u32 mode,ref mut Status status,ref mut usize work){
    effects(out,m::header(graph).effects,status,work);text(out,if(mode==0){b"graph-v1"}else{b"graph"},status,work);number(out,m::header(graph).entry,status,work);number(out,v::len<m::Block>(m::blocks(graph)),status,work);
    var usize i=0;while(i<v::len<m::Block>(m::blocks(graph)) && status.code==0){let block=*v::at<m::Block>(m::blocks(graph),i);number(out,block.id,status,work);number(out,block.parameter_count,status,work);var usize j=0;
        while(j<block.parameter_count && status.code==0){let p=*v::at<m::Value>(m::parameters(graph),block.parameter_start+j);number(out,p.id,status,work);text(out,state::codec(p.codec),status,work);j+=1;}
        number(out,block.operation_count,status,work);j=0;while(j<block.operation_count && status.code==0){operation(out,proof,graph,*v::at<m::Operation>(m::operations(graph),block.operation_start+j),mode,status,work);j+=1;}terminator(out,graph,block.term,status,work);i+=1;
    }
}
