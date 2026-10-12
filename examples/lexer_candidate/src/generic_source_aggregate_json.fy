module generic_source_aggregate_json;
import generic_source_declaration_json as declarations;
import generic_source_documentation as documentation;
import generic_source_expression_json as expressions;
import generic_source_type_json as types;
import generic_source_tokens as tokens;
import generic_source_json as json;
import generic_identity as identity;
import syntax as syntax;
import std::bytes as bytes;

fn bool field(ref syntax::Parsed p,usize index,u32 kind,ref mut bytes::Buffer out,ref mut json::Ledger work){if(index>=syntax::field_count(p)){return json::reject(work,9);}let f=syntax::field_at(p,index);
    if(f.kind!=kind || f.start>f.end || f.end>syntax::source_view(p).len || f.name_start<f.start || f.name_start>f.name_end || f.name_end>f.end){return json::reject(work,9);}
    return json::raw(out,b"{\"Name\":",work) && tokens::text(p,f.name_start,f.name_end,out,work) && json::raw(out,b",\"Span\":",work) && json::span(out,f.start,f.end,work) && json::raw(out,b",\"NameSpan\":",work) && json::span(out,f.name_start,f.name_end,work) && json::raw(out,b",\"Type\":",work) && types::value(p,f.type_root,0,out,work) && json::raw(out,b"}",work);
}
fn bool fields(ref syntax::Parsed p,usize first,usize count,u32 kind,ref mut bytes::Buffer out,ref mut json::Ledger work){if(first>syntax::field_count(p) || count>syntax::field_count(p)-first){return json::reject(work,9);}if(count==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}var usize index=0;
    while(index<count){if(!json::charge(work,1)){return false;}if(index!=0 && !json::raw(out,b",",work)){return false;}if(!field(p,first+index,kind,out,work)){return false;}index+=1;}return json::raw(out,b"]",work);
}
fn bool variants(ref syntax::Parsed p,syntax::Declaration d,ref mut bytes::Buffer out,ref mut json::Ledger work){if(d.field_start>syntax::field_count(p) || d.field_count>syntax::field_count(p)-d.field_start){return json::reject(work,9);}var usize count=0;var usize index=0;
    while(index<d.field_count){if(!json::charge(work,1)){return false;}let f=syntax::field_at(p,d.field_start+index);if(f.kind==3){count+=1;}else if(f.kind!=4){return json::reject(work,9);}index+=1;}
    if(count==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}index=0;var usize written=0;
    while(index<d.field_count){if(!json::charge(work,1)){return false;}let slot=d.field_start+index;let f=syntax::field_at(p,slot);if(f.kind==3){
        if(f.start>f.end || f.end>d.end || f.name_start<f.start || f.name_start>f.name_end || f.name_end>f.end || f.payload_start>slot || f.payload_count>slot-f.payload_start || f.payload_count!=0 && f.payload_start<d.field_start){return json::reject(work,9);}
        if(written!=0 && !json::raw(out,b",",work)){return false;}if(!json::raw(out,b"{\"Name\":",work) || !tokens::text(p,f.name_start,f.name_end,out,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,f.start,f.end,work) || !json::raw(out,b",\"NameSpan\":",work) || !json::span(out,f.name_start,f.name_end,work) || !json::raw(out,b",\"Fields\":",work) || !fields(p,f.payload_start,f.payload_count,4,out,work) || !json::raw(out,b"}",work)){return false;}written+=1;
    }index+=1;}return json::raw(out,b"]",work);
}
fn bool cleanup_value(ref syntax::Parsed p,syntax::Declaration d,ref mut bytes::Buffer out,ref mut json::Ledger work){if(d.cleanup_body==0){if(d.cleanup_start!=0 || d.cleanup_end!=0 || d.cleanup_attribute_count!=0){return json::reject(work,9);}return json::raw(out,b"null",work);}
    if(d.cleanup_start>d.cleanup_end || d.cleanup_end>d.end){return json::reject(work,9);}return json::raw(out,b"{\"Span\":",work) && json::span(out,d.cleanup_start,d.cleanup_end,work) && json::raw(out,b",\"ReceiverName\":\"this\",\"Attributes\":",work) && declarations::attributes(p,d.cleanup_attribute_start,d.cleanup_attribute_count,out,work) && json::raw(out,b",\"Body\":",work) && expressions::block(p,d.cleanup_body,0,out,work) && json::raw(out,b"}",work);
}
pub fn bool value(ref syntax::Parsed p,usize index,ref mut bytes::Buffer out,ref mut json::Ledger work){if(!json::charge(work,1)){return false;}if(syntax::problem_count(p)!=0 || index>=syntax::declaration_count(p)){return json::reject(work,9);}let d=syntax::declaration_at(p,index);
    if((d.kind!=4 && d.kind!=6 && d.kind!=11) || d.start>d.end || d.end>syntax::source_view(p).len){return json::reject(work,9);}let docs=documentation::declaration(p,index,work);if(work.code!=0){return false;}
    if(!json::raw(out,b"{",work)){return false;}if(d.kind!=11 && (!json::raw(out,b"\"TypeParameters\":",work) || !declarations::type_parameters(p,d,out,work) || !json::raw(out,b",",work))){return false;}
    if(!json::raw(out,b"\"Name\":",work) || !tokens::text(p,d.name_start,d.name_end,out,work) || !json::raw(out,b",\"Public\":",work) || !json::flag(out,d.public,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,d.start,d.end,work) || !json::raw(out,b",\"NameSpan\":",work) || !json::span(out,d.name_start,d.name_end,work) || !json::raw(out,b",\"DocumentationSpan\":",work) || !json::span(out,docs.start,docs.end,work) || !json::raw(out,b",\"Attributes\":",work) || !declarations::attributes(p,d.attribute_start,d.attribute_count,out,work)){return false;}
    if(d.kind==11){if(!json::raw(out,b",\"Variants\":",work) || !variants(p,d,out,work)){return false;}}
    else{if(!json::raw(out,b",\"Fields\":",work) || !fields(p,d.field_start,d.field_count,0,out,work)){return false;}if(d.kind==6 && (!json::raw(out,b",\"Cleanup\":",work) || !cleanup_value(p,d,out,work))){return false;}}
    return json::raw(out,b"}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,usize index,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=value(p,index,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
pub fn Result<bytes::Buffer,u32> template(ref syntax::Parsed p,usize index,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};if(index>=syntax::declaration_count(p)){return .err(9);}let d=syntax::declaration_at(p,index);if(d.kind!=4 && d.kind!=6){return .err(9);}var bytes::Buffer out=bytes::empty();
    let first=json::raw(ref mut out,b"{\"Struct\":",ref mut ledger) && if(d.kind==4){value(p,index,ref mut out,ref mut ledger)}else{json::raw(ref mut out,b"null",ref mut ledger)};
    let second=first && json::raw(ref mut out,b",\"Resource\":",ref mut ledger) && if(d.kind==6){value(p,index,ref mut out,ref mut ledger)}else{json::raw(ref mut out,b"null",ref mut ledger)};
    let ok=second && json::raw(ref mut out,b"}",ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);
}
pub fn Result<bytes::Buffer,u32> definition(ref syntax::Parsed p,usize index,ref mut usize work){let encoded=switch(template(p,index,work)){case .ok(value):{value}case .err(code):{return .err(code);}};return identity::digest_text(b"source-aggregate-ast-v0",bytes::view(ref encoded),work);}
