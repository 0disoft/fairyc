module generic_source_module_json;
import generic_source_function_json as functions;
import generic_source_aggregate_json as aggregates;
import generic_source_item_json as items;
import generic_source_observations as observations;
import generic_source_tokens as tokens;
import generic_source_json as json;
import generic_identity as identity;
import syntax as syntax;
import std::bytes as bytes;

fn bool list(ref syntax::Parsed p,u32 kind,ref mut bytes::Buffer out,ref mut json::Ledger work){var usize count=0;var usize index=1;
    while(index<syntax::declaration_count(p)){if(!json::charge(work,1)){return false;}if(syntax::declaration_at(p,index).kind==kind){count+=1;}index+=1;}
    if(count==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}index=1;var usize written=0;
    while(index<syntax::declaration_count(p)){if(!json::charge(work,1)){return false;}let d=syntax::declaration_at(p,index);if(d.kind==kind){if(written!=0 && !json::raw(out,b",",work)){return false;}
        let ok=if(kind==3){functions::value(p,index,out,work)}else if(kind==4 || kind==6 || kind==11){aggregates::value(p,index,out,work)}else{items::value(p,index,out,work)};if(!ok){return false;}written+=1;}index+=1;
    }return json::raw(out,b"]",work);
}
pub fn bool value(ref syntax::Parsed p,ref mut bytes::Buffer out,ref mut json::Ledger work){if(!json::charge(work,1)){return false;}if(syntax::problem_count(p)!=0 || syntax::declaration_count(p)==0){return json::reject(work,9);}let module_item=syntax::declaration_at(p,0);let source=syntax::source_view(p);if(module_item.kind!=1 || module_item.start>module_item.end || module_item.end>source.len || module_item.name_start>=module_item.name_end || module_item.name_end>module_item.end){return json::reject(work,9);}var usize end=module_item.end;var usize index=1;
    while(index<syntax::declaration_count(p)){if(!json::charge(work,1)){return false;}let d=syntax::declaration_at(p,index);if((d.kind!=2 && d.kind!=3 && d.kind!=4 && d.kind!=6 && d.kind!=7 && d.kind!=8 && d.kind!=9 && d.kind!=10 && d.kind!=11) || d.start<end || d.start>d.end || d.end>source.len){return json::reject(work,9);}end=d.end;index+=1;}
    if(!json::raw(out,b"{\"Path\":",work) || !tokens::path(p,module_item.name_start,module_item.name_end,out,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,module_item.start,end,work) || !json::raw(out,b",\"SourceSize\":",work) || !json::number(out,source.len,work) || !json::raw(out,b",\"SourceLineStarts\":[0",work)){return false;}
    index=0;while(index<source.len){if(!json::charge(work,1)){return false;}if(source[index]==10 && (!json::raw(out,b",",work) || !json::number(out,index+1,work))){return false;}index+=1;}
    return json::raw(out,b"],\"Imports\":",work) && list(p,2,out,work) && json::raw(out,b",\"Constants\":",work) && list(p,7,out,work) && json::raw(out,b",\"StaticAssertions\":",work) && list(p,8,out,work) && json::raw(out,b",\"States\":",work) && list(p,10,out,work) && json::raw(out,b",\"Structs\":",work) && list(p,4,out,work) && json::raw(out,b",\"Resources\":",work) && list(p,6,out,work) && json::raw(out,b",\"Enums\":",work) && list(p,11,out,work) && json::raw(out,b",\"ForeignFunctions\":",work) && list(p,9,out,work) && json::raw(out,b",\"Functions\":",work) && list(p,3,out,work) && json::raw(out,b",\"Bindings\":",work) && observations::bindings(p,out,work) && json::raw(out,b",\"Literals\":",work) && observations::literals(p,out,work) && json::raw(out,b"}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=value(p,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
// Conservative standalone nominal identity includes every original module fact,
// helper body and exact source geometry. Package graph context is a later domain.
pub fn Result<bytes::Buffer,u32> nominal_definition(ref syntax::Parsed p,usize index,ref mut usize work){if(index>=syntax::declaration_count(p)){return .err(9);}let d=syntax::declaration_at(p,index);if((d.kind!=4 && d.kind!=6) || d.generic_parameters!=0 || d.name_start>=d.name_end || d.name_end>syntax::source_view(p).len){return .err(9);}var encoded=switch(encode(p,work)){case .ok(value):{value}case .err(code):{return .err(code);}};var json::Ledger ledger=json::Ledger{.used=*work,.code=0};let ok=json::raw(ref mut encoded,b"\n",ref mut ledger) && json::raw(ref mut encoded,syntax::source_view(p)[d.name_start..d.name_end],ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return identity::digest_text(b"source-nominal-module-v0",bytes::view(ref encoded),work);}
