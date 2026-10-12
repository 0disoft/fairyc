module generic_source_declaration_json;
import generic_source_json as json;
import generic_source_tokens as tokens;
import generic_source_type_json as types;
import generic_source_documentation as documentation;
import syntax as syntax;
import scanner as scanner;
import literal_types as literals;
import std::bytes as bytes;

fn bool cooked_argument(ref syntax::Parsed p,usize token,ref mut bytes::Buffer out,ref mut json::Ledger work){let encoded=scanner::token_cooked(syntax::source_stream(p),token);if(!json::charge(work,encoded.len+1)){return false;}var bytes::Buffer decoded=bytes::empty();let code=literals::decode(ref mut decoded,encoded);if(code!=0){return json::reject(work,if(code==3){11}else{9});}return json::quoted(out,bytes::view(ref decoded),work);}
pub fn bool attributes(ref syntax::Parsed p,usize first,usize count,ref mut bytes::Buffer out,ref mut json::Ledger work){if(first>syntax::attribute_count(p) || count>syntax::attribute_count(p)-first){return json::reject(work,9);}if(count==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}var usize index=0;
    while(index<count){if(!json::charge(work,1)){return false;}let a=syntax::attribute_at(p,first+index);if(a.start>a.end || a.end>syntax::source_view(p).len || a.argument_start>syntax::attribute_argument_count(p) || a.argument_count>syntax::attribute_argument_count(p)-a.argument_start){return json::reject(work,9);}
        if(index!=0 && !json::raw(out,b",",work)){return false;}if(!json::raw(out,b"{\"Name\":",work) || !tokens::path(p,a.name_start,a.name_end,out,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,a.start,a.end,work) || !json::raw(out,b",\"Arguments\":",work)){return false;}
        if(a.argument_count==0){if(!json::raw(out,b"null",work)){return false;}}else{if(!json::raw(out,b"[",work)){return false;}var usize j=0;while(j<a.argument_count){if(!json::charge(work,1)){return false;}let argument=syntax::attribute_argument_at(p,a.argument_start+j);
            if(argument.start>argument.end || argument.end>syntax::source_view(p).len || argument.token>=syntax::token_count(p)){return json::reject(work,9);}if(j!=0 && !json::raw(out,b",",work)){return false;}if(!json::raw(out,b"{\"Raw\":",work)){return false;}
            if(argument.kind==1){if(!tokens::path(p,argument.start,argument.end,out,work)){return false;}}else if(!tokens::text(p,argument.start,argument.end,out,work)){return false;}
            if(!json::raw(out,b",\"Cooked\":",work)){return false;}let raw=syntax::attribute_argument_text(p,a.argument_start+j);
            if(raw.len!=0 && raw[0]==34){if(!cooked_argument(p,argument.token,out,work)){return false;}}else if(!json::quoted(out,b"",work)){return false;}
            if(!json::raw(out,b",\"Span\":",work) || !json::span(out,argument.start,argument.end,work) || !json::raw(out,b"}",work)){return false;}j+=1;}
            if(!json::raw(out,b"]",work)){return false;}}
        if(!json::raw(out,b"}",work)){return false;}index+=1;
    }return json::raw(out,b"]",work);
}
fn bool parameters(ref syntax::Parsed p,syntax::Declaration d,ref mut bytes::Buffer out,ref mut json::Ledger work){if(d.entry_start>syntax::entry_count(p) || d.entry_count>syntax::entry_count(p)-d.entry_start){return json::reject(work,9);}var usize count=0;var usize index=0;
    while(index<d.entry_count){if(!json::charge(work,1)){return false;}if(syntax::entry_at(p,d.entry_start+index).kind==1){count+=1;}index+=1;}if(count==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}index=0;var usize written=0;
    while(index<d.entry_count){if(!json::charge(work,1)){return false;}let p_entry=syntax::entry_at(p,d.entry_start+index);if(p_entry.kind==1){if(p_entry.start>p_entry.end || p_entry.end>syntax::source_view(p).len){return json::reject(work,9);}if(written!=0 && !json::raw(out,b",",work)){return false;}
        if(!json::raw(out,b"{\"Name\":",work) || !tokens::text(p,p_entry.name_start,p_entry.name_end,out,work) || !json::raw(out,b",\"Mutable\":false,\"Span\":",work) || !json::span(out,p_entry.start,p_entry.end,work) || !json::raw(out,b",\"NameSpan\":",work) || !json::span(out,p_entry.name_start,p_entry.name_end,work) || !json::raw(out,b",\"Type\":",work) || !types::value(p,p_entry.type_root,0,out,work) || !json::raw(out,b"}",work)){return false;}written+=1;}index+=1;
    }return json::raw(out,b"]",work);
}
fn bool type_parameters(ref syntax::Parsed p,syntax::Declaration d,ref mut bytes::Buffer out,ref mut json::Ledger work){if(d.generic_parameters==0){return json::raw(out,b"null",work);}var usize link=d.generic_parameters;var usize count=0;
    while(link!=0){if(!json::charge(work,1)){return false;}if(link>syntax::type_count(p)){return json::reject(work,9);}let n=syntax::type_at(p,link);if(n.kind!=14 || n.left==0 || n.left>=link || n.right>=link || syntax::type_at(p,n.left).kind!=13){return json::reject(work,9);}count+=1;link=n.right;}
    if(!json::raw(out,b"[",work)){return false;}var usize index=0;while(index<count){link=d.generic_parameters;var usize remaining=count-1;while(remaining>index){if(!json::charge(work,1)){return false;}link=syntax::type_at(p,link).right;remaining-=1;}let n=syntax::type_at(p,syntax::type_at(p,link).left);
        if(index!=0 && !json::raw(out,b",",work)){return false;}if(!tokens::text(p,n.start,n.end,out,work)){return false;}index+=1;}return json::raw(out,b"]",work);
}
// Prefix shared by the complete function definition writer. It intentionally
// leaves the object open for the Body field; this is not a definition digest.
pub fn bool function_fields(ref syntax::Parsed p,usize index,ref mut bytes::Buffer out,ref mut json::Ledger work){if(syntax::problem_count(p)!=0 || index>=syntax::declaration_count(p)){return json::reject(work,9);}let d=syntax::declaration_at(p,index);
    if(d.kind!=3 || d.start>d.end || d.end>syntax::source_view(p).len){return json::reject(work,9);}let docs=documentation::declaration(p,index,work);if(work.code!=0){return false;}
    return json::raw(out,b"{\"TypeParameters\":",work) && type_parameters(p,d,out,work) && json::raw(out,b",\"Name\":",work) && tokens::text(p,d.name_start,d.name_end,out,work) && json::raw(out,b",\"Owner\":",work) && tokens::text(p,d.owner_start,d.owner_end,out,work) && json::raw(out,b",\"Public\":",work) && json::flag(out,d.public,work) && json::raw(out,b",\"Unsafe\":",work) && json::flag(out,d.unsafe_end>d.unsafe_start,work) && json::raw(out,b",\"Span\":",work) && json::span(out,d.start,d.end,work) && json::raw(out,b",\"NameSpan\":",work) && json::span(out,d.name_start,d.name_end,work) && json::raw(out,b",\"OwnerSpan\":",work) && json::span(out,d.owner_start,d.owner_end,work) && json::raw(out,b",\"DocumentationSpan\":",work) && json::span(out,docs.start,docs.end,work) && json::raw(out,b",\"Attributes\":",work) && attributes(p,d.attribute_start,d.attribute_count,out,work) && json::raw(out,b",\"Parameters\":",work) && parameters(p,d,out,work) && json::raw(out,b",\"ResultType\":",work) && types::value(p,d.result_type,0,out,work);
}
pub fn Result<bytes::Buffer,u32> header(ref syntax::Parsed p,usize index,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=function_fields(p,index,ref mut out,ref mut ledger) && json::raw(ref mut out,b"}",ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
