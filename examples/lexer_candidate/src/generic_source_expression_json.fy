module generic_source_expression_json;
import generic_source_json as json;
import generic_source_tokens as tokens;
import generic_source_type_json as types;
import generic_source_expression_fields as fields;
import generic_source_statement_fields as statements;
import generic_source_view_loop_json as view_loops;
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
    if(n.kind==1 || n.kind==2 || n.kind==17 || n.kind==28 || n.kind==33 || n.kind==12 || n.kind==24 && n.right==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}
    if(n.kind==24){if(!children_list(p,id,n.right,25,depth,false,out,work)){return false;}}
    else if(n.kind==31){if(!block_expression(p,n.left,depth+1,out,work)){return false;}}
    else if(n.kind==13){if(!child(p,id,n.left,depth,out,work) || n.right==0 || n.right>=id){return json::reject(work,9);}let branches=syntax::expression_at(p,n.right);if(branches.kind!=14){return json::reject(work,9);}
        if(!json::raw(out,b",",work) || !block_expression(p,branches.left,depth+1,out,work)){return false;}if(branches.right!=0 && (!json::raw(out,b",",work) || !child(p,id,branches.right,depth,out,work))){return false;}}
    else if(n.kind==15){if(!child(p,id,n.left,depth,out,work)){return false;}if(n.right!=0 && (!json::raw(out,b",",work) || !children_list(p,id,n.right,19,depth,false,out,work))){return false;}}
    else if(n.kind==18){if(!child(p,id,n.left,depth,out,work) || !json::raw(out,b",",work) || !block_expression(p,n.right,depth+1,out,work)){return false;}}
    else if(n.kind==16 || n.kind==32){if(n.token>=syntax::token_count(p)){return json::reject(work,9);}let token=syntax::token_info(p,n.token);var bytes::Buffer name=bytes::empty();if(!json::raw(ref mut name,b".",work) || !json::raw(ref mut name,syntax::source_view(p)[token.start..token.end],work) || !statements::expression(b"path",n.start,token.end,bytes::view(ref name),out,work) || !json::raw(out,b"null,\"Block\":null}",work)){return false;}
        if(n.kind==16){if(n.left!=0 && (!json::raw(out,b",",work) || !child(p,id,n.left,depth,out,work))){return false;}}
        else if(n.right!=0 && (!json::raw(out,b",",work) || !children_list(p,id,n.right,33,depth,true,out,work))){return false;}}
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
    if(!json::raw(out,b",\"Children\":",work) || !children(p,id,depth,out,work) || !json::raw(out,b",\"Block\":",work)){return false;}
    if(n.kind==12){if(!block(p,n.left,depth+1,out,work)){return false;}}else if(!json::raw(out,b"null",work)){return false;}return json::raw(out,b"}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,usize id,ref mut usize work){if(syntax::problem_count(p)!=0){return .err(9);}var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=value(p,id,0,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}

fn bool optional_expression(ref syntax::Parsed p,usize id,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){return if(id==0){json::raw(out,b"null",work)}else{value(p,id,depth,out,work)};}
fn bool block_expression(ref syntax::Parsed p,usize id,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){if(!statements::valid(p,id,work)){return false;}let b=syntax::statement_at(p,id);return statements::expression(b"block",b.start,b.end,b"",out,work) && json::raw(out,b"null,\"Block\":",work) && block(p,id,depth,out,work) && json::raw(out,b"}",work);}
pub fn bool binding(ref syntax::Parsed p,syntax::Statement s,usize end,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){if(s.entry==0 || s.entry>syntax::entry_count(p) || !json::charge(work,syntax::entry_count(p))){return json::reject(work,9);}let e=syntax::entry_at(p,s.entry-1);if((e.kind!=2 && e.kind!=3) || e.name_start>e.name_end || e.name_end>end){return json::reject(work,9);}
    if(!json::raw(out,b"{\"Name\":",work) || !tokens::text(p,e.name_start,e.name_end,out,work) || !json::raw(out,b",\"Mutable\":",work) || !json::flag(out,e.kind==3,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,s.start,end,work) || !json::raw(out,b",\"NameSpan\":",work) || !json::span(out,e.name_start,e.name_end,work) || !json::raw(out,b",\"Type\":",work)){return false;}
    if(e.type_root!=0){if(!types::value(p,e.type_root,depth+1,out,work)){return false;}}else if(!json::raw(out,b"null",work)){return false;}
    return json::raw(out,b",\"Expression\":",work) && optional_expression(p,e.expression,depth+1,out,work) && json::raw(out,b"}",work);
}
fn bool statement_expression(ref syntax::Parsed p,syntax::Statement s,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){
    if(s.kind==18){let admitted=view_loops::admitted(p,s,work);if(work.code!=0){return false;}if(!admitted){return json::raw(out,b"null",work);}var bytes::Buffer view=bytes::empty();var bytes::Buffer type=bytes::empty();var bytes::Buffer item=bytes::empty();var bytes::Buffer body=bytes::empty();
        return value(p,s.expression,depth+1,ref mut view,work) && types::value(p,s.second,depth+1,ref mut type,work) && view_loops::item(p,s,bytes::view(ref view),bytes::view(ref type),ref mut item,work) && block_with_prefix(p,s.first,depth+1,bytes::view(ref item),ref mut body,work) && view_loops::synthetic(p,s,bytes::view(ref view),bytes::view(ref body),out,work);}
    if(s.kind==8 || s.kind==9 || s.kind==10 || s.kind==17){var bytes::Buffer text=bytes::empty();if(s.kind==17){if(s.target==0 || s.target>syntax::expression_count(p)){return json::reject(work,9);}let name=syntax::expression_at(p,s.target);if(name.kind!=17 || name.token>=syntax::token_count(p)){return json::reject(work,9);}let token=syntax::token_info(p,name.token);if(!json::raw(ref mut text,syntax::source_view(p)[token.start..token.end],work)){return false;}}
        if(!statements::expression(statements::kind(s.kind),s.start,s.end,bytes::view(ref text),out,work)){return false;}
        if(s.kind==9 || s.kind==10){if(!json::raw(out,b"null",work)){return false;}}
        else if(!json::raw(out,b"[",work) || !optional_expression(p,s.expression,depth+1,out,work) || !json::raw(out,b",",work) || !block_expression(p,s.first,depth+1,out,work) || !json::raw(out,b"]",work)){return false;}
        return json::raw(out,b",\"Block\":null}",work);
    }return optional_expression(p,if(s.kind==2 || s.kind==14){0}else{s.expression},depth+1,out,work);
}
fn bool classic_for(ref syntax::Parsed p,syntax::Statement s,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){if(!statements::valid(p,s.second,work)){return false;}let header=syntax::statement_at(p,s.second);if(header.kind!=15){return json::reject(work,9);}
    if(!json::raw(out,b"{\"Span\":",work) || !json::span(out,s.start,s.end,work) || !json::raw(out,b",\"Initializer\":",work)){return false;}
    if(header.first!=0){if(!statement(p,header.first,depth+1,true,out,work)){return false;}}else if(!json::raw(out,b"null",work)){return false;}
    if(!json::raw(out,b",\"Condition\":",work) || !optional_expression(p,s.expression,depth+1,out,work) || !json::raw(out,b",\"Post\":",work)){return false;}
    if(header.second!=0){if(!statement(p,header.second,depth+1,false,out,work)){return false;}}else if(!json::raw(out,b"null",work)){return false;}
    return json::raw(out,b",\"Body\":",work) && block(p,s.first,depth+1,out,work) && json::raw(out,b"}",work);
}
pub fn bool statement(ref syntax::Parsed p,usize id,usize depth,bool initializer,ref mut bytes::Buffer out,ref mut json::Ledger work){if(depth>=64){return json::reject(work,10);}if(!json::charge(work,1) || !statements::valid(p,id,work)){return false;}let s=syntax::statement_at(p,id);let kind=statements::kind(s.kind);if(kind.len==0){return json::reject(work,if(s.kind==18){4503}else{9});}var usize end=s.end;
    if(initializer){if(s.expression==0 || s.expression>syntax::expression_count(p)){return json::reject(work,9);}end=syntax::expression_at(p,s.expression).end;}
    if(!json::raw(out,b"{\"Kind\":",work) || !json::quoted(out,kind,work) || !json::raw(out,b",\"Span\":",work) || !json::span(out,s.start,end,work) || !json::raw(out,b",\"Operator\":",work)){return false;}
    if(s.kind==4){if(s.operator>=syntax::token_count(p)){return json::reject(work,9);}let token=syntax::token_info(p,s.operator);if(!tokens::text(p,token.start,token.end,out,work)){return false;}}else if(!json::quoted(out,b"",work)){return false;}
    if(!json::raw(out,b",\"Binding\":",work)){return false;}if(s.kind==2){if(!binding(p,s,end,depth,out,work)){return false;}}else if(!json::raw(out,b"null",work)){return false;}
    if(!json::raw(out,b",\"Target\":",work) || !optional_expression(p,if(s.kind==4){s.target}else{0},depth+1,out,work) || !json::raw(out,b",\"Expression\":",work) || !statement_expression(p,s,depth,out,work) || !json::raw(out,b",\"For\":",work)){return false;}
    if(s.kind==14){if(!classic_for(p,s,depth,out,work)){return false;}}else if(!json::raw(out,b"null",work)){return false;}if(!json::raw(out,b",\"ByteFor\":",work)){return false;}
    if(s.kind==18){if(!view_loops::valid(p,s,work)){return false;}let c=view_loops::context(p,s);
        if(!json::raw(out,b"{\"Span\":",work) || !json::span(out,s.start,s.end,work) || !json::raw(out,b",\"Name\":",work) || !tokens::text(p,c.name_start,c.name_end,out,work) || !json::raw(out,b",\"NameSpan\":",work) || !json::span(out,c.name_start,c.name_end,work) || !json::raw(out,b",\"Type\":",work) || !types::value(p,s.second,depth+1,out,work) || !json::raw(out,b",\"View\":",work) || !value(p,s.expression,depth+1,out,work) || !json::raw(out,b",\"Body\":",work) || !block(p,s.first,depth+1,out,work) || !json::raw(out,b"}",work)){return false;}
    }else if(!json::raw(out,b"null",work)){return false;}return json::raw(out,b"}",work);
}
fn bool block_with_prefix(ref syntax::Parsed p,usize id,usize depth,ref u8[] prefix,ref mut bytes::Buffer out,ref mut json::Ledger work){if(depth>=64){return json::reject(work,10);}if(!json::charge(work,1) || !statements::valid(p,id,work)){return false;}let b=syntax::statement_at(p,id);if(b.kind!=1){return json::reject(work,9);}var usize link=b.first;var usize count=0;
    while(link!=0){if(!json::charge(work,1) || !statements::valid(p,link,work)){return false;}let n=syntax::statement_at(p,link);if(n.kind!=7 || n.first==0 || n.first>=link || n.second>=link){return json::reject(work,9);}count+=1;link=n.second;}if(count!=b.second){return json::reject(work,9);}
    if(!json::raw(out,b"{\"Span\":",work) || !json::span(out,b.start,b.end,work) || !json::raw(out,b",\"Statements\":",work)){return false;}
    if(count==0 && prefix.len==0){if(!json::raw(out,b"null",work)){return false;}}else{if(!json::raw(out,b"[",work)){return false;}if(prefix.len!=0 && !json::raw(out,prefix,work)){return false;}var usize index=0;while(index<count){link=b.first;var usize remaining=count-1;while(remaining>index){if(!json::charge(work,1)){return false;}link=syntax::statement_at(p,link).second;remaining-=1;}
        if((index!=0 || prefix.len!=0) && !json::raw(out,b",",work)){return false;}if(!statement(p,syntax::statement_at(p,link).first,depth+1,false,out,work)){return false;}index+=1;}if(!json::raw(out,b"]",work)){return false;}}
    return json::raw(out,b",\"Tail\":",work) && optional_expression(p,b.expression,depth+1,out,work) && json::raw(out,b"}",work);
}
pub fn bool block(ref syntax::Parsed p,usize id,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){return block_with_prefix(p,id,depth,b"",out,work);}
