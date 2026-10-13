module project_cache_workspace;
import project_compiler_core as core;
import module_project_identified_ir as checked;
import generic_package_bindings as stamps;
import project_compiler_input as input;
import project_module_visibility as visibility;
import project_workspace_graph as graph;
import project_coordinate_graph as coordinates;
import module_public_type_text as table;
import project_instance_graph as local;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_sdk_bundle as sdk;
import project_identity_bytes as binary;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

// Detached workspace facts, authenticated against one opaque complete result.
// Dependencies use this result's owner numbers, never persisted cache identity.
// A later key builder must select their coordinate/interface identities.
pub resource Package{bytes::Buffer instance;bytes::Buffer coordinate;bytes::Buffer source;v::Vec<bytes::Buffer> features;v::Vec<usize> dependencies;cleanup{}}
pub resource Facts{v::Vec<Package> packages;bytes::Buffer sdk_instance;cleanup{}}
pub fn usize count(ref Facts p){return v::len<Package>(ref p.packages);}
pub fn ref Package at(ref Facts p,usize owner){assert(owner<count(p));return v::at<Package>(ref p.packages,owner);}
pub fn ref u8[] instance(ref Package p){return bytes::view(ref p.instance);}
pub fn ref u8[] coordinate(ref Package p){return bytes::view(ref p.coordinate);}
pub fn ref u8[] source(ref Package p){return bytes::view(ref p.source);}
pub fn usize feature_count(ref Package p){return v::len<bytes::Buffer>(ref p.features);}
pub fn ref u8[] feature(ref Package p,usize index){return bytes::view(v::at<bytes::Buffer>(ref p.features,index));}
pub fn usize dependency_count(ref Package p){return v::len<usize>(ref p.dependencies);}
pub fn usize dependency(ref Package p,usize index){return *v::at<usize>(ref p.dependencies,index);}
// Empty means this graph has no SDK node. It is not a usable cache-key digest.
pub fn ref u8[] sdk_instance(ref Facts p){return bytes::view(ref p.sdk_instance);}
fn Result<bytes::Buffer,u32> copy(ref u8[] value,ref u8[] prefix,ref mut usize work){
    switch(binary::decode(value,prefix,true,work)){case .ok(_):{}case .err(code):{return .err(code);}};
    if(!budget::charge(work,value.len+1)){return .err(10);}return switch(bytes::copy(value)){case .ok(p):{.ok(p)}case .err(_):{.err(11)}};
}
// Fresh projection is owned by the loaded resolved graph; no caller hashes or
// import records can replace its bytes, owners, providers or coordinates.
fn u32 current(ref visibility::Resolved loaded,ref core::Complete complete,ref table::Coordinates ids,ref mut usize work){
    let projected=switch(input::project(loaded,work)){case .ok(p):{p}case .err(p):{return if(p.code==1){10}else if(p.code==2){11}else{9};}};
    let fresh=switch(stamps::collect(ref projected,ids,work)){case .ok(p):{p}case .err(p):{return p.code;}};
    let old=checked::source_provenance(core::checked_program(complete));if(stamps::count(old)!=stamps::count(ref fresh) || stamps::import_count(old)!=stamps::import_count(ref fresh)){return 9;}var usize i=0;
    while(i<stamps::count(old)){if(!budget::charge(work,9)){return 10;}let a=stamps::module_at(old,i);let b=stamps::module_at(ref fresh,i);
        if(a.owner!=b.owner || a.first!=b.first || a.count!=b.count){return 9;}if(a.source_size!=b.source_size){return 12;}var usize j=0;while(j<8){if(a.digest[j]!=b.digest[j]){return 12;}j+=1;}
        let an=stamps::text(old,a.namespace);let bn=stamps::text(ref fresh,b.namespace);let ap=stamps::text(old,a.path);let bp=stamps::text(ref fresh,b.path);if(!budget::charge(work,an.len+bn.len+ap.len+bp.len+1)){return 10;}if(!bytes::equal(an,bn) || !bytes::equal(ap,bp)){return 9;}i+=1;
    }
    i=0;while(i<stamps::import_count(old)){if(!budget::charge(work,1)){return 10;}let a=stamps::import_at(old,i);let b=stamps::import_at(ref fresh,i);if(a.module_index!=b.module_index || a.declaration!=b.declaration || a.target!=b.target){return 9;}
        let aa=stamps::text(old,a.alias);let ba=stamps::text(ref fresh,b.alias);let ab=stamps::text(old,a.binding);let bb=stamps::text(ref fresh,b.binding);if(!budget::charge(work,aa.len+ba.len+ab.len+bb.len+1)){return 10;}if(!bytes::equal(aa,ba) || !bytes::equal(ab,bb)){return 9;}i+=1;
    }return 0;
}
fn Result<v::Vec<bytes::Buffer>,u32> features(ref graph::Graph g,usize owner,ref mut usize work){
    var v::Vec<bytes::Buffer> out=v::empty<bytes::Buffer>();if(graph::is_sdk(g,owner)){return .ok(out);}let w=local::input(graph::local_graph(g));let n=graph::node(g,owner);
    if(n.feature_edge==0){if(owner!=0 || n.package_index!=0){return .err(9);}return .ok(out);}let edge=n.feature_edge-1;if(edge>=workspace::edge_count(w) || workspace::edge(w,edge).target!=n.package_index){return .err(9);}
    var usize id=workspace::feature_first(w,edge);while(id!=0){let text=workspace::feature(w,edge,id);if(!budget::charge(work,text.len+1)){return .err(10);}let value=switch(bytes::copy(text)){case .ok(p):{p}case .err(_):{return .err(11);}};if(!v::push<bytes::Buffer>(ref mut out,value).is_ok){return .err(11);}id=workspace::next_feature(w,edge,id);}return .ok(out);
}
fn Result<v::Vec<usize>,u32> dependencies(ref graph::Graph g,usize owner,ref mut usize work){
    let n=graph::node(g,owner);var bool[512] seen=[false;512];var usize i=0;while(i<n.count){if(!budget::charge(work,1)){return .err(10);}let e=graph::edge(g,n.first+i);if(e.owner!=owner || e.target>=graph::node_count(g) || e.target==owner){return .err(9);}seen[e.target]=true;i+=1;}
    var v::Vec<usize> out=v::empty<usize>();i=0;while(i<graph::node_count(g)){if(!budget::charge(work,1)){return .err(10);}if(seen[i] && !v::push<usize>(ref mut out,i).is_ok){return .err(11);}i+=1;}return .ok(out);
}
pub fn Result<Facts,u32> collect(ref visibility::Resolved loaded,ref core::Complete complete,ref mut usize work){
    if(!budget::charge(work,0)){return .err(10);}let g=visibility::workspace_graph(loaded);let total=graph::node_count(g);if(total==0 || total>512){return .err(9);}
    let ids=switch(coordinates::collect(g,work)){case .ok(p):{p}case .err(p):{return .err(if(p.code==6){10}else if(p.code==5){11}else{9});}};
    let code=current(loaded,complete,ref ids,work);if(code!=0){return .err(code);}var v::Vec<Package> packages=v::empty<Package>();var bytes::Buffer sdk_instance=bytes::empty();var usize owner=0;
    while(owner<total){if(!budget::charge(work,1)){return .err(10);}let id=switch(copy(graph::node_id(g,owner),b"fypi1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(table::coordinate_at(ref ids,owner).owner!=owner){return .err(9);}let coordinate=switch(copy(table::coordinate_text(ref ids,owner),b"fypc1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        let digest=if(graph::is_sdk(g,owner)){sdk::digest(graph::standard_library(g))}else{catalog::digest(workspace::local_catalog(local::input(graph::local_graph(g))),graph::node(g,owner).package_index)};
        let source=switch(copy(digest,b"fysrc1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};let selected=switch(features(g,owner,work)){case .ok(p):{p}case .err(code):{return .err(code);}};let deps=switch(dependencies(g,owner,work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(graph::is_sdk(g,owner)){sdk_instance=switch(copy(graph::node_id(g,owner),b"fypi1-sha256-",work)){case .ok(p):{p}case .err(code):{return .err(code);}};}
        if(!v::push<Package>(ref mut packages,Package{.instance=id,.coordinate,.source,.features=selected,.dependencies=deps}).is_ok){return .err(11);}owner+=1;
    }return .ok(Facts{.packages,.sdk_instance});
}
