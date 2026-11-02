module generic_source_function_json;
import generic_source_declaration_json as declarations;
import generic_source_expression_json as expressions;
import generic_source_json as json;
import generic_identity as identity;
import syntax as syntax;
import static_operation_declarations as operations;
import std::bytes as bytes;

pub fn bool value(ref syntax::Parsed p,usize index,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let original=syntax::written_function(p,index+1);if(original.len!=0){return json::raw(out,original,work);}
    if(!declarations::function_fields(p,index,out,work)){return false;}let d=syntax::declaration_at(p,index);
    if(operations::operation(p,index+1).kind==1){let b=syntax::statement_at(p,d.body);return json::raw(out,b",\"Body\":{\"Span\":",work) && json::span(out,b.start,b.end,work) && json::raw(out,b",\"Statements\":null,\"Tail\":null}}",work);}
    return json::raw(out,b",\"Body\":",work) && expressions::block(p,d.body,0,out,work) && json::raw(out,b"}",work);
}
pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,usize index,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();let ok=value(p,index,ref mut out,ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);}
// The standalone template definition matches genericplan.DigestText over the
// original Go-shaped function AST. Package import bindings are a later domain.
pub fn Result<bytes::Buffer,u32> definition(ref syntax::Parsed p,usize index,ref mut usize work){let encoded=switch(encode(p,index,work)){case .ok(value):{value}case .err(code):{return .err(code);}};return identity::digest_text(b"source-function-ast-v0",bytes::view(ref encoded),work);}
