module foreign_declarations;
import c_abi_types as c_types;
import pointer_types as pointers;
import syntax as syntax;
import type_state as state;
import ffi_symbols as symbols;
import nominal_identity as nominal;
import repr_c_declarations as repr;
import std::bytes as bytes;
import sdk_scopes as sdk;
pub struct Info {u32 code;usize start;usize end;usize link_start;usize link_end;}
fn u32 scalar_type(ref syntax::Parsed p,usize root){
    if(root==0){return 0;}let u32 c_type=c_types::resolve(p,root);if(c_types::valid(c_type)){return c_type;}if(syntax::type_at(p,root).kind==11 || syntax::type_at(p,root).kind==12){return pointers::resolve(p,root);}if(syntax::type_at(p,root).kind!=1){return 0;}let text=syntax::type_text(p,root);
    if(bytes::equal(text,b"i32") || bytes::equal(text,b"i64") || bytes::equal(text,b"u8") || bytes::equal(text,b"u32") || bytes::equal(text,b"usize") || bytes::equal(text,b"void")){return state::primitive(p,root);}return 0;
}
pub fn bool result_type(ref syntax::Parsed p,usize root){
    if(symbols::scalar(scalar_type(p,root),true)){return true;}
    if(root==0 || syntax::type_at(p,root).kind!=1){return false;}
    let u32 bound=nominal::bound(p,root);let u32 type=if(bound!=0){bound}else{nominal::named(p,syntax::type_text(p,root))};
    return nominal::valid(type) && repr::read(p,(type-256) as usize).repr_c;
}
pub fn Info read(ref syntax::Parsed p,usize index){return read_result(p,index,result_type(p,syntax::declaration_at(p,index).result_type));}
// The source-set consumer resolves an imported result to its actual provider.
// This temporary admission never replaces owned HIR/MIR ABI validation.
pub fn Info read_result(ref syntax::Parsed p,usize index,bool admitted_result){
    let d=syntax::declaration_at(p,index);var Info info=Info{.code=0,.start=d.start,.end=d.end,.link_start=d.name_start,.link_end=d.name_end};
    if(d.kind!=9 || d.unsafe_end==0 || d.body!=0 || d.generic_parameters!=0 || d.owner_end!=0){info.code=3;return info;}
    if(!admitted_result){info.code=5102;info.start=syntax::type_at(p,d.result_type).start;info.end=syntax::type_at(p,d.result_type).end;return info;}
    var usize i=d.entry_start;while(i<d.entry_start+d.entry_count){let e=syntax::entry_at(p,i);if(e.kind==1){if(!symbols::scalar(scalar_type(p,e.type_root),false)){info.code=5102;info.start=e.type_start;info.end=e.type_end;return info;}
        var usize j=d.entry_start;while(j<i){let old=syntax::entry_at(p,j);if(old.kind==1 && bytes::equal(syntax::source_view(p)[old.name_start..old.name_end],syntax::source_view(p)[e.name_start..e.name_end])){info.code=2;info.start=e.name_start;info.end=e.name_end;return info;}j+=1;}}i+=1;}
    var bool seen=false;i=0;while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"link_name")){
        if(seen || a.argument_count!=1){info.code=5106;info.start=a.start;info.end=a.end;return info;}seen=true;
        let arg=syntax::attribute_argument_at(p,a.argument_start);let raw=syntax::attribute_argument_text(p,a.argument_start);
        if(raw.len<3 || raw[0]!=34 || raw[raw.len-1]!=34){info.code=5106;info.start=a.start;info.end=a.end;return info;}
        info.link_start=arg.start+1;info.link_end=arg.end-1;
    }i+=1;}
    if(!symbols::valid(syntax::source_view(p)[info.link_start..info.link_end]) && !sdk::parsed_foreign(p,index,info.link_start,info.link_end)){info.code=5106;info.start=info.link_start;info.end=info.link_end;}return info;
}
