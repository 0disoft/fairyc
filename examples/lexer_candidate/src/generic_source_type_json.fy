module generic_source_type_json;
import generic_source_json as json;
import syntax as syntax;
import std::bytes as bytes;

fn ref u8[] kind(u32 code) {
    if(code==1 || code==6){return b"named";}if(code==2 || code==3){return b"reference";}
    if(code==11 || code==12){return b"raw-pointer";}if(code==4){return b"slice";}
    if(code==5){return b"grouped";}if(code==8){return b"array";}if(code==15){return b"foreign-function";}return b"";
}
fn bool path(ref syntax::Parsed p,usize id,ref mut bytes::Buffer out,ref mut json::Ledger work) {
    let n=syntax::type_at(p,id);if(n.start>n.end || n.end>syntax::source_view(p).len){return json::reject(work,9);}
    var usize first=0;var usize stop=syntax::token_count(p);
    while(first<stop){if(!json::charge(work,1)){return false;}let middle=first+(stop-first)/2;
        if(syntax::token_info(p,middle).start<n.start){first=middle+1;}else{stop=middle;}
    }
    var bytes::Buffer normalized=bytes::empty();
    while(first<syntax::token_count(p)){if(!json::charge(work,1)){return false;}let token=syntax::token_info(p,first);if(token.start>=n.end){break;}
        if(token.end>n.end){return json::reject(work,9);}if(token.kind!=2 && !json::raw(ref mut normalized,syntax::source_view(p)[token.start..token.end],work)){return false;}first+=1;
    }
    return json::quoted(out,bytes::view(ref normalized),work);
}
fn bool list(ref syntax::Parsed p,usize tail,u32 link_kind,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work) {
    if(tail==0){return json::raw(out,b"null",work);}var usize count=0;var usize link=tail;
    while(link!=0){if(!json::charge(work,1)){return false;}if(link>syntax::type_count(p)){return json::reject(work,9);}
        let n=syntax::type_at(p,link);if(n.kind!=link_kind || n.left==0 || n.left>=link || n.right>=link){return json::reject(work,9);}count+=1;link=n.right;
    }
    if(!json::raw(out,b"[",work)){return false;}var usize index=0;
    while(index<count){link=tail;var usize remaining=count-1;while(remaining>index){if(!json::charge(work,1)){return false;}link=syntax::type_at(p,link).right;remaining-=1;}
        if(index!=0 && !json::raw(out,b",",work)){return false;}
        if(!value(p,syntax::type_at(p,link).left,depth+1,out,work)){return false;}index+=1;
    }return json::raw(out,b"]",work);
}
pub fn bool value(ref syntax::Parsed p,usize id,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work) {
    if(depth>=64){return json::reject(work,10);}if(!json::charge(work,1)){return false;}
    if(id==0 || id>syntax::type_count(p)){return json::reject(work,9);}let n=syntax::type_at(p,id);let name=kind(n.kind);
    if(name.len==0 || n.start>n.end || n.end>syntax::source_view(p).len){return json::reject(work,9);}
    if(!json::raw(out,b"{\"Kind\":",work) || !json::quoted(out,name,work) || !json::raw(out,b",\"Text\":",work) || !json::quoted(out,syntax::type_text(p,id),work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,n.start,n.end,work) || !json::raw(out,b",\"Path\":",work)){return false;}
    if(n.kind==1){if(!path(p,id,out,work)){return false;}}
    else if(n.kind==6){if(n.left==0 || n.left>=id || syntax::type_at(p,n.left).kind!=1){return json::reject(work,9);}if(!path(p,n.left,out,work)){return false;}}
    else if(!json::quoted(out,b"",work)){return false;}
    if(!json::raw(out,b",\"Mutable\":",work) || !json::flag(out,n.kind==3 || n.kind==12,work) || !json::raw(out,b",\"Element\":",work)){return false;}
    if(n.kind==1 || n.kind==6 || n.kind==15){if(!json::raw(out,b"null",work)){return false;}}
    else{if(n.left==0 || n.left>=id){return json::reject(work,9);}if(!value(p,n.left,depth+1,out,work)){return false;}}
    if(!json::raw(out,b",\"Arguments\":",work) || !list(p,if(n.kind==6){n.right}else{0},7,depth,out,work) || !json::raw(out,b",\"Parameters\":",work) || !list(p,if(n.kind==15){n.right}else{0},16,depth,out,work) || !json::raw(out,b",\"Result\":",work)){return false;}
    if(n.kind==15){if(n.left==0 || n.left>=id){return json::reject(work,9);}if(!value(p,n.left,depth+1,out,work)){return false;}}
    else if(!json::raw(out,b"null",work)){return false;}
    if(!json::raw(out,b",\"Length\":",work)){return false;}
    if(n.kind==8){if(n.right==0 || n.right>=id){return json::reject(work,9);}let length=syntax::type_at(p,n.right);
        if(length.kind!=9 && length.kind!=10 || length.start>length.end || length.end>syntax::source_view(p).len){return json::reject(work,9);}
        if(!json::raw(out,b"{\"Text\":",work)){return false;}if(length.kind==10){if(!path(p,n.right,out,work)){return false;}}else if(!json::quoted(out,syntax::type_text(p,n.right),work)){return false;}
        if(!json::raw(out,b",\"Span\":",work) || !json::span(out,length.start,length.end,work) || !json::raw(out,b",\"Kind\":",work) || !json::quoted(out,if(length.kind==9){b"literal"}else{b"path"},work) || !json::raw(out,b"}",work)){return false;}
    }else if(!json::raw(out,b"null",work)){return false;}
    return json::raw(out,b",\"ABI\":",work) && json::quoted(out,if(n.kind==15){b"C"}else{b""},work) && json::raw(out,b"}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,usize id,ref mut usize work) {
    if(syntax::problem_count(p)!=0){return .err(9);}var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();
    let ok=value(p,id,0,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);
}
