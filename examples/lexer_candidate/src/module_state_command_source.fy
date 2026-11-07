module module_state_command_source;
import project_compiler_core as core;
import module_state_graph_source as source;
import module_state_graph_model as graph;
import module_state_layout as layout;
import project_coordinate_identity as identity;
import module_public_type_text as coordinates;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// Fresh source authority only. The admitted layout owns the UID; neither a
// caller coordinate nor a persisted artifact can select this command's identity.
pub fn Result<core::Complete,u32> prepare(ref u8[] actual,ref mut usize work){
    let produced=switch(source::produce(actual,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
    let uid=layout::uid(source::layouts(ref produced),graph::header(source::graph(ref produced)).layout);
    let features=v::empty<bytes::Buffer>();
    let coordinate=switch(identity::compute(uid,b"experimental-state-graph-source/1",ref features,work)){case .ok(p):{p}case .err(code):{return .err(if(code==6){1002}else if(code==5){1003}else{1001});}};
    let raw=identity::digest(ref coordinate);
    if(!budget::charge(work,raw.len+2)){return .err(1002);}
    let names=switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{return .err(1003);}};
    var rows=v::empty<coordinates::Coordinate>();
    if(!v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=raw.len}).is_ok){return .err(1003);}
    let table=coordinates::assemble(names,rows);
    // Re-admit the exact source into a sealed Complete using the SAME ledger.
    return switch(core::build_complete_state_graph(actual,ref table,0,work)){case .ok(p):{.ok(p)}case .err(problem):{.err(problem.code)}};
}
