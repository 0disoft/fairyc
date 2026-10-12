module project_coordinate_graph;
import project_coordinate_identity as identity;
import project_workspace_graph as graph;
import project_instance_graph as local;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_manifest as manifest;
import project_sdk_bundle as sdk;
import module_public_type_text as public_type;
import candidate_limits as limits;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Owner numbers stay aligned with project_compiler_input. Duplicate coordinate
// text is legitimate for distinct source instances; exact node IDs stay separate.
pub struct Problem{u32 code;usize owner;}
// 1..6 mirror the encoder; 7 rejects inconsistent graph/feature coordinates.
fn Problem problem(u32 code,usize owner){return Problem{.code,.owner};}
fn Result<identity::Identity,u32> node(ref graph::Graph input,usize owner,ref mut usize meter){
    var v::Vec<bytes::Buffer> features=v::empty<bytes::Buffer>();
    if(graph::is_sdk(input,owner)){return identity::compute(sdk::uid(),sdk::version(),ref features,meter);}
    let w=local::input(graph::local_graph(input));let c=workspace::local_catalog(w);let n=graph::node(input,owner);if(n.package_index>=catalog::count(c)){return .err(7);}let m=catalog::package(c,n.package_index);
    if(n.feature_edge==0){if(owner!=0 || n.package_index!=0){return .err(7);}}
    else{let edge=n.feature_edge-1;if(edge>=workspace::edge_count(w) || workspace::edge(w,edge).target!=n.package_index){return .err(7);}var usize id=workspace::feature_first(w,edge);while(id!=0){let value=workspace::feature(w,edge,id);if(value.len>=16777216 || !work::charge(meter,value.len+1)){return .err(6);}let copied=switch(bytes::copy(value)){case .ok(b):{b}case .err(_):{return .err(5);}};if(!v::push<bytes::Buffer>(ref mut features,copied).is_ok){return .err(5);}id=workspace::next_feature(w,edge,id);}}
    return identity::compute(manifest::package_value(m,b"uid"),manifest::package_value(m,b"version"),ref features,meter);
}
pub fn Result<public_type::Coordinates,Problem> collect(ref graph::Graph input,ref mut usize meter){
    let count=graph::node_count(input);if(count==0){return .err(problem(7,0));}if(count>limits::source_files() || !work::charge(meter,count+1)){return .err(problem(6,0));}var bytes::Buffer names=bytes::empty();var v::Vec<public_type::Coordinate> rows=v::empty<public_type::Coordinate>();var usize owner=0;
    while(owner<count){let key=switch(node(input,owner,meter)){case .ok(value):{value}case .err(code):{return .err(problem(code,owner));}};let id=identity::digest(ref key);if(!identity::valid_text(id)){return .err(problem(7,owner));}if(!work::charge(meter,id.len+1)){return .err(problem(6,owner));}let start=bytes::len(ref names);if(!bytes::append(ref mut names,id).is_ok){return .err(problem(5,owner));}if(!v::push<public_type::Coordinate>(ref mut rows,public_type::Coordinate{.owner,.start,.end=bytes::len(ref names)}).is_ok){return .err(problem(5,owner));}owner+=1;}
    return .ok(public_type::assemble(names,rows));
}
