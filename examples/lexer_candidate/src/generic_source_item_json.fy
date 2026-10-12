module generic_source_item_json;
import generic_source_declaration_json as declarations;
import generic_source_documentation as documentation;
import generic_source_expression_json as expressions;
import generic_source_type_json as types;
import generic_source_tokens as tokens;
import generic_source_json as json;
import syntax as syntax;
import std::bytes as bytes;

fn bool import_value(ref syntax::Parsed p,syntax::Declaration d,ref mut bytes::Buffer out,ref mut json::Ledger work){var usize index=tokens::first(p,d.start,work);var usize first=0;var usize last=0;var bool keyword=false;var bool alias=false;
    while(index<syntax::token_count(p)){if(!json::charge(work,1)){return false;}let t=syntax::token_info(p,index);if(t.start>=d.name_start){break;}if(t.start>t.end || t.end>syntax::source_view(p).len){return json::reject(work,9);}if(t.kind!=2){let raw=syntax::source_view(p)[t.start..t.end];
        if(!keyword){if(t.start!=d.start || !bytes::equal(raw,b"import")){return json::reject(work,9);}keyword=true;}
        else if(bytes::equal(raw,b"as")){alias=true;break;}else{if(first==0){first=t.start;}last=t.end;}}
        index+=1;
    }
    if(!keyword || !alias || first==0 || first>=last || last>d.name_start || d.name_start>=d.name_end || d.name_end>d.end){return json::reject(work,9);}
    return json::raw(out,b"{\"Path\":",work) && tokens::path(p,first,last,out,work) && json::raw(out,b",\"Alias\":",work) && tokens::text(p,d.name_start,d.name_end,out,work) && json::raw(out,b",\"Span\":",work) && json::span(out,d.start,d.end,work) && json::raw(out,b",\"PathSpan\":",work) && json::span(out,first,last,work) && json::raw(out,b",\"AliasSpan\":",work) && json::span(out,d.name_start,d.name_end,work) && json::raw(out,b"}",work);
}
fn bool state_fields(ref syntax::Parsed p,syntax::Declaration d,ref mut bytes::Buffer out,ref mut json::Ledger work){if(d.field_start>syntax::field_count(p) || d.field_count>syntax::field_count(p)-d.field_start){return json::reject(work,9);}if(d.field_count==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}var usize index=0;
    while(index<d.field_count){if(!json::charge(work,1)){return false;}let f=syntax::field_at(p,d.field_start+index);if((f.kind!=1 && f.kind!=2) || f.start>f.end || f.end>d.end || f.name_start<f.start || f.name_start>=f.name_end || f.name_end>f.end || (f.kind==1 && f.key_type!=0)){return json::reject(work,9);}
        if(index!=0 && !json::raw(out,b",",work)){return false;}
        if(!json::raw(out,b"{\"Kind\":",work) || !json::quoted(out,if(f.kind==1){b"cell"}else{b"map"},work) || !json::raw(out,b",\"Name\":",work) || !tokens::text(p,f.name_start,f.name_end,out,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,f.start,f.end,work) || !json::raw(out,b",\"NameSpan\":",work) || !json::span(out,f.name_start,f.name_end,work) || !json::raw(out,b",\"KeyType\":",work)){return false;}
        if(f.kind==2){if(!types::value(p,f.key_type,0,out,work)){return false;}}else if(!json::raw(out,b"null",work)){return false;}
        if(!json::raw(out,b",\"ValueType\":",work) || !types::value(p,f.type_root,0,out,work) || !json::raw(out,b"}",work)){return false;}index+=1;
    }return json::raw(out,b"]",work);
}
// Remaining top-level source AST records required by the original module hash.
// These writers retain syntax facts and do not widen language admission.
pub fn bool value(ref syntax::Parsed p,usize index,ref mut bytes::Buffer out,ref mut json::Ledger work){if(!json::charge(work,1)){return false;}if(syntax::problem_count(p)!=0 || index>=syntax::declaration_count(p)){return json::reject(work,9);}let d=syntax::declaration_at(p,index);if(d.start>d.end || d.end>syntax::source_view(p).len){return json::reject(work,9);}
    if(d.kind==2){return import_value(p,d,out,work);}if(d.kind!=7 && d.kind!=8 && d.kind!=9 && d.kind!=10){return json::reject(work,9);}
    if(!json::raw(out,b"{",work)){return false;}if(d.kind==7 || d.kind==9){if(!json::raw(out,b"\"Name\":",work) || !tokens::text(p,d.name_start,d.name_end,out,work) || !json::raw(out,b",",work)){return false;}}
    if(!json::raw(out,b"\"Public\":",work) || !json::flag(out,d.public,work)){return false;}if(d.kind==9 && !json::raw(out,b",\"ABI\":\"C\"",work)){return false;}
    if(!json::raw(out,b",\"Span\":",work) || !json::span(out,d.start,d.end,work)){return false;}
    if((d.kind==7 || d.kind==9) && (!json::raw(out,b",\"NameSpan\":",work) || !json::span(out,d.name_start,d.name_end,work))){return false;}
    if(d.kind==9){let docs=documentation::declaration(p,index,work);if(work.code!=0 || !json::raw(out,b",\"DocumentationSpan\":",work) || !json::span(out,docs.start,docs.end,work)){return false;}}
    if(!json::raw(out,b",\"Attributes\":",work) || !declarations::attributes(p,d.attribute_start,d.attribute_count,out,work)){return false;}
    if(d.kind==7){if(!json::raw(out,b",\"Type\":",work) || !types::value(p,d.result_type,0,out,work)){return false;}}
    if(d.kind==7 || d.kind==8){if(!json::raw(out,b",\"Expression\":",work) || !expressions::value(p,d.expression,0,out,work)){return false;}}
    if(d.kind==9){if(!json::raw(out,b",\"Parameters\":",work) || !declarations::parameters(p,d,out,work) || !json::raw(out,b",\"ResultType\":",work) || !types::value(p,d.result_type,0,out,work)){return false;}}
    if(d.kind==10 && (!json::raw(out,b",\"Fields\":",work) || !state_fields(p,d,out,work))){return false;}
    return json::raw(out,b"}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,usize index,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=value(p,index,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
