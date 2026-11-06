module module_state_graph_source_artifact;
import module_state_graph_source as source;
import module_state_graph_model as m;
import module_state_graph_program as program;
import module_state_graph_artifact as artifact;
import candidate_artifact_wire as wire;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// Exact current-source binding for the admitted source producer. Neither this
// private owner nor a detached graph authorizes Complete, cache hits or execution.
pub resource Restored{source::Produced current;program::Verified graph;cleanup{}}
pub fn ref source::Produced current(ref Restored p){return ref p.current;}
pub fn ref program::Verified graph(ref Restored p){return ref p.graph;}
fn u32 raw(ref mut bytes::Buffer out,ref u8[] value,ref mut usize work){
    if(value.len>wire::byte_limit() || bytes::len(out)>wire::byte_limit()-value.len){return 1002;}
    let code=budget::raw(out,value,work);return if(code==0){0}else if(code==4){1003}else{1002};
}
fn Result<m::Graph,u32> copy(ref m::Graph input,ref mut usize work){
    if(!budget::charge(work,m::data(input).len+1)){return .err(1002);}
    let data=switch(bytes::copy(m::data(input))){case .ok(p):{p}case .err(_):{return .err(1003);}};
    var v::Vec<m::Value> parameters=v::empty<m::Value>();var v::Vec<usize> arguments=v::empty<usize>();var v::Vec<m::Operation> operations=v::empty<m::Operation>();var v::Vec<m::Block> blocks=v::empty<m::Block>();var usize i=0;
    while(i<v::len<m::Value>(m::parameters(input))){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<m::Value>(ref mut parameters,*v::at<m::Value>(m::parameters(input),i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<v::len<usize>(m::arguments(input))){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<usize>(ref mut arguments,*v::at<usize>(m::arguments(input),i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<v::len<m::Operation>(m::operations(input))){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<m::Operation>(ref mut operations,*v::at<m::Operation>(m::operations(input),i)).is_ok){return .err(1003);}i+=1;}
    i=0;while(i<v::len<m::Block>(m::blocks(input))){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<m::Block>(ref mut blocks,*v::at<m::Block>(m::blocks(input),i)).is_ok){return .err(1003);}i+=1;}
    return .ok(m::assemble(m::header(input),data,parameters,arguments,operations,blocks));
}
fn Result<wire::Payload,u32> packet(ref source::Produced current,ref mut usize work){
    let root=switch(copy(source::graph(current),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let proof=switch(program::verify(program::assemble(root,v::empty<m::Graph>(),v::empty<program::Name>(),bytes::empty()),source::layouts(current),work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    return artifact::encode(ref proof,work);
}
pub fn Result<wire::Payload,u32> encode(ref source::Produced current,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}let input=source::source(current);if(input.len>source::source_limit()){return .err(1002);}
    let saved=switch(packet(current,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    var status=wire::Status{.code=0};var out=wire::writer(b"FYSGSRC0",ref mut status,work);
    wire::number(ref mut out,input.len,ref mut status,work);if(status.code!=0){return .err(status.code);}let first=raw(ref mut out,input,work);if(first!=0){return .err(first);}
    wire::number(ref mut out,wire::content(ref saved).len,ref mut status,work);if(status.code!=0){return .err(status.code);}let second=raw(ref mut out,wire::content(ref saved),work);if(second!=0){return .err(second);}
    return wire::finish(out,status);
}
pub fn Result<Restored,u32> restore(ref u8[] saved,ref u8[] actual,ref mut usize work){
    if(!budget::charge(work,0) || actual.len>source::source_limit()){return .err(1002);}
    var c=switch(wire::cursor(saved,b"FYSGSRC0",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let size=wire::count_from(saved,ref mut c,source::source_limit(),1,work);if(c.code!=0){return .err(c.code);}
    if(!budget::charge(work,size+actual.len+1)){return .err(1002);}if(!bytes::equal(saved[c.offset..c.offset+size],actual)){return .err(1001);}c.offset+=size;
    let length=wire::count_from(saved,ref mut c,wire::byte_limit(),1,work);if(c.code!=0){return .err(c.code);}let encoded=saved[c.offset..c.offset+length];c.offset+=length;
    let ended=wire::end(saved,ref mut c,work);if(ended!=0){return .err(ended);}
    let current=switch(source::produce(actual,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let expected=switch(packet(ref current,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    if(!budget::charge(work,encoded.len+wire::content(ref expected).len+1)){return .err(1002);}if(!bytes::equal(encoded,wire::content(ref expected))){return .err(1001);}
    let graph=switch(artifact::decode(encoded,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    return .ok(Restored{.current,.graph});
}
