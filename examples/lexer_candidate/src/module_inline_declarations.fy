module module_inline_declarations;
import syntax as syntax;
import type_state as scalars;
import module_inline_identity as identity;
import generic_source_json as json;
import std::bytes as bytes;

pub struct Info{bool annotated;u32 code;usize start;usize end;usize expression;usize parameters;u32[8] hash;}
fn Info invalid(Info info){var Info out=info;out.code=3;return out;}
fn bool i32_type(ref syntax::Parsed p,usize root){return root!=0 && syntax::type_at(p,root).kind==1 && scalars::primitive(p,root)==1;}
fn bool expression(ref syntax::Parsed p,syntax::Declaration d,usize id,usize visible,ref mut usize count,ref mut bytes::Buffer out,ref mut json::Ledger work){
    if(id==0 || id>syntax::expression_count(p) || *count>=64){return json::reject(work,9);};*count+=1;let e=syntax::expression_at(p,id);
    if(e.kind==3){return e.left<id && expression(p,d,e.left,visible,count,out,work);}
    if(e.kind==1){let n=switch(identity::integer(syntax::expression_text(p,id))){case .ok(n):{n}case .err(_):{return json::reject(work,9);}};return identity::leaf(out,b"literal",n,work);}
    if(e.kind==2){let name=syntax::expression_text(p,id);var usize local=visible;while(local>0){local-=1;let st=syntax::statement_at(p,syntax::block_statement(p,d.body,local));let a=syntax::entry_at(p,st.entry-1);if(bytes::equal(name,syntax::source_view(p)[a.name_start..a.name_end])){return identity::leaf(out,b"local",local,work);}}var usize i=d.entry_start;var usize position=0;while(i<d.entry_start+d.entry_count){let a=syntax::entry_at(p,i);if(a.kind==1){if(bytes::equal(name,syntax::source_view(p)[a.name_start..a.name_end])){return identity::leaf(out,b"path",position,work);}position+=1;}i+=1;}return json::reject(work,9);}
    if(e.kind!=4 && e.kind!=5){return json::reject(work,9);}let op=syntax::expression_text(p,id);
    if(e.kind==4 && bytes::equal(op,b"-") && e.left!=0 && e.left<id && syntax::expression_at(p,e.left).kind==1){let n=switch(identity::integer(syntax::expression_text(p,e.left))){case .ok(n):{n}case .err(_):{return json::reject(work,9);}};if(n==2147483648){return identity::minimum(out,work);}}
    if(!identity::operator(op,e.kind==5) || e.left>=id || !identity::branch(out,op,work) || !expression(p,d,e.left,visible,count,out,work)){return json::reject(work,9);}
    if(e.kind==5 && (e.right>=id || !json::raw(out,b",",work) || !expression(p,d,e.right,visible,count,out,work))){return json::reject(work,9);}return json::raw(out,b"]}",work);
}
pub fn Info read(ref syntax::Parsed p,usize declaration){
    let d=syntax::declaration_at(p,declaration);var Info info=Info{.annotated=false,.code=0,.start=d.name_start,.end=d.name_end,.expression=0,.parameters=0,.hash=[0;8]};var usize i=0;
    while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"inline")){info.start=a.start;info.end=a.end;if(info.annotated || a.argument_count!=0){return invalid(info);}info.annotated=true;}i+=1;}
    i=0;while(i<d.cleanup_attribute_count){let a=syntax::attribute_at(p,d.cleanup_attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"inline")){info.start=a.start;info.end=a.end;return invalid(info);}i+=1;}
    if(!info.annotated){return info;}if(d.kind!=3 || !d.public || d.unsafe_end!=0 || d.owner_end!=0 || d.generic_parameters!=0 || !i32_type(p,d.result_type) || d.body==0){return invalid(info);}
    i=0;while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"effects")){return invalid(info);}i+=1;}
    i=d.entry_start;while(i<d.entry_start+d.entry_count){let a=syntax::entry_at(p,i);if(a.kind==1){if(!i32_type(p,a.type_root)){return invalid(info);}info.parameters+=1;}i+=1;}if(info.parameters>4){return invalid(info);}
    let b=syntax::statement_at(p,d.body);if(b.kind!=1){return invalid(info);}let count=syntax::block_statement_count(p,d.body);var usize locals=count;if(b.expression!=0){info.expression=b.expression;}else{if(count==0){return invalid(info);}let last=syntax::statement_at(p,syntax::block_statement(p,d.body,count-1));if(last.kind!=3 || last.expression==0){return invalid(info);}info.expression=last.expression;locals-=1;}if(locals>16){return invalid(info);}
    i=0;while(i<locals){let s=syntax::statement_at(p,syntax::block_statement(p,d.body,i));if(s.kind!=2 || s.entry==0 || s.expression==0){return invalid(info);}let a=syntax::entry_at(p,s.entry-1);if(a.kind!=2 || a.type_root!=0 && !i32_type(p,a.type_root)){return invalid(info);}i+=1;}
    var bytes::Buffer out=bytes::empty();var json::Ledger work=json::empty();var usize nodes=0;
    if(!if(locals==0){identity::header(ref mut out,info.parameters,ref mut work)}else{identity::locals_header(ref mut out,info.parameters,ref mut work)}){return invalid(info);}
    i=0;while(i<locals){let s=syntax::statement_at(p,syntax::block_statement(p,d.body,i));if(!expression(p,d,s.expression,i,ref mut nodes,ref mut out,ref mut work) || !json::raw(ref mut out,b",",ref mut work)){return invalid(info);}i+=1;}
    if(!expression(p,d,info.expression,locals,ref mut nodes,ref mut out,ref mut work) || locals!=0 && !json::raw(ref mut out,b"]}",ref mut work)){return invalid(info);}info.hash=switch(identity::finish(ref mut out,ref mut work)){case .ok(h):{h}case .err(_):{return invalid(info);}};return info;
}
