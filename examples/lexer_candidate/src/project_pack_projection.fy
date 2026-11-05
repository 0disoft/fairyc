module project_pack_projection;
import project_catalog_json as w;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_json_tree as json;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Explicit checked-wire -> snapshot projection. Wire names/order and Go query
// struct names/order are different; raw map JSON is never a query record.
fn Result<bool,u32> fields(ref mut bytes::Buffer out,ref json::Tree t,usize row,ref u8[] names,ref mut usize meter){
    var usize start=0;while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}var usize equal=start;while(equal<end && names[equal]!=61){equal+=1;}if(equal==end){return .err(9);}
        if(start>0){discard w::raw(out,b",",meter)?;}discard w::pair(out,names[start..equal],wire::string(t,row,names[equal+1..end]),meter)?;start=end+1;
    }return .ok(true);
}
pub fn Result<bytes::Buffer,u32> header(ref validate::Partition p,ref mut usize meter){
    let t=validate::tree(p);let h=validate::header(p);var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;
    discard fields(ref mut out,t,h,b"LogicalSchema=logical_schema|SnapshotModel=snapshot_model|CompilerBuild=compiler_build|CompilerBuildAlgorithm=compiler_build_algorithm|Language=language|PackageUID=package_uid|PackageInstance=package_instance|PackageCoordinate=package_coordinate|PackageSource=package_source|PackageSourceKind=package_source_kind|RuntimeABI=runtime_abi|TargetScope=target_scope|Target=target|PackageInstanceAlgorithm=package_instance_algorithm|PackageCoordinateAlgorithm=package_coordinate_algorithm|PackageSourceAlgorithm=package_source_algorithm|SourceFileAlgorithm=source_file_algorithm|LineMapAlgorithm=line_map_algorithm|SymbolAlgorithm=symbol_algorithm|InterfaceAlgorithm=interface_algorithm|InterfaceHash=interface_hash|GenericInterfaceAlgorithm=generic_interface_algorithm|GenericInterfaceHash=generic_interface_hash",meter)?;
    discard w::raw(ref mut out,b",\"Features\":",meter)?;discard w::strings(ref mut out,t,wire::field(t,h,b"features"),false,meter)?;
    discard w::raw(ref mut out,b",\"Dependencies\":",meter)?;discard w::strings(ref mut out,t,wire::field(t,h,b"dependencies"),false,meter)?;
    let refs=model::rows(t,wire::field(t,h,b"package_references"),33,meter)?;discard w::raw(ref mut out,b",\"PackageReferences\":",meter)?;
    if(v::len<usize>(ref refs)==0){discard w::raw(ref mut out,b"[]",meter)?;}else{discard w::raw(ref mut out,b"[",meter)?;var usize i=0;
        while(i<v::len<usize>(ref refs)){if(i>0){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"{",meter)?;discard fields(ref mut out,t,*v::at<usize>(ref refs,i),b"UID=uid|Instance=instance|Coordinate=coordinate",meter)?;discard w::raw(ref mut out,b"}",meter)?;i+=1;}discard w::raw(ref mut out,b"]",meter)?;
    }
    discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"GenerationStatus",wire::string(t,h,b"generation_status"),meter)?;
    discard w::raw(ref mut out,b",\"Omissions\":",meter)?;discard w::strings(ref mut out,t,wire::field(t,h,b"omissions"),false,meter)?;
    discard w::raw(ref mut out,b",\"Counts\":{",meter)?;let counts=wire::field(t,h,b"counts");let names=b"Files=files|Modules=modules|Symbols=symbols|Functions=functions|Specializations=specializations|Relations=relations|Completeness=completeness";var usize start=0;
    while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}var usize equal=start;while(names[equal]!=61){equal+=1;}if(start>0){discard w::raw(ref mut out,b",",meter)?;}discard w::string(ref mut out,names[start..equal],meter)?;discard w::raw(ref mut out,b":",meter)?;discard w::number(ref mut out,wire::integer(t,counts,names[equal+1..end]),meter)?;start=end+1;}
    discard w::raw(ref mut out,b"}}",meter)?;return .ok(out);
}
fn ref u8[] kind(usize index){let ref u8[] names=b"enumforeign-functionfunctionresourcestruct";return if(index==0){names[..4]}else if(index==1){names[4..20]}else if(index==2){names[20..28]}else if(index==3){names[28..36]}else{names[36..42]};}
pub fn Result<bytes::Buffer,u32> package(ref validate::Partition p,ref mut usize meter){
    var usize[5] counts=[0;5];let t=validate::tree(p);var usize i=0;
    while(i<validate::symbol_count(p)){if(!work::charge(meter,1)){return .err(3);}let declaration=validate::symbol_declaration(p,i);let name=wire::string(t,wire::field(t,declaration,b"key"),b"kind");var usize k=0;
        while(k<5 && !bytes::equal(name,kind(k))){k+=1;}if(k==5){return .err(9);}counts[k]+=1;i+=1;
    }
    let h=header(p,meter)?;var out=bytes::empty();discard w::raw(ref mut out,b"{\"header\":",meter)?;discard w::raw(ref mut out,bytes::view(ref h),meter)?;discard w::raw(ref mut out,b",\"declaration_kinds\":[",meter)?;var bool first=true;i=0;
    while(i<5){if(counts[i]>0){if(!first){discard w::raw(ref mut out,b",",meter)?;}first=false;discard w::raw(ref mut out,b"{\"kind\":",meter)?;discard w::string(ref mut out,kind(i),meter)?;discard w::raw(ref mut out,b",\"count\":",meter)?;discard w::number(ref mut out,counts[i],meter)?;discard w::raw(ref mut out,b"}",meter)?;}i+=1;}
    discard w::raw(ref mut out,b"]}",meter)?;return .ok(out);
}
pub fn Result<bytes::Buffer,u32> completeness(ref validate::Partition p,usize index,ref mut usize meter){
    let t=validate::tree(p);let row=validate::completeness(p,index);var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;
    discard fields(ref mut out,t,row,b"Relation=relation|Status=status",meter)?;
    discard w::raw(ref mut out,b",\"Resolved\":",meter)?;discard w::number(ref mut out,wire::integer(t,row,b"resolved"),meter)?;
    discard w::raw(ref mut out,b",\"External\":",meter)?;discard w::number(ref mut out,wire::integer(t,row,b"external"),meter)?;
    discard w::raw(ref mut out,b",\"Unknown\":",meter)?;discard w::number(ref mut out,wire::integer(t,row,b"unknown"),meter)?;
    discard w::raw(ref mut out,b",\"Reasons\":",meter)?;discard w::strings(ref mut out,t,wire::field(t,row,b"reasons"),true,meter)?;discard w::raw(ref mut out,b"}",meter)?;return .ok(out);
}
pub fn Result<bytes::Buffer,u32> module_record(ref validate::Partition p,usize row,ref mut usize meter){
    var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;discard fields(ref mut out,validate::tree(p),row,b"PackageInstance=package_instance|Path=path|Source=source",meter)?;discard w::raw(ref mut out,b"}",meter)?;return .ok(out);
}
pub fn Result<bool,u32> omission(ref mut bytes::Buffer out,ref u8[] code,ref u8[] section,usize count,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;discard w::pair(out,b"code",code,meter)?;discard w::raw(out,b",",meter)?;discard w::pair(out,b"section",section,meter)?;
    if(count>0){discard w::raw(out,b",\"count\":",meter)?;discard w::number(out,count,meter)?;}return w::raw(out,b",\"count_known\":true}",meter);
}
pub fn Result<bool,u32> identity(ref mut bytes::Buffer out,ref validate::Partition p,ref mut usize meter){
    discard w::raw(out,b"{",meter)?;discard fields(out,validate::tree(p),validate::header(p),b"uid=package_uid|instance=package_instance|coordinate=package_coordinate|source=package_source",meter)?;return w::raw(out,b"}",meter);
}
pub fn Result<bool,u32> compatibility(ref mut bytes::Buffer out,ref validate::Partition p,ref mut usize meter){
    let t=validate::tree(p);let h=validate::header(p);discard w::raw(out,b"{",meter)?;discard fields(out,t,h,b"compiler_build=compiler_build|language=language|interface_hash=interface_hash",meter)?;
    let generic=wire::string(t,h,b"generic_interface_hash");if(generic.len>0){discard w::raw(out,b",",meter)?;discard w::pair(out,b"generic_interface_hash",generic,meter)?;}
    discard w::raw(out,b",",meter)?;discard w::pair(out,b"generation",wire::string(t,h,b"generation_status"),meter)?;discard w::raw(out,b",\"map_omissions\":",meter)?;discard w::strings(out,t,wire::field(t,h,b"omissions"),false,meter)?;return w::raw(out,b"}",meter);
}
