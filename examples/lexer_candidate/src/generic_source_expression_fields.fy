module generic_source_expression_fields;
import generic_source_json as json;
import generic_source_tokens as tokens;
import syntax as syntax;
import std::bytes as bytes;

pub fn ref u8[] kind(u32 code){
    if(code==1){return b"literal";}if(code==2 || code==28){return b"path";}if(code==3){return b"group";}
    if(code==4 || code==21 || code==22){return b"prefix";}if(code==5){return b"binary";}if(code==6){return b"call";}
    if(code==8){return b"field-expression";}if(code==9){return b"index";}if(code==10){return b"record";}if(code==11){return b"field";}
    if(code==20){return b"specialize";}if(code==23){return b"cast";}if(code==24 || code==29){return b"array";}
    if(code==26){return b"slice";}if(code==30){return b"propagate";}return b"";
}
pub fn bool text(ref syntax::Parsed p,usize id,ref mut bytes::Buffer out,ref mut json::Ledger work){let n=syntax::expression_at(p,id);
    if(n.kind==2 || n.kind==28){return tokens::path(p,n.start,n.end,out,work);}
    if(n.kind==10){if(n.left==0 || n.left>=id){return json::reject(work,9);}var usize root=n.left;if(syntax::expression_at(p,root).kind==20){let base=syntax::expression_at(p,root);if(base.left==0 || base.left>=root){return json::reject(work,9);}root=base.left;}let base=syntax::expression_at(p,root);if(base.kind!=2 && base.kind!=28){return json::reject(work,9);}return tokens::path(p,base.start,base.end,out,work);}
    if(n.kind==21 || n.kind==22){return json::quoted(out,if(n.kind==21){b"&"}else{b"&mut"},work);}
    if(n.kind==24){return json::quoted(out,if(n.right==0){b"empty"}else{b"list"},work);}if(n.kind==29){return json::quoted(out,b"repeat",work);}
    if(n.kind==26){return json::quoted(out,b"..",work);}if(n.kind==1 || n.kind==4 || n.kind==5 || n.kind==8 || n.kind==11 || n.kind==23 || n.kind==30){
        if(n.token>=syntax::token_count(p)){return json::reject(work,9);}let token=syntax::token_info(p,n.token);return tokens::text(p,token.start,token.end,out,work);
    }return json::quoted(out,b"",work);
}
pub fn bool prefix(ref syntax::Parsed p,usize id,ref mut bytes::Buffer out,ref mut json::Ledger work){let n=syntax::expression_at(p,id);let name=kind(n.kind);
    if(name.len==0){return json::reject(work,if(n.kind==12 || n.kind==13 || n.kind==15 || n.kind==31){4503}else{9});}
    if(!json::raw(out,b"\"Kind\":",work) || !json::quoted(out,name,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,n.start,n.end,work) || !json::raw(out,b",\"Text\":",work) || !text(p,id,out,work) || !json::raw(out,b",\"Cooked\":",work)){return false;}
    if(n.kind==1){if(n.token>=syntax::token_count(p)){return json::reject(work,9);}return json::quoted(out,syntax::literal_cooked(p,id),work);}return json::quoted(out,b"",work);
}
pub fn bool omitted(usize start,usize end,bool first,ref mut bytes::Buffer out,ref mut json::Ledger work){
    return json::raw(out,b"{\"TypeArguments\":null,\"Kind\":\"omitted-range-bound\",\"Span\":",work) && json::span(out,start,end,work) && json::raw(out,b",\"Text\":",work) && json::quoted(out,if(first){b"start"}else{b"end"},work) && json::raw(out,b",\"Cooked\":\"\",\"Type\":null,\"Children\":null,\"Block\":null}",work);
}
