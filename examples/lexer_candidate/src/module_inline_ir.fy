module module_inline_ir;
import module_inline_identity as identity;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import generic_source_json as json;
import std::bytes as bytes;

struct Walk {usize nodes;usize calls;usize block_count;usize[64] blocks;}
fn bool scalar_kind(ref clean::Program p,usize id,u32 kind){return id!=0 && id<=clean::type_count(p) && clean::type_at(p,id).kind==1 && clean::type_at(p,id).scalar==kind;}
fn bool scalar(ref clean::Program p,usize id){return scalar_kind(p,id,1);}
fn bool condition(ref ir::Body b,ref clean::Program p,usize id,usize arity,usize visible,usize[20] versions,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let index=ir::node(b,id);
    if(index==0 || walk.nodes>=64){return json::reject(work,9);}
    walk.nodes+=1;
    let e=ir::node_at(b,index-1).value;
    if(e.state!=1 || !scalar_kind(p,e.type,6)){return json::reject(work,9);}
    if(e.kind==3){return e.left<id && condition(b,p,e.left,arity,visible,versions,walk,out,work);}
    let op=ir::node_text(b,index-1);
    if(e.kind==1){return e.left==0 && e.right==0 && identity::boolean(out,op,work);}
    return e.kind==5 && e.left<id && e.right<id && identity::comparison(op) &&
        identity::branch(out,op,work) && expression(b,p,e.left,arity,visible,versions,walk,out,work) &&
        json::raw(out,b",",work) && expression(b,p,e.right,arity,visible,versions,walk,out,work) && json::raw(out,b"]}",work);
}
fn bool value_block(ref ir::Body b,ref clean::Program p,usize block,usize parent,usize arity,usize visible,usize[20] versions,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let index=ir::statement(b,block);
    if(index==0 || walk.nodes>=64 || walk.block_count>=64){return json::reject(work,9);}
    let s=ir::statement_at(b,index-1).value;
    if(s.kind!=1 || s.first!=0 || s.second!=0 || s.expression==0 || s.expression>=parent){return json::reject(work,9);}
    var usize i=0;
    while(i<walk.block_count){if(walk.blocks[i]==block){return json::reject(work,9);}i+=1;}
    let slot=walk.block_count;
    walk.blocks[slot]=block;
    walk.block_count+=1;
    walk.nodes+=1;
    return expression(b,p,s.expression,arity,visible,versions,walk,out,work);
}
fn bool expression(ref ir::Body b,ref clean::Program p,usize id,usize arity,usize visible,usize[20] versions,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let index=ir::node(b,id);if(index==0 || walk.nodes>=64){return json::reject(work,9);}walk.nodes+=1;let n=ir::node_at(b,index-1);let e=n.value;if(e.state!=1 || !scalar(p,e.type)){return json::reject(work,9);}let op=ir::node_text(b,index-1);
    if(e.kind==3){return e.left<id && expression(b,p,e.left,arity,visible,versions,walk,out,work);}
    if(e.kind==13){
        let pair=ir::node(b,e.right);
        if(e.left>=id || pair==0 || e.right>=id){return json::reject(work,9);}
        let arms=ir::node_at(b,pair-1).value;
        if(arms.kind!=14 || arms.state!=2 || arms.right>=e.right){return json::reject(work,9);}
        if(!identity::branch(out,b"if",work) || !condition(b,p,e.left,arity,visible,versions,walk,out,work) ||
            !json::raw(out,b",",work) || !value_block(b,p,arms.left,e.right,arity,visible,versions,walk,out,work) ||
            !json::raw(out,b",",work)){return json::reject(work,9);}
        let alternative=ir::node(b,arms.right);
        if(alternative==0){return json::reject(work,9);}
        let other=ir::node_at(b,alternative-1).value;
        if(other.kind==12){
            if(other.state!=1 || other.type!=e.type || !value_block(b,p,other.left,arms.right,arity,visible,versions,walk,out,work)){return json::reject(work,9);}
        }else if(other.kind!=13 || !expression(b,p,arms.right,arity,visible,versions,walk,out,work)){return json::reject(work,9);}
        return json::raw(out,b"]}",work);
    }
    if(e.kind==1){let value=switch(identity::integer(op)){case .ok(n):{n}case .err(_):{return json::reject(work,9);}};return identity::leaf(out,b"literal",value,work);}
    if(e.kind==2){if(e.binding==0 || e.binding>20){return json::reject(work,9);}if(e.binding<=arity){return identity::leaf(out,b"path",e.binding-1,work);}let version=versions[e.binding-1];if(version==0 || version>visible){return json::reject(work,9);}return identity::leaf(out,b"local",version-1,work);}
    if(e.kind==6){
        let target=ir::node(b,e.left);if(target==0 || e.left>=id || e.builtin.kind!=0){return json::reject(work,9);}let path=ir::node_at(b,target-1).value;
        if(path.kind!=2 || path.state!=2 || path.binding!=0 || path.builtin.kind!=0){return json::reject(work,9);}var usize found=0;var usize i=0;
        while(i<ir::call_count(b)){let c=ir::call_at(b,i);if(c.source_index==id){if(found!=0){return json::reject(work,9);}found=i+1;}i+=1;}
        if(found==0){return json::reject(work,9);}let c=ir::call_at(b,found-1);
        if(c.kind!=1 || !c.checked || c.instance!=0 || c.expression!=index || c.start!=e.start || c.end!=e.end || c.argument_count>4 || c.argument_start>ir::argument_count(b) || c.argument_count>ir::argument_count(b)-c.argument_start || !identity::call(out,ir::node_text(b,target-1),c.argument_count!=0,work)){return json::reject(work,9);}
        walk.calls+=1;var usize link=e.right;i=c.argument_count;
        // Syntax argument lists are reverse linked; retained arguments are in
        // source order. Check both representations before hashing that order.
        while(i>0){i-=1;let pair=ir::node(b,link);if(pair==0 || link>=id){return json::reject(work,9);}let row=ir::node_at(b,pair-1).value;let a=ir::argument_at(b,c.argument_start+i);
            if(row.kind!=7 || row.left!=a.source_index || a.invocation!=found || a.position!=i || !a.checked || a.shared || a.label_start!=a.label_end || a.expression==0 || a.expression>ir::node_count(b) || ir::node_at(b,a.expression-1).value.source_index!=a.source_index || !scalar(p,a.expected) || a.actual!=a.expected){return json::reject(work,9);}link=row.right;}
        if(link!=0){return json::reject(work,9);}i=0;while(i<c.argument_count){let a=ir::argument_at(b,c.argument_start+i);if(a.source_index>=id || i!=0 && !json::raw(out,b",",work) || !expression(b,p,a.source_index,arity,visible,versions,walk,out,work)){return json::reject(work,9);}i+=1;}
        return c.argument_count==0 || json::raw(out,b"]}",work);
    }
    if(e.kind!=4 && e.kind!=5){return json::reject(work,9);}
    if(e.kind==4 && bytes::equal(op,b"-") && e.left!=0 && e.left<id){let child=ir::node(b,e.left);if(child!=0 && ir::node_at(b,child-1).value.kind==1){let v=switch(identity::integer(ir::node_text(b,child-1))){case .ok(v):{v}case .err(_):{return json::reject(work,9);}};if(v==2147483648){return identity::minimum(out,work);}}}
    if(!identity::operator(op,e.kind==5) || e.left>=id || !identity::branch(out,op,work) || !expression(b,p,e.left,arity,visible,versions,walk,out,work)){return json::reject(work,9);}if(e.kind==5 && (e.right>=id || !json::raw(out,b",",work) || !expression(b,p,e.right,arity,visible,versions,walk,out,work))){return json::reject(work,9);}return json::raw(out,b"]}",work);
}
// An absent marker is ordinary IR. A present marker must describe this exact
// pure detached body, including its result and original lexical parameter order.
pub fn u32 check(ref ir::Body b,ref clean::Program p,usize frame){
    if(identity::zero(ir::inline_hash(b))){return 0;}let f=clean::frame_at(p,frame);if(f.kind!=3 || f.effects!=0 || !scalar(p,f.result) || ir::local_count(b)>20 || ir::call_count(b)>64 || ir::argument_count(b)>256 || ir::region_count(b)!=0 || ir::placement_count(b)!=0 || ir::reset_count(b)!=0){return 1001;}
    var usize arity=0;var usize i=1;var bool mutating=false;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(!scalar(p,l.type)){return 1001;}if(l.kind==1 && i==arity+1){if(l.origin.mutable){return 1001;}arity+=1;}else if(l.kind==2 || l.kind==3){mutating=mutating || l.kind==3;}else{return 1001;}i+=1;}if(arity>4){return 1001;}let locals=ir::local_count(b)-arity;if(locals>16){return 1001;}
    let root_index=ir::statement(b,ir::body(b));if(root_index==0){return 1001;}let root=ir::statement_at(b,root_index-1).value;if(root.kind!=1 || root.second<if(root.expression==0){1}else{0} || root.second-if(root.expression==0){1}else{0}>64){return 1001;}
    let min_statements=1+2*root.second;if(ir::statement_count(b)<min_statements){return 1001;}
    var bytes::Buffer out=bytes::empty();var json::Ledger work=json::empty();var Walk walk=Walk{.nodes=0,.calls=0,.block_count=0,.blocks=[0;64]};let conditional=ir::statement_count(b)>1+2*root.second;if(!if(conditional){identity::conditional_header(ref mut out,arity,locals!=0,mutating,ref mut work)}else if(mutating){identity::steps_header(ref mut out,arity,ref mut work)}else if(ir::call_count(b)!=0){identity::calls_header(ref mut out,arity,locals!=0,ref mut work)}else if(locals==0){identity::header(ref mut out,arity,ref mut work)}else{identity::locals_header(ref mut out,arity,ref mut work)}){return 1001;}
    var usize returned=root.expression;var usize declarations=0;var usize[20] versions=[0;20];let steps=root.second-if(root.expression==0){1}else{0};i=0;while(i<root.second){var usize link=root.first;var usize remaining=root.second-1;while(remaining>i){let idx=ir::statement(b,link);if(idx==0 || ir::statement_at(b,idx-1).value.kind!=7){return 1001;}link=ir::statement_at(b,idx-1).value.second;remaining-=1;}let idx=ir::statement(b,link);if(idx==0 || ir::statement_at(b,idx-1).value.kind!=7){return 1001;}let item=ir::statement(b,ir::statement_at(b,idx-1).value.first);if(item==0){return 1001;}let s=ir::statement_at(b,item-1).value;
        if(i<steps){var usize binding=0;if(s.kind==2){if(declarations>=locals){return 1001;}binding=arity+declarations+1;let l=ir::local_at(b,binding);if(s.entry!=l.source_index || s.expression!=l.initializer){return 1001;}declarations+=1;}
        else if(s.kind==4){let target=ir::node(b,s.target);if(target==0 || !bytes::equal(ir::assignment_operator(b,item-1),b"=")){return 1001;}let e=ir::node_at(b,target-1).value;binding=e.binding;if(e.kind!=2 || e.state!=1 || !scalar(p,e.type) || binding<=arity || binding>ir::local_count(b) || ir::local_at(b,binding).kind!=3 || versions[binding-1]==0){return 1001;}}else{return 1001;}
        if(!expression(b,p,s.expression,arity,i,versions,ref mut walk,ref mut out,ref mut work) || !json::raw(ref mut out,b",",ref mut work)){return 1001;}versions[binding-1]=i+1;}else{if(s.kind!=3){return 1001;}returned=s.expression;}i+=1;}if(declarations!=locals){return 1001;}
    if(!expression(b,p,returned,arity,steps,versions,ref mut walk,ref mut out,ref mut work) || walk.calls!=ir::call_count(b) || walk.block_count!=ir::statement_count(b)-min_statements || locals!=0 && !json::raw(ref mut out,b"]}",ref mut work)){return 1001;}let hash=switch(identity::finish(ref mut out,ref mut work)){case .ok(h):{h}case .err(_):{return 1001;}};return if(identity::equal(hash,ir::inline_hash(b))){0}else{1001};
}
