module module_const_eval_declarations;
import syntax as syntax;
import type_state as scalars;
import std::bytes as bytes;

// Declaration admission is separate from expression admission and evaluation.
// IDs belong to the exact retained syntax, including the explicit return.
pub struct Info {bool annotated;u32 code;usize start;usize end;usize expression;u32 result;usize parameters;}
pub fn bool scalar(u32 value){return value==1 || value==2 || value==3 || value==4 || value==6;}
fn bool scalar_type(ref syntax::Parsed source,usize root){return root!=0 && syntax::type_at(source,root).kind==1 && scalar(scalars::primitive(source,root));}
fn Info invalid(Info value,usize start,usize end){var Info result=value;result.code=4301;result.start=start;result.end=end;return result;}
pub fn Info read(ref syntax::Parsed source,usize declaration){
    let d=syntax::declaration_at(source,declaration);
    var Info info=Info{.annotated=false,.code=0,.start=d.name_start,.end=d.name_end,.expression=0,.result=0,.parameters=0};
    var usize i=0;
    while(i<d.attribute_count){let a=syntax::attribute_at(source,d.attribute_start+i);
        if(bytes::equal(syntax::source_view(source)[a.name_start..a.name_end],b"const_eval")){
            if(info.annotated || a.argument_count!=0 || d.kind!=3){return invalid(info,a.start,a.end);}info.annotated=true;
        }i+=1;
    }
    i=0;while(i<d.cleanup_attribute_count){let a=syntax::attribute_at(source,d.cleanup_attribute_start+i);if(bytes::equal(syntax::source_view(source)[a.name_start..a.name_end],b"const_eval")){return invalid(info,a.start,a.end);}i+=1;}
    if(!info.annotated){return info;}
    if(d.public || d.unsafe_end!=0 || d.owner_end!=0 || d.generic_parameters!=0 || !scalar_type(source,d.result_type) || d.body==0){return invalid(info,d.name_start,d.name_end);}
    i=0;while(i<d.attribute_count){let a=syntax::attribute_at(source,d.attribute_start+i);if(bytes::equal(syntax::source_view(source)[a.name_start..a.name_end],b"effects")){return invalid(info,d.name_start,d.name_end);}i+=1;}
    let body=syntax::statement_at(source,d.body);
    if(body.kind!=1 || body.expression!=0 || syntax::block_statement_count(source,d.body)!=1){return invalid(info,d.name_start,d.name_end);}
    let returned=syntax::statement_at(source,syntax::block_statement(source,d.body,0));
    if(returned.kind!=3 || returned.expression==0){return invalid(info,d.name_start,d.name_end);}
    i=d.entry_start;while(i<d.entry_start+d.entry_count){let parameter=syntax::entry_at(source,i);if(parameter.kind==1){if(!scalar_type(source,parameter.type_root)){return invalid(info,d.name_start,d.name_end);}info.parameters+=1;}i+=1;}
    info.expression=returned.expression;info.result=scalars::primitive(source,d.result_type);return info;
}
