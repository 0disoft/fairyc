module generic_source_expression_json;
import generic_source_json as json;
import generic_source_tokens as tokens;
import generic_source_type_json as types;
import generic_source_expression_fields as fields;
import syntax as syntax;
import std::bytes as bytes;

// Lists are reverse parser links, never source expressions. Walk and emit in
// source order under the same work ledger; descending IDs reject cycles.
fn usize list_count(ref syntax::Parsed p,usize tail,u32 kind,ref mut json::Ledger work){var usize count=0;var usize link=tail;
    while(link!=0){if(!json::charge(work,1)){return 0;}if(link>syntax::expression_count(p)){discard json::reject(work,9);return 0;}let n=syntax::expression_at(p,link);if(n.kind!=kind || n.left==0 || n.left>=link || n.right>=link){discard json::reject(work,9);return 0;}count+=1;link=n.right;}return count;
}
fn usize list_entry(ref syntax::Parsed p,usize tail,usize count,usize index,ref mut json::Ledger work){var usize link=tail;var usize remaining=count-1;while(remaining>index){if(!json::charge(work,1)){return 0;}link=syntax::expression_at(p,link).right;remaining-=1;}return link;}
fn bool labels(ref syntax::Parsed p,syntax::Expression n,ref mut bytes::Buffer out,ref mut json::Ledger work){if(n.kind!=6){return true;}let count=list_count(p,n.right,7,work);if(work.code!=0){return false;}var usize selected=0;var usize link=n.right;
    while(link!=0){if(!json::charge(work,1)){return false;}let a=syntax::expression_at(p,link);if(a.token>=syntax::token_count(p)){return json::reject(work,9);}if(syntax::token_info(p,a.token).start!=a.start){selected+=1;}link=a.right;}if(selected==0){return true;}
    if(n.left==0 || n.left>syntax::expression_count(p) || !json::raw(out,b"\"CallLabels\":{\"Labels\":[",work)){return json::reject(work,9);}var usize index=0;var usize written=0;
    while(index<count){link=list_entry(p,n.right,count,index,work);if(work.code!=0){return false;}let a=syntax::expression_at(p,link);let token=syntax::token_info(p,a.token);
        if(token.start!=a.start){if(written!=0 && !json::raw(out,b",",work)){return false;}if(!json::raw(out,b"{\"Index\":",work) || !json::number(out,index,work) || !json::raw(out,b",\"Name\":",work) || !tokens::text(p,token.start,token.end,out,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,a.start,a.end,work) || !json::raw(out,b"}",work)){return false;}written+=1;}index+=1;
    }return json::raw(out,b"],\"ExplicitTypeArguments\":",work) && json::flag(out,syntax::expression_at(p,n.left).kind==20,work) && json::raw(out,b"},",work);
}
fn bool arguments(ref syntax::Parsed p,usize tail,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){if(tail==0){return json::raw(out,b"null",work);}var usize count=0;var usize link=tail;
    while(link!=0){if(!json::charge(work,1)){return false;}if(link>syntax::type_count(p)){return json::reject(work,9);}let n=syntax::type_at(p,link);if(n.kind!=7 || n.left==0 || n.left>=link || n.right>=link){return json::reject(work,9);}count+=1;link=n.right;}
    if(!json::raw(out,b"[",work)){return false;}var usize index=0;while(index<count){link=tail;var usize remaining=count-1;while(remaining>index){if(!json::charge(work,1)){return false;}link=syntax::type_at(p,link).right;remaining-=1;}
        if(index!=0 && !json::raw(out,b",",work)){return false;}if(!types::value(p,syntax::type_at(p,link).left,depth+1,out,work)){return false;}index+=1;}return json::raw(out,b"]",work);
}
fn bool child(ref syntax::Parsed p,usize parent,usize id,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){if(id==0 || id>=parent){return json::reject(work,9);}return value(p,id,depth+1,out,work);}
fn bool children_list(ref syntax::Parsed p,usize parent,usize tail,u32 kind,usize depth,bool keep_entries,ref mut bytes::Buffer out,ref mut json::Ledger work){let count=list_count(p,tail,kind,work);if(work.code!=0){return false;}var usize index=0;while(index<count){let link=list_entry(p,tail,count,index,work);if(work.code!=0){return false;}if(index!=0 && !json::raw(out,b",",work)){return false;}if(!child(p,parent,if(keep_entries){link}else{syntax::expression_at(p,link).left},depth,out,work)){return false;}index+=1;}return true;}
fn bool children(ref syntax::Parsed p,usize id,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){let n=syntax::expression_at(p,id);
    if(n.kind==1 || n.kind==2 || n.kind==28 || n.kind==24 && n.right==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}
    if(n.kind==24){if(!children_list(p,id,n.right,25,depth,false,out,work)){return false;}}
    else{var usize first=n.left;if(n.kind==10 && first!=0 && first<id && syntax::expression_at(p,first).kind==20){first=syntax::expression_at(p,first).left;}if(!child(p,id,first,depth,out,work)){return false;}
        if(n.kind==5 || n.kind==9 || n.kind==29){if(!json::raw(out,b",",work) || !child(p,id,n.right,depth,out,work)){return false;}}
        else if(n.kind==6 || n.kind==10){if(n.right!=0){if(!json::raw(out,b",",work) || !children_list(p,id,n.right,if(n.kind==6){7}else{11},depth,n.kind==10,out,work)){return false;}}}
        else if(n.kind==26){if(n.right==0 || n.right>=id){return json::reject(work,9);}let b=syntax::expression_at(p,n.right);if(b.kind!=27 || b.token>=syntax::token_count(p)){return json::reject(work,9);}let range=syntax::token_info(p,b.token);
            if(!json::raw(out,b",",work)){return false;}if(b.left!=0){if(!child(p,id,b.left,depth,out,work)){return false;}}else if(!fields::omitted(range.start,range.end,true,out,work)){return false;}
            if(!json::raw(out,b",",work)){return false;}if(b.right!=0){if(!child(p,id,b.right,depth,out,work)){return false;}}else{let close=tokens::first(p,n.end-1,work);if(work.code!=0){return false;}if(close>=syntax::token_count(p)){return json::reject(work,9);}let t=syntax::token_info(p,close);if(t.end!=n.end || !bytes::equal(syntax::source_view(p)[t.start..t.end],b"]")){return json::reject(work,9);}if(!fields::omitted(t.start,t.start,false,out,work)){return false;}}
        }
    }return json::raw(out,b"]",work);
}
pub fn bool value(ref syntax::Parsed p,usize id,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){if(depth>=64){return json::reject(work,10);}if(!json::charge(work,1)){return false;}if(id==0 || id>syntax::expression_count(p)){return json::reject(work,9);}let n=syntax::expression_at(p,id);if(n.start>n.end || n.end>syntax::source_view(p).len){return json::reject(work,9);}
    if(fields::kind(n.kind).len==0){return json::reject(work,if(n.kind==12 || n.kind==13 || n.kind==15 || n.kind==31){4503}else{9});}
    var usize type_arguments=if(n.kind==20){n.right}else{0};if(n.kind==10 && n.left!=0 && n.left<id && syntax::expression_at(p,n.left).kind==20){type_arguments=syntax::expression_at(p,n.left).right;}
    if(!json::raw(out,b"{",work) || !labels(p,n,out,work) || !json::raw(out,b"\"TypeArguments\":",work) || !arguments(p,type_arguments,depth,out,work) || !json::raw(out,b",",work) || !fields::prefix(p,id,out,work) || !json::raw(out,b",\"Type\":",work)){return false;}
    if(n.kind==23){if(!types::value(p,n.right,depth+1,out,work)){return false;}}else if(!json::raw(out,b"null",work)){return false;}
    return json::raw(out,b",\"Children\":",work) && children(p,id,depth,out,work) && json::raw(out,b",\"Block\":null}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,usize id,ref mut usize work){if(syntax::problem_count(p)!=0){return .err(9);}var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=value(p,id,0,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
