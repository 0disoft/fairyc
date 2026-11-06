module module_state_graph_program;
import module_state_graph_model as m;
import module_state_graph_verify as core;
import module_state_layout as state;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

pub struct Name{usize start;usize end;}
pub resource Program{m::Graph root;v::Vec<m::Graph> helpers;v::Vec<Name> names;bytes::Buffer names_data;cleanup{}}
pub resource Verified{Program program;state::Plan layouts;cleanup{}}
pub fn Program assemble(m::Graph root,v::Vec<m::Graph> helpers,v::Vec<Name> names,bytes::Buffer names_data){return Program{.root,.helpers,.names,.names_data};}
pub fn usize helper_limit(){return 32;}
pub fn ref Program program(ref Verified proof){return ref proof.program;}
pub fn ref m::Graph root(ref Verified proof){return ref proof.program.root;}
pub fn usize helper_count(ref Verified proof){return v::len<m::Graph>(ref proof.program.helpers);}
pub fn ref m::Graph helper(ref Verified proof,usize index){return v::at<m::Graph>(ref proof.program.helpers,index);}
pub fn ref state::Plan layouts(ref Verified proof){return ref proof.layouts;}
pub fn ref u8[] helper_name(ref Verified proof,usize index){let span=*v::at<Name>(ref proof.program.names,index);return bytes::view(ref proof.program.names_data)[span.start..span.end];}
fn bool first(u8 c){return c==95 || c>=65 && c<=90 || c>=97 && c<=122;}
fn bool logical(ref u8[] name){
    var usize i=0;var usize segments=0;while(i<name.len){if(!first(name[i])){return false;}i+=1;
        while(i<name.len && (first(name[i]) || name[i]>=48 && name[i]<=57)){i+=1;}
        segments+=1;if(i==name.len){return segments>=2;}if(name.len-i<3 || name[i]!=58 || name[i+1]!=58){return false;}i+=2;
    }return false;
}
fn u32 names(ref Program p,ref mut usize work){
    let count=v::len<m::Graph>(ref p.helpers);let data=bytes::view(ref p.names_data);
    if(count>helper_limit() || data.len>m::byte_limit()){return 1002;}if(v::len<Name>(ref p.names)!=count){return 1001;}
    var usize end=0;var usize i=0;while(i<count){if(!budget::charge(work,1)){return 1002;}let span=*v::at<Name>(ref p.names,i);
        if(span.start!=end || span.start>span.end || span.end>data.len){return 1001;}let name=data[span.start..span.end];
        if(!budget::charge(work,name.len+1)){return 1002;}if(!logical(name)){return 1001;}
        if(i!=0){let before=*v::at<Name>(ref p.names,i-1);let previous=data[before.start..before.end];if(!budget::charge(work,previous.len+name.len+1)){return 1002;}if(!budget::before(previous,name)){return 1001;}}
        end=span.end;i+=1;
    }return if(end==data.len){0}else{1001};
}
fn ref m::Graph function(ref Program p,usize index){if(index==0){return ref p.root;}return v::at<m::Graph>(ref p.helpers,index-1);}
// Acyclic module call graph, independently bounded and reachable from its root.
// Context validation below still checks every call's remaining fields/effects.
fn u32 calls(ref Program p,ref mut usize work){
    let count=v::len<m::Graph>(ref p.helpers)+1;let owner=m::header(ref p.root);
    var bool[1089] edges=[false;1089];var usize[33] incoming=[0;33];var bool[33] reached=[false;33];var usize[33] queue=[0;33];var usize total=0;var usize i=0;
    while(i<count){if(!budget::charge(work,1)){return 1002;}let graph=function(p,i);let h=m::header(graph);let size=v::len<m::Operation>(m::operations(graph));
        if(size>m::operation_limit()-total){return 1002;}total+=size;if(h.layout!=owner.layout || h.capability!=owner.capability || h.effects>3){return 1001;}
        var usize j=0;while(j<size){if(!budget::charge(work,1)){return 1002;}let op=*v::at<m::Operation>(m::operations(graph),j);
            if(op.kind==11){if(op.callee==0 || op.callee>=count){return 1001;}let at=i*33+op.callee;if(!edges[at]){edges[at]=true;incoming[op.callee]+=1;}}j+=1;}i+=1;
    }
    queue[0]=0;reached[0]=true;var usize read=0;var usize written=1;
    while(read<written){let source=queue[read];var usize target=1;while(target<count){if(!budget::charge(work,1)){return 1002;}if(edges[source*33+target] && !reached[target]){reached[target]=true;queue[written]=target;written+=1;}target+=1;}read+=1;}
    if(written!=count){return 1001;}read=0;written=0;i=0;while(i<count){if(incoming[i]==0){queue[written]=i;written+=1;}i+=1;}
    while(read<written){let source=queue[read];var usize target=1;while(target<count){if(!budget::charge(work,1)){return 1002;}if(edges[source*33+target]){incoming[target]-=1;if(incoming[target]==0){queue[written]=target;written+=1;}}target+=1;}read+=1;}
    return if(written==count){0}else{1001};
}
pub fn u32 validate(ref Program p,ref state::Plan layouts,ref mut usize work){
    if(!budget::charge(work,1)){return 1002;}let named=names(p,work);if(named!=0){return named;}let topology=calls(p,work);if(topology!=0){return topology;}
    var v::Vec<u32> effects=v::empty<u32>();var usize i=0;while(i<v::len<m::Graph>(ref p.helpers)){if(!budget::charge(work,1)){return 1002;}if(!v::push<u32>(ref mut effects,m::header(v::at<m::Graph>(ref p.helpers,i)).effects).is_ok){return 1003;}i+=1;}
    i=0;while(i<v::len<m::Graph>(ref p.helpers)){let code=core::validate_function(v::at<m::Graph>(ref p.helpers,i),layouts,ref effects,true,true,work);if(code!=0){return code;}i+=1;}
    return core::validate_function(ref p.root,layouts,ref effects,v::len<u32>(ref effects)!=0,false,work);
}
// This seals detached helper/control validity, never current source or storage.
pub fn Result<Verified,u32> verify(Program p,ref state::Plan layouts,ref mut usize work){
    let code=validate(ref p,layouts,work);if(code!=0){return .err(code);}var usize measured=0;
    let owned=switch(state::copy(layouts,ref mut measured)){
        case .ok(value):{if(!budget::charge(work,measured)){return .err(1002);}value}
        case .err(code):{if(!budget::charge(work,measured)){return .err(1002);}return .err(if(code==10){1002}else if(code==11){1003}else{1001});}
    };return .ok(Verified{.program=p,.layouts=owned});
}
