module module_inline_declarations;
import syntax as syntax;
import type_state as scalars;
import module_inline_identity as identity;
import generic_source_json as json;
import std::bytes as bytes;
import std::vec as v;

pub struct Info{bool annotated;u32 code;usize start;usize end;usize expression;usize parameters;u32[8] hash;}
struct Row{usize declaration;usize edges;Info info;}
fn Info invalid(Info info){var Info out=info;out.code=3;return out;}
fn bool i32_type(ref syntax::Parsed p,usize root){return root!=0 && syntax::type_at(p,root).kind==1 && scalars::primitive(p,root)==1;}
fn ref u8[] local_name(ref syntax::Parsed p,syntax::Statement s){if(s.kind==2 && s.entry!=0){let a=syntax::entry_at(p,s.entry-1);return syntax::source_view(p)[a.name_start..a.name_end];}if(s.kind==4 && s.target!=0 && syntax::expression_at(p,s.target).kind==2){return syntax::expression_text(p,s.target);}return b"";}
fn bool bound(ref syntax::Parsed p,syntax::Declaration d,ref u8[] name,usize visible){var usize i=0;while(i<visible){let s=syntax::statement_at(p,syntax::block_statement(p,d.body,i));if(bytes::equal(name,local_name(p,s))){return true;}i+=1;}i=d.entry_start;while(i<d.entry_start+d.entry_count){let a=syntax::entry_at(p,i);if(a.kind==1 && bytes::equal(name,syntax::source_view(p)[a.name_start..a.name_end])){return true;}i+=1;}return false;}
fn bool has_calls(ref syntax::Parsed p,syntax::Declaration d){let b=syntax::statement_at(p,d.body);var usize i=1;while(i<=syntax::expression_count(p)){let e=syntax::expression_at(p,i);if(e.kind==6 && e.start>=b.start && e.end<=b.end){return true;}i+=1;}return false;}
fn bool edge(ref syntax::Parsed p,syntax::Declaration d,usize target,ref mut v::Vec<Row> graph){var usize owner=64;var usize callee=64;var usize i=0;while(i<v::len<Row>(graph)){let row=*v::at<Row>(graph,i);if(syntax::declaration_at(p,row.declaration).name_start==d.name_start){owner=i;}if(row.declaration==target){callee=i;}i+=1;}if(owner==64 || callee==64){return false;}let usize one=1;let row=v::at_mut<Row>(graph,owner);row.edges=row.edges|(one<<callee);return true;}
fn bool expression(ref syntax::Parsed p,syntax::Declaration d,usize id,usize visible,ref mut usize count,ref mut bytes::Buffer out,ref mut json::Ledger work,ref mut v::Vec<Row> graph){
    if(id==0 || id>syntax::expression_count(p) || *count>=64){return json::reject(work,9);};*count+=1;let e=syntax::expression_at(p,id);
    if(e.kind==3){return e.left<id && expression(p,d,e.left,visible,count,out,work,graph);}
    if(e.kind==1){let n=switch(identity::integer(syntax::expression_text(p,id))){case .ok(n):{n}case .err(_):{return json::reject(work,9);}};return identity::leaf(out,b"literal",n,work);}
    if(e.kind==2){let name=syntax::expression_text(p,id);var usize local=visible;while(local>0){local-=1;let st=syntax::statement_at(p,syntax::block_statement(p,d.body,local));if(bytes::equal(name,local_name(p,st))){return identity::leaf(out,b"local",local,work);}}var usize i=d.entry_start;var usize position=0;while(i<d.entry_start+d.entry_count){let a=syntax::entry_at(p,i);if(a.kind==1){if(bytes::equal(name,syntax::source_view(p)[a.name_start..a.name_end])){return identity::leaf(out,b"path",position,work);}position+=1;}i+=1;}return json::reject(work,9);}
    if(e.kind==6){
        if(e.left==0 || e.left>=id || syntax::expression_at(p,e.left).kind!=2){return json::reject(work,9);}let name=syntax::expression_text(p,e.left);if(bound(p,d,name,visible)){return json::reject(work,9);}
        var usize target=syntax::declaration_count(p);var usize i=0;while(i<syntax::declaration_count(p)){let candidate=syntax::declaration_at(p,i);if(candidate.kind==3 && candidate.owner_end==0 && bytes::equal(name,syntax::source_view(p)[candidate.name_start..candidate.name_end])){if(target!=syntax::declaration_count(p)){return json::reject(work,9);}target=i;}i+=1;}if(target==syntax::declaration_count(p)){return json::reject(work,9);}
        let callee=read_graph(p,target,graph);let arguments=syntax::call_argument_count(p,id);if(!callee.annotated || callee.code!=0 || arguments!=callee.parameters || !edge(p,d,target,graph) || !identity::call(out,name,arguments!=0,work)){return json::reject(work,9);}i=0;while(i<arguments){let argument=syntax::call_argument(p,id,i);if(syntax::call_argument_labeled(p,syntax::call_argument_entry(p,id,i)) || argument==0 || argument>=id || i!=0 && !json::raw(out,b",",work) || !expression(p,d,argument,visible,count,out,work,graph)){return json::reject(work,9);}i+=1;}return arguments==0 || json::raw(out,b"]}",work);
    }
    if(e.kind!=4 && e.kind!=5){return json::reject(work,9);}let op=syntax::expression_text(p,id);
    if(e.kind==4 && bytes::equal(op,b"-") && e.left!=0 && e.left<id && syntax::expression_at(p,e.left).kind==1){let n=switch(identity::integer(syntax::expression_text(p,e.left))){case .ok(n):{n}case .err(_):{return json::reject(work,9);}};if(n==2147483648){return identity::minimum(out,work);}}
    if(!identity::operator(op,e.kind==5) || e.left>=id || !identity::branch(out,op,work) || !expression(p,d,e.left,visible,count,out,work,graph)){return json::reject(work,9);}
    if(e.kind==5 && (e.right>=id || !json::raw(out,b",",work) || !expression(p,d,e.right,visible,count,out,work,graph))){return json::reject(work,9);}return json::raw(out,b"]}",work);
}
fn Info signature(ref syntax::Parsed p,usize declaration){
    let d=syntax::declaration_at(p,declaration);var Info info=Info{.annotated=false,.code=0,.start=d.name_start,.end=d.name_end,.expression=0,.parameters=0,.hash=[0;8]};var usize i=0;
    while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"inline")){info.start=a.start;info.end=a.end;if(info.annotated || a.argument_count!=0){return invalid(info);}info.annotated=true;}i+=1;}
    i=0;while(i<d.cleanup_attribute_count){let a=syntax::attribute_at(p,d.cleanup_attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"inline")){info.start=a.start;info.end=a.end;return invalid(info);}i+=1;}
    if(!info.annotated){return info;}if(d.kind!=3 || !d.public || d.unsafe_end!=0 || d.owner_end!=0 || d.generic_parameters!=0 || !i32_type(p,d.result_type) || d.body==0){return invalid(info);}
    i=0;while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"effects")){return invalid(info);}i+=1;}
    i=d.entry_start;while(i<d.entry_start+d.entry_count){let a=syntax::entry_at(p,i);if(a.kind==1){if(!i32_type(p,a.type_root)){return invalid(info);}info.parameters+=1;}i+=1;}if(info.parameters>4){return invalid(info);}
    return info;
}
fn Info read_body(ref syntax::Parsed p,usize declaration,ref mut v::Vec<Row> graph){
    let d=syntax::declaration_at(p,declaration);var Info info=signature(p,declaration);if(!info.annotated || info.code!=0){return info;}var usize i=0;
    let b=syntax::statement_at(p,d.body);if(b.kind!=1){return invalid(info);}let count=syntax::block_statement_count(p,d.body);var usize locals=count;if(b.expression!=0){info.expression=b.expression;}else{if(count==0){return invalid(info);}let last=syntax::statement_at(p,syntax::block_statement(p,d.body,count-1));if(last.kind!=3 || last.expression==0){return invalid(info);}info.expression=last.expression;locals-=1;}if(locals>64){return invalid(info);}var usize declarations=0;var bool mutating=false;
    i=0;while(i<locals){let id=syntax::block_statement(p,d.body,i);let s=syntax::statement_at(p,id);if(s.expression==0){return invalid(info);}if(s.kind==2){if(s.entry==0){return invalid(info);}let a=syntax::entry_at(p,s.entry-1);if((a.kind!=2 && a.kind!=3) || a.type_root!=0 && !i32_type(p,a.type_root)){return invalid(info);}declarations+=1;mutating=mutating || a.kind==3;}
        else if(s.kind==4){let name=local_name(p,s);if(name.len==0 || !bytes::equal(syntax::assignment_operator(p,id),b"=")){return invalid(info);}var bool writable=false;var usize j=0;while(j<i){let old=syntax::statement_at(p,syntax::block_statement(p,d.body,j));if(old.kind==2 && bytes::equal(name,local_name(p,old))){writable=syntax::entry_at(p,old.entry-1).kind==3;}j+=1;}if(!writable){return invalid(info);}mutating=true;}else{return invalid(info);}i+=1;}if(declarations>16){return invalid(info);}
    var bytes::Buffer out=bytes::empty();var json::Ledger work=json::empty();var usize nodes=0;
    if(!if(mutating){identity::steps_header(ref mut out,info.parameters,ref mut work)}else if(has_calls(p,d)){identity::calls_header(ref mut out,info.parameters,locals!=0,ref mut work)}else if(locals==0){identity::header(ref mut out,info.parameters,ref mut work)}else{identity::locals_header(ref mut out,info.parameters,ref mut work)}){return invalid(info);}
    i=0;while(i<locals){let s=syntax::statement_at(p,syntax::block_statement(p,d.body,i));if(!expression(p,d,s.expression,i,ref mut nodes,ref mut out,ref mut work,graph) || !json::raw(ref mut out,b",",ref mut work)){return invalid(info);}i+=1;}
    if(!expression(p,d,info.expression,locals,ref mut nodes,ref mut out,ref mut work,graph) || locals!=0 && !json::raw(ref mut out,b"]}",ref mut work)){return invalid(info);}info.hash=switch(identity::finish(ref mut out,ref mut work)){case .ok(h):{h}case .err(_):{return invalid(info);}};return info;
}
// Collect signatures without recursively checking bodies. A long valid chain
// must not keep sixty-four complete body-checker frames on the native stack.
fn Info read_graph(ref syntax::Parsed p,usize declaration,ref mut v::Vec<Row> graph){
    var usize i=0;while(i<v::len<Row>(graph)){let row=*v::at<Row>(graph,i);if(row.declaration==declaration){return row.info;}i+=1;}let info=signature(p,declaration);if(!info.annotated || info.code!=0){return info;}if(i>=64 || !v::push<Row>(graph,Row{.declaration,.edges=0,.info}).is_ok){return invalid(info);}return info;
}
fn bool acyclic(ref v::Vec<Row> graph){
    let usize one=1;var usize done=0;var usize count=0;while(count<v::len<Row>(graph)){let previous=count;var usize i=0;while(i<v::len<Row>(graph)){let bit=one<<i;let row=*v::at<Row>(graph,i);if((done&bit)==0 && (row.edges&done)==row.edges){done=done|bit;count+=1;}i+=1;}if(count==previous){return false;}}return true;
}
pub fn Info read(ref syntax::Parsed p,usize declaration){
    // Ordinary declarations keep the allocation-free path. Most compiler
    // sources contain no inline annotation; do not allocate one graph per fn.
    let d=syntax::declaration_at(p,declaration);var bool requested=false;var usize i=0;while(i<d.attribute_count+d.cleanup_attribute_count){let a=syntax::attribute_at(p,if(i<d.attribute_count){d.attribute_start+i}else{d.cleanup_attribute_start+i-d.attribute_count});if(bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"inline")){requested=true;break;}i+=1;}
    if(!requested){return Info{.annotated=false,.code=0,.start=d.name_start,.end=d.name_end,.expression=0,.parameters=0,.hash=[0;8]};}var v::Vec<Row> graph=v::empty<Row>();let root=read_graph(p,declaration,ref mut graph);if(!root.annotated || root.code!=0){return root;}i=0;while(i<v::len<Row>(ref graph)){let selected=v::at<Row>(ref graph,i).declaration;let info=read_body(p,selected,ref mut graph);if(!info.annotated || info.code!=0){return invalid(root);}{let row=v::at_mut<Row>(ref mut graph,i);row.info=info;}i+=1;}return if(acyclic(ref graph)){v::at<Row>(ref graph,0).info}else{invalid(root)};
}
