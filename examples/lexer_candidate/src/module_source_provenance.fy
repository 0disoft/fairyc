module module_source_provenance;
import generic_package_bindings as bindings;
import module_signatures as signatures;
import module_public_type_text as coordinates;
import module_state_layout as phase;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Metadata checking has one independent bounded history. Source/coordinate
// checking folds that history into the existing 16MiB caller ledger once.
// 9 rejects owner/import/coordinate drift, 12 stale exact source bytes.
pub fn u32 match_metadata(ref bindings::Bindings input,ref signatures::Program metadata,ref mut usize work){
    let count=bindings::count(input);if(count==0 || count>512 || count!=signatures::module_count(metadata)){return 9;}var usize i=0;
    while(i<count){if(!phase::charge(work,9)){return 10;}let stamp=bindings::module_at(input,i);if(stamp.owner>=512 || stamp.owner!=signatures::module_owner(metadata,i)){return 9;}let digest=signatures::source_digest(metadata,i);var usize j=0;while(j<8){if(stamp.digest[j]!=digest[j]){return 12;}j+=1;}i+=1;}
    i=0;while(i<bindings::import_count(input)){let row=bindings::import_at(input,i);if(row.module_index>=count || row.target>=count){return 9;}var usize measured=0;let target=signatures::import_target_measured(metadata,row.module_index,bindings::text(input,row.alias),ref mut measured);if(!phase::charge(work,measured+1)){return 10;}if(target.code!=0 || target.item!=row.target+1){return 9;}i+=1;}return 0;
}
fn u32 coordinate(ref bindings::Bindings input,bindings::Module stamp,ref coordinates::Coordinates table,ref mut usize work){
    var usize measured=0;let code=coordinates::validate_owner(table,stamp.owner,ref mut measured);if(!budget::charge(work,measured)){return 10;}if(code!=0){return code;}var usize i=0;var usize found=coordinates::coordinate_count(table);
    while(i<coordinates::coordinate_count(table)){if(!budget::charge(work,1)){return 10;}if(coordinates::coordinate_at(table,i).owner==stamp.owner){found=i;break;}i+=1;}if(found==coordinates::coordinate_count(table)){return 9;}let value=coordinates::coordinate_text(table,found);let namespace=bindings::text(input,stamp.namespace);let prefix=b"package_coordinate_v1_";
    if(!budget::charge(work,value.len+namespace.len+1)){return 10;}return if(namespace.len!=prefix.len+64 || !bytes::equal(namespace[..prefix.len],prefix) || !bytes::equal(namespace[prefix.len..],value[13..])){9}else{0};
}
pub fn u32 current(ref bindings::Bindings input,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize work){
    if(!budget::charge(work,0)){return 10;}var usize measured=0;let checked=match_metadata(input,metadata,ref mut measured);if(!budget::charge(work,measured)){return 10;}if(checked!=0){return checked;}if(bindings::count(input)!=v::len<syntax::Parsed>(sources)){return 9;}var bool[512] seen=[false;512];var usize i=0;
    while(i<bindings::count(input)){let stamp=bindings::module_at(input,i);let parsed=v::at<syntax::Parsed>(sources,i);let raw=syntax::source_view(parsed);if(!budget::charge(work,raw.len+1)){return 10;}if(raw.len!=stamp.source_size || !signatures::source_matches(metadata,i,parsed)){return 12;}if(stamp.owner>=512){return 9;}if(!seen[stamp.owner]){let code=coordinate(input,stamp,table,work);if(code!=0){return code;}seen[stamp.owner]=true;}i+=1;}return 0;
}
