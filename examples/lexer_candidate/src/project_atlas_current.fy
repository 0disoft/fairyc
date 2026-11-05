module project_atlas_current;
import project_atlas_wire as wire;
import project_atlas_validate as validate;
import project_atlas_model as model;
import project_json_tree as json;
import std::bytes as bytes;
import std::text as text;

// An explicit independent caller expectation is required before graph joining.
// It must come from verified current retained source/compiler/interface facts,
// never from the candidate map's own header or from an unauthenticated cache.
pub resource Ready{validate::Partition partition;cleanup{}}
pub fn ref validate::Partition partition(ref Ready r){return ref r.partition;}
pub fn Result<Ready,u32> read(bytes::Buffer source,bytes::Buffer expected,ref mut usize meter){return admit(source,expected,false,meter);}
// Independent current expectations are required for generic v3 maps too. The
// source producer, rather than a transported/cache header, must own expectations.
pub fn Result<Ready,u32> read_generics(bytes::Buffer source,bytes::Buffer expected,ref mut usize meter){return admit(source,expected,true,meter);}
fn Result<Ready,u32> admit(bytes::Buffer source,bytes::Buffer expected,bool generics,ref mut usize meter){
 let doc=switch(wire::read(source,4194304,meter)){case .ok(p):{p}case .err(p):{return .err(p.code);}};let partition=if(generics){validate::validate_generics(doc,meter)?}else{validate::validate(doc,meter)?};
 if(bytes::len(ref expected)==0 || bytes::len(ref expected)>4194304 || !text::validate_utf8(bytes::view(ref expected)).is_ok){return .err(8);}
 let current=switch(json::parse(expected,meter)){case .ok(p):{p}case .err(p):{return .err(p.code);}};
 if(json::token(ref current,0).kind!=1){return .err(8);}let t=validate::tree(ref partition);let h=validate::header(ref partition);let root=validate::root(ref partition);
 let names=b"compiler_build|interface_hash|generic_interface_hash|language|package_uid|package_instance|package_coordinate|package_source|package_source_kind|runtime_abi|features|dependencies|package_references|files";
 var usize start=0;var usize at=1;var usize index=0;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let name=names[start..end];if(json::token(ref current,at).kind!=7 || !bytes::equal(json::value(ref current,at),name)){return .err(8);}
 let value=at+2;if(index<10){if(json::token(ref current,value).kind!=7 || !bytes::equal(json::value(ref current,value),wire::string(t,h,name))){return .err(8);}}
 else{let actual=wire::field(t,if(index==13){root}else{h},name);if(!bytes::equal(model::raw(ref current,value),model::raw(t,actual))){return .err(8);}}
 at=json::next(ref current,value);if(json::token(ref current,at).kind==6){at+=1;}index+=1;start=end+1;
 }
 if(json::token(ref current,at).kind!=2){return .err(8);}return .ok(Ready{.partition});
}
