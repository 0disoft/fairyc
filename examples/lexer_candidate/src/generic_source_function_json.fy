module generic_source_function_json;
import generic_source_declaration_json as declarations;
import generic_source_expression_json as expressions;
import generic_source_json as json;
import generic_identity as identity;
import syntax as syntax;
import std::bytes as bytes;

pub fn Result<bytes::Buffer,u32> encode(ref syntax::Parsed p,usize index,ref mut usize work){var json::Ledger ledger=json::Ledger{.used=*work,.code=0};var bytes::Buffer out=bytes::empty();
    if(!declarations::function_fields(p,index,ref mut out,ref mut ledger)){*work=ledger.used;return .err(ledger.code);}let d=syntax::declaration_at(p,index);
    let ok=json::raw(ref mut out,b",\"Body\":",ref mut ledger) && expressions::block(p,d.body,0,ref mut out,ref mut ledger) && json::raw(ref mut out,b"}",ref mut ledger);*work=ledger.used;if(!ok){return .err(ledger.code);}return .ok(out);
}
// The standalone template definition matches genericplan.DigestText over the
// original Go-shaped function AST. Package import bindings are a later domain.
pub fn Result<bytes::Buffer,u32> definition(ref syntax::Parsed p,usize index,ref mut usize work){let encoded=switch(encode(p,index,work)){case .ok(value):{value}case .err(code):{return .err(code);}};return identity::digest_text(b"source-function-ast-v0",bytes::view(ref encoded),work);}
