module module_source_interface;
import generic_source_dependency_model as dependencies;
import generic_source_catalog as catalog;
import generic_package_bindings as bindings;
import module_public_templates as templates;
import module_generic_interface as generic;
import module_public_interface_extract as extract;
import module_public_type_text as coordinates;
import module_unit_interfaces as typed;
import module_signatures as signatures;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// These source-interface facts do not admit executable artifacts. The owning
// compilation path must gate publication on completed body/Flow verification.
fn u32 current(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,ref dependencies::Graph input,ref mut usize work){
    let packages=catalog::packages(dependencies::templates(input));let count=bindings::count(packages);if(count!=v::len<syntax::Parsed>(sources) || count!=signatures::module_count(metadata)){return 9;}
    var usize[512] positions=[0;512];var usize i=0;
    while(i<count){if(!budget::charge(work,1)){return 10;}let owned_module=bindings::module_at(packages,i);if(owned_module.owner>=512 || owned_module.owner!=signatures::module_owner(metadata,i)){return 9;}let raw=syntax::source_view(v::at<syntax::Parsed>(sources,i));if(!budget::charge(work,raw.len+1)){return 10;}if(raw.len!=owned_module.source_size || !signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return 12;}let digest=signatures::source_digest(metadata,i);var usize j=0;while(j<8){if(digest[j]!=owned_module.digest[j]){return 12;}j+=1;}
        if(positions[owned_module.owner]==0){var usize measured=0;let code=coordinates::validate_owner(table,owned_module.owner,ref mut measured);if(!budget::charge(work,measured)){return 10;}if(code!=0){return code;}j=0;while(j<coordinates::coordinate_count(table)){if(!budget::charge(work,1)){return 10;}if(coordinates::coordinate_at(table,j).owner==owned_module.owner){positions[owned_module.owner]=j+1;break;}j+=1;}}
        if(positions[owned_module.owner]==0){return 9;}let value=coordinates::coordinate_text(table,positions[owned_module.owner]-1);let namespace=bindings::text(packages,owned_module.namespace);let prefix=b"package_coordinate_v1_";
        if(!budget::charge(work,value.len+namespace.len+1)){return 10;}if(namespace.len!=prefix.len+64 || !bytes::equal(namespace[..prefix.len],prefix) || !bytes::equal(namespace[prefix.len..],value[13..])){return 9;}i+=1;
    }i=0;while(i<bindings::import_count(packages)){if(!budget::charge(work,1)){return 10;}let row=bindings::import_at(packages,i);var usize measured=0;let target=signatures::import_target_measured(metadata,row.module_index,bindings::text(packages,row.alias),ref mut measured);if(!budget::charge(work,measured)){return 10;}if(target.code!=0 || target.item!=row.target+1){return 9;}i+=1;}return 0;
}
pub fn Result<bytes::Buffer,u32> hash(ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref coordinates::Coordinates table,ref dependencies::Graph input,usize owner,ref mut usize work){
    let code=current(sources,metadata,table,input,work);if(code!=0){return .err(code);}let public=switch(templates::collect(input,owner,work)){case .ok(value):{value}case .err(code):{return .err(code);}};
    var usize measured=0;let ordinary=switch(extract::extract(sources,metadata,graph,table,owner,ref mut measured)){case .ok(value):{value}case .err(code):{discard budget::charge(work,measured);return .err(code);}};if(!budget::charge(work,measured)){return .err(10);}if(extract::has_templates(ref ordinary)!=(generic::count(ref public)!=0)){return .err(9);}
    measured=0;let digest=switch(generic::hash(signatures::state_layouts(metadata),owner,extract::descriptor(ref ordinary),ref public,ref mut measured)){case .ok(value):{value}case .err(code):{discard budget::charge(work,measured);return .err(code);}};if(!budget::charge(work,measured)){return .err(10);}return .ok(digest);
}
