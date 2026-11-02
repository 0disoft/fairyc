module generic_source_copied_closure_json;
import generic_source_json as json;
import generic_source_type_json as types;
import generic_source_expression_json as expressions;
import syntax as syntax;
import std::bytes as bytes;

fn bool parameters(ref syntax::Parsed p,usize first,usize count,ref mut bytes::Buffer out,ref mut json::Ledger work){if(count==0){return json::raw(out,b"null",work);}if(!json::raw(out,b"[",work)){return false;}var usize i=0;while(i<count){let e=syntax::entry_at(p,first+i-1);if(i!=0 && !json::raw(out,b",",work)){return false;}if(!json::raw(out,b"{\"Name\":",work) || !json::quoted(out,syntax::source_view(p)[e.name_start..e.name_end],work) || !json::raw(out,b",\"Mutable\":false,\"Span\":",work) || !json::span(out,e.start,e.end,work) || !json::raw(out,b",\"NameSpan\":",work) || !json::span(out,e.name_start,e.name_end,work) || !json::raw(out,b",\"Type\":",work) || !types::value(p,e.type_root,0,out,work) || !json::raw(out,b"}",work)){return false;}i+=1;}return json::raw(out,b"]",work);}
pub fn bool value(ref syntax::Parsed p,syntax::Expression n,usize depth,ref mut bytes::Buffer out,ref mut json::Ledger work){
    if(depth>=64 || n.kind!=34 || n.right==0 || n.right>syntax::expression_count(p)){return json::reject(work,9);}let h=syntax::expression_at(p,n.right);if(h.kind!=35 || h.left==0){return json::reject(work,9);}let captures=h.right/129;let count=h.right%129;
    if(h.left-1>syntax::entry_count(p) || captures+count>syntax::entry_count(p)-(h.left-1)){return json::reject(work,9);}
    return json::raw(out,b"\"Closure\":{\"Captures\":",work) && parameters(p,h.left,captures,out,work) && json::raw(out,b",\"Function\":{\"TypeParameters\":null,\"Name\":\"\",\"Owner\":\"\",\"Public\":false,\"Unsafe\":false,\"Span\":",work) && json::span(out,n.start,n.end,work) && json::raw(out,b",\"NameSpan\":{\"StartByte\":0,\"EndByte\":0},\"OwnerSpan\":{\"StartByte\":0,\"EndByte\":0},\"DocumentationSpan\":{\"StartByte\":0,\"EndByte\":0},\"Attributes\":null,\"Parameters\":",work) && parameters(p,h.left+captures,count,out,work) && json::raw(out,b",\"ResultType\":",work) && types::value(p,h.token,depth+1,out,work) && json::raw(out,b",\"Body\":",work) && expressions::block(p,n.left,depth+1,out,work) && json::raw(out,b"}},",work);
}
