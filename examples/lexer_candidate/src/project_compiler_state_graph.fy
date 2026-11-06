module project_compiler_state_graph;
import module_state_graph_source as authored;
import module_state_graph_source_artifact as artifact;
import module_state_graph_source_identity as identity;
import module_state_graph_program as verified;
import module_project_interfaces as interfaces;
import module_public_type_text as coordinates;
import project_cache_encoding as budget;
import std::vec as v;

// Full admission for the existing explicit single-module graph source family.
// Ordinary function IR is not a replacement for this checked state program.
pub resource Complete{v::Vec<authored::Produced> cold;v::Vec<artifact::Restored> restored;identity::Bound identity;interfaces::Records interfaces;usize owner;cleanup{}}
pub fn ref authored::Produced source(ref Complete p){return if(v::len<authored::Produced>(ref p.cold)==1){v::at<authored::Produced>(ref p.cold,0)}else{artifact::current(v::at<artifact::Restored>(ref p.restored,0))};}
pub fn ref verified::Verified program(ref Complete p){return if(v::len<authored::Produced>(ref p.cold)==1){authored::program(source(p))}else{artifact::graph(v::at<artifact::Restored>(ref p.restored,0))};}
pub fn ref identity::Bound body_identity(ref Complete p){return ref p.identity;}
pub fn ref interfaces::Records public_interfaces(ref Complete p){return ref p.interfaces;}
pub fn usize owner(ref Complete p){return p.owner;}
fn u32 mapped(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn Result<Complete,u32> finish(v::Vec<authored::Produced> cold,v::Vec<artifact::Restored> restored,ref coordinates::Coordinates table,usize owner,ref mut usize work){
    if(!budget::charge(work,1)){return .err(1002);}let current=if(v::len<authored::Produced>(ref cold)==1){v::at<authored::Produced>(ref cold,0)}else{artifact::current(v::at<artifact::Restored>(ref restored,0))};
    let identity=switch(identity::derive(current,table,owner,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let interfaces=switch(interfaces::build_state_graph(current,owner,work)){case .ok(p):{p}case .err(code):{return .err(mapped(code));}};
    return .ok(Complete{.cold,.restored,.identity,.interfaces,.owner});
}
pub fn Result<Complete,u32> build(ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref mut usize work){
    let current=switch(authored::produce(actual,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var cold=v::empty<authored::Produced>();if(!v::push<authored::Produced>(ref mut cold,current).is_ok){return .err(1003);}return finish(cold,v::empty<artifact::Restored>(),table,owner,work);
}
pub fn Result<Complete,u32> restore(ref u8[] saved,ref u8[] actual,ref coordinates::Coordinates table,usize owner,ref mut usize work){
    let current=switch(artifact::restore(saved,actual,work)){case .ok(p):{p}case .err(code):{return .err(code);}};var restored=v::empty<artifact::Restored>();if(!v::push<artifact::Restored>(ref mut restored,current).is_ok){return .err(1003);}return finish(v::empty<authored::Produced>(),restored,table,owner,work);
}
