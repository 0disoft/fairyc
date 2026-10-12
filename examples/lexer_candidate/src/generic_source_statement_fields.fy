module generic_source_statement_fields;
import generic_source_json as json;
import syntax as syntax;
import std::bytes as bytes;

pub fn ref u8[] kind(u32 code){
    if(code==2){return b"binding";}if(code==3){return b"return";}if(code==4){return b"assignment";}if(code==6){return b"expression-statement";}
    if(code==8){return b"while";}if(code==9){return b"break";}if(code==10){return b"continue";}if(code==11){return b"assert";}
    if(code==12){return b"discard";}if(code==13){return b"defer";}if(code==14){return b"for";}if(code==16){return b"burn";}if(code==17){return b"region";}return b"";
}
// Go AST expressions also represent statement-only control nodes. Keep one
// exact field prefix for synthetic block, region, while and jump projections.
pub fn bool expression(ref u8[] kind,usize start,usize end,ref u8[] text,ref mut bytes::Buffer out,ref mut json::Ledger work){return json::raw(out,b"{\"TypeArguments\":null,\"Kind\":",work) && json::quoted(out,kind,work) && json::raw(out,b",\"Span\":",work) && json::span(out,start,end,work) && json::raw(out,b",\"Text\":",work) && json::quoted(out,text,work) && json::raw(out,b",\"Cooked\":\"\",\"Type\":null,\"Children\":",work);}
pub fn bool valid(ref syntax::Parsed p,usize id,ref mut json::Ledger work){if(id==0 || id>syntax::statement_count(p)){return json::reject(work,9);}let n=syntax::statement_at(p,id);return if(n.start<=n.end && n.end<=syntax::source_view(p).len){true}else{json::reject(work,9)};}
