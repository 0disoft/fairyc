module module_inline_ir;
import module_inline_identity as identity;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import generic_source_json as json;
import std::bytes as bytes;

fn bool scalar(ref clean::Program p,usize id){return id!=0 && id<=clean::type_count(p) && clean::type_at(p,id).kind==1 && clean::type_at(p,id).scalar==1;}
fn bool expression(ref ir::Body b,ref clean::Program p,usize id,usize arity,usize visible,ref mut usize count,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let index=ir::node(b,id);if(index==0 || *count>=64){return json::reject(work,9);};*count+=1;let n=ir::node_at(b,index-1);let e=n.value;if(e.state!=1 || !scalar(p,e.type)){return json::reject(work,9);}let op=ir::node_text(b,index-1);
    if(e.kind==3){return e.left<id && expression(b,p,e.left,arity,visible,count,out,work);}
    if(e.kind==1){let value=switch(identity::integer(op)){case .ok(n):{n}case .err(_):{return json::reject(work,9);}};return identity::leaf(out,b"literal",value,work);}
    if(e.kind==2){if(e.binding==0 || e.binding>arity+visible){return json::reject(work,9);}return identity::leaf(out,if(e.binding<=arity){b"path"}else{b"local"},if(e.binding<=arity){e.binding-1}else{e.binding-arity-1},work);}
    if(e.kind!=4 && e.kind!=5){return json::reject(work,9);}
    if(e.kind==4 && bytes::equal(op,b"-") && e.left!=0 && e.left<id){let child=ir::node(b,e.left);if(child!=0 && ir::node_at(b,child-1).value.kind==1){let v=switch(identity::integer(ir::node_text(b,child-1))){case .ok(v):{v}case .err(_):{return json::reject(work,9);}};if(v==2147483648){return identity::minimum(out,work);}}}
    if(!identity::operator(op,e.kind==5) || e.left>=id || !identity::branch(out,op,work) || !expression(b,p,e.left,arity,visible,count,out,work)){return json::reject(work,9);}if(e.kind==5 && (e.right>=id || !json::raw(out,b",",work) || !expression(b,p,e.right,arity,visible,count,out,work))){return json::reject(work,9);}return json::raw(out,b"]}",work);
}
// An absent marker is ordinary IR. A present marker must describe this exact
// pure detached body, including its result and original lexical parameter order.
pub fn u32 check(ref ir::Body b,ref clean::Program p,usize frame){
    if(identity::zero(ir::inline_hash(b))){return 0;}let f=clean::frame_at(p,frame);if(f.kind!=3 || f.effects!=0 || !scalar(p,f.result) || ir::local_count(b)>20 || ir::call_count(b)!=0 || ir::region_count(b)!=0 || ir::placement_count(b)!=0 || ir::reset_count(b)!=0){return 1001;}
    var usize arity=0;var usize i=1;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(!scalar(p,l.type) || l.origin.mutable){return 1001;}if(l.kind==1 && i==arity+1){arity+=1;}else if(l.kind!=2){return 1001;}i+=1;}if(arity>4){return 1001;}let locals=ir::local_count(b)-arity;if(locals>16){return 1001;}
    let root_index=ir::statement(b,ir::body(b));if(root_index==0){return 1001;}let root=ir::statement_at(b,root_index-1).value;if(root.kind!=1 || root.second!=locals+if(root.expression==0){1}else{0} || ir::statement_count(b)!=1+2*root.second){return 1001;}
    var bytes::Buffer out=bytes::empty();var json::Ledger work=json::empty();var usize count=0;if(!if(locals==0){identity::header(ref mut out,arity,ref mut work)}else{identity::locals_header(ref mut out,arity,ref mut work)}){return 1001;}
    var usize returned=root.expression;i=0;while(i<root.second){var usize link=root.first;var usize remaining=root.second-1;while(remaining>i){let idx=ir::statement(b,link);if(idx==0 || ir::statement_at(b,idx-1).value.kind!=7){return 1001;}link=ir::statement_at(b,idx-1).value.second;remaining-=1;}let idx=ir::statement(b,link);if(idx==0 || ir::statement_at(b,idx-1).value.kind!=7){return 1001;}let item=ir::statement(b,ir::statement_at(b,idx-1).value.first);if(item==0){return 1001;}let s=ir::statement_at(b,item-1).value;
        if(i<locals){let l=ir::local_at(b,arity+i+1);if(s.kind!=2 || s.entry!=l.source_index || s.expression!=l.initializer || !expression(b,p,s.expression,arity,i,ref mut count,ref mut out,ref mut work) || !json::raw(ref mut out,b",",ref mut work)){return 1001;}}else{if(s.kind!=3){return 1001;}returned=s.expression;}i+=1;}
    if(!expression(b,p,returned,arity,locals,ref mut count,ref mut out,ref mut work) || locals!=0 && !json::raw(ref mut out,b"]}",ref mut work)){return 1001;}let hash=switch(identity::finish(ref mut out,ref mut work)){case .ok(h):{h}case .err(_):{return 1001;}};return if(identity::equal(hash,ir::inline_hash(b))){0}else{1001};
}
