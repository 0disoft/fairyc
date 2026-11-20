module module_inline_ir;
import module_inline_identity as identity;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import generic_source_json as json;
import std::bytes as bytes;
import syntax as syntax;

struct Walk {
    usize nodes; usize calls; usize block_count; usize[64] blocks;
    usize arm_statements; usize depth; usize declarations; usize declared;
    bool statement_mode; usize loop_depth; usize[4] visited; usize visited_count;
}
fn bool declare(ref mut Walk walk,usize binding,usize arity){
    if(binding<=arity || binding>20){return false;}
    let usize one=1;
    let bit=one<<(binding-1);
    if((walk.declared&bit)!=0){return false;}
    walk.declared=walk.declared|bit;
    walk.declarations+=1;
    return true;
}
// Block entries are reverse linked. Fetch the nth statement in source order.
fn usize block_link(ref ir::Body b,syntax::Statement block,usize position){
    var usize link=block.first;
    var usize remaining=block.second-position;
    while(remaining>0){
        let index=ir::statement(b,link);
        if(index==0){return 0;}
        let row=ir::statement_at(b,index-1).value;
        if(row.kind!=7 || row.second>=link || row.first>=link){return 0;}
        if(remaining==1){
            if(position==0 && row.second!=0){return 0;}
            return index;
        }
        link=row.second;
        remaining-=1;
    }
    return 0;
}
fn bool visit_statement(ref mut Walk walk,usize index){
    if(!walk.statement_mode){return true;}
    if(index==0 || index>256){return false;}
    let bucket=(index-1)/64;
    let usize one=1;
    let mask=one<<((index-1)%64);
    let previous=walk.visited[bucket];
    if((previous&mask)!=0){return false;}
    walk.visited[bucket]=previous|mask;
    walk.visited_count+=1;
    return true;
}
fn usize block_item(ref ir::Body b,syntax::Statement block,usize position,ref mut Walk walk){
    let link=block_link(b,block,position);
    if(link==0 || !visit_statement(walk,link)){return 0;}
    let item=ir::statement(b,ir::statement_at(b,link-1).value.first);
    return if(visit_statement(walk,item)){item}else{0};
}
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
    if(e.kind==4 && bytes::equal(op,b"!")){
        return e.left<id && e.right==0 && identity::branch(out,op,work) &&
            condition(b,p,e.left,arity,visible,versions,walk,out,work) && json::raw(out,b"]}",work);
    }
    if(e.kind==5 && identity::logical(op)){
        return e.left<id && e.right<id && identity::branch(out,op,work) &&
            condition(b,p,e.left,arity,visible,versions,walk,out,work) && json::raw(out,b",",work) &&
            condition(b,p,e.right,arity,visible,versions,walk,out,work) && json::raw(out,b"]}",work);
    }
    return e.kind==5 && e.left<id && e.right<id && identity::comparison(op) &&
        identity::branch(out,op,work) && expression(b,p,e.left,arity,visible,versions,walk,out,work) &&
        json::raw(out,b",",work) && expression(b,p,e.right,arity,visible,versions,walk,out,work) && json::raw(out,b"]}",work);
}
fn bool value_block(ref ir::Body b,ref clean::Program p,usize block,usize parent,usize arity,usize visible,usize[20] versions,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let index=ir::statement(b,block);
    if(index==0 || walk.nodes>=64 || walk.block_count>=64 || !visit_statement(walk,index)){return json::reject(work,9);}
    let s=ir::statement_at(b,index-1).value;
    if(s.kind!=1 || s.expression==0 || s.expression>=parent || s.second>16 || (s.first==0)!=(s.second==0)){return json::reject(work,9);}
    var usize i=0;
    while(i<walk.block_count){if(walk.blocks[i]==block){return json::reject(work,9);}i+=1;}
    let slot=walk.block_count;
    walk.blocks[slot]=block;
    walk.block_count+=1;
    walk.nodes+=1;
    if(s.second==0){return expression(b,p,s.expression,arity,visible,versions,walk,out,work);}
    if(!identity::branch(out,b"arm-bindings",work)){return json::reject(work,9);}
    let depth=walk.depth;
    var usize[20] active=versions;
    i=0;
    while(i<s.second){
        let item=block_item(b,s,i,walk);
        if(item==0){return json::reject(work,9);}
        let st=ir::statement_at(b,item-1).value;
        if(st.kind!=2 || st.expression==0 || st.expression>=parent){return json::reject(work,9);}
        let binding=ir::local(b,st.entry);
        if(!declare(walk,binding,arity)){return json::reject(work,9);}
        let local=ir::local_at(b,binding);
        if(local.kind!=2 || local.origin.mutable || local.initializer!=st.expression ||
            !expression(b,p,st.expression,arity,visible,active,walk,out,work) ||
            !json::raw(out,b",",work)){return json::reject(work,9);}
        active[binding-1]=identity::arm_position(walk.depth)+1;
        walk.depth+=1;
        walk.arm_statements+=2;
        i+=1;
    }
    let ok=expression(b,p,s.expression,arity,visible,active,walk,out,work) && json::raw(out,b"]}",work);
    walk.depth=depth;
    return ok;
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
    if(e.kind==2){if(e.binding==0 || e.binding>20){return json::reject(work,9);}if(e.binding<=arity){return identity::leaf(out,b"path",e.binding-1,work);}let version=versions[e.binding-1];if(version==0 || version>visible && version<=identity::arm_position(0)){return json::reject(work,9);}return identity::leaf(out,b"local",version-1,work);}
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
// State 0 rejects the body, 1 falls through, and 2 returns on every path.
fn u32 control_if(ref ir::Body b,ref clean::Program p,usize id,usize arity,usize[20] active,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let index=ir::node(b,id);
    if(index==0 || walk.nodes>=64){return 0;}
    let e=ir::node_at(b,index-1).value;
    if(e.kind!=13 || e.state!=1 || !scalar_kind(p,e.type,7) || e.left>=id || e.right>=id){return 0;}
    let pair=ir::node(b,e.right);
    if(pair==0){return 0;}
    let arms=ir::node_at(b,pair-1).value;
    if(arms.kind!=14 || arms.state!=2 || arms.right>=e.right){return 0;}
    walk.nodes+=1;
    if(!identity::branch(out,b"if-statement",work) || !condition(b,p,e.left,arity,0,active,walk,out,work) ||
        !json::raw(out,b",",work) || walk.nodes>=64){return 0;}
    walk.nodes+=1;
    let left=control_block(b,p,arms.left,e.right,false,arity,active,walk,out,work);
    if(left==0){return 0;}
    var u32 right=1;
    if(arms.right!=0){
        let alternative=ir::node(b,arms.right);
        if(alternative==0 || !json::raw(out,b",",work)){return 0;}
        let other=ir::node_at(b,alternative-1).value;
        if(other.kind==13){right=control_if(b,p,arms.right,arity,active,walk,out,work);}
        else if(other.kind==12 && other.state==1 && other.type==e.type && walk.nodes<64){
            walk.nodes+=1;
            right=control_block(b,p,other.left,arms.right,false,arity,active,walk,out,work);
        }else{return 0;}
    }
    return if(right==0 || !json::raw(out,b"]}",work)){0}else if(left==2 && right==2){2}else{1};
}
struct Slots { usize[20] versions; }
fn bool loop_clause(ref ir::Body b,ref clean::Program p,usize id,usize parent,usize arity,bool initializer,ref mut Slots active,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    if(id==0){return identity::omitted(out,work);}
    let item=ir::statement(b,id);
    if(item==0 || !visit_statement(walk,item)){return false;}
    let clause=ir::statement_at(b,item-1).value;
    return (clause.kind==4 || initializer && clause.kind==2) &&
        control_item(b,p,item,parent,arity,active,walk,out,work)==1;
}
fn u32 control_loop(ref ir::Body b,ref clean::Program p,ir::Statement row,usize parent,usize arity,usize[20] versions,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let s=row.value;
    if(s.first==0 || s.first>=row.source_index){return 0;}
    let depth=walk.depth;
    var Slots active=Slots{.versions};
    if(s.kind==8){
        if(!identity::branch(out,b"while",work) ||
            !condition(b,p,s.expression,arity,0,active.versions,walk,out,work)){return 0;}
    }else{
        let header_id=ir::statement(b,s.second);
        if(s.second>=row.source_index || header_id==0 || !visit_statement(walk,header_id)){return 0;}
        let header=ir::statement_at(b,header_id-1).value;
        if(header.kind!=15 || header.first>=s.second || header.second>=s.second || !identity::branch(out,b"for",work) ||
            !loop_clause(b,p,header.first,parent,arity,true,ref mut active,walk,out,work) || !json::raw(out,b",",work) ||
            !(if(s.expression==0){identity::omitted(out,work)}else{condition(b,p,s.expression,arity,0,active.versions,walk,out,work)}) ||
            !json::raw(out,b",",work) || !loop_clause(b,p,header.second,parent,arity,false,ref mut active,walk,out,work)){return 0;}
    }
    if(walk.nodes>=64 || !json::raw(out,b",",work)){return 0;}
    walk.nodes+=1;
    walk.loop_depth+=1;
    let body=control_block(b,p,s.first,parent,false,arity,active.versions,walk,out,work);
    walk.loop_depth-=1;
    walk.depth=depth;
    return if(body!=0 && json::raw(out,b"]}",work)){1}else{0};
}
fn u32 control_item(ref ir::Body b,ref clean::Program p,usize item,usize parent,usize arity,ref mut Slots active,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let row=ir::statement_at(b,item-1);
    let s=row.value;
    if(walk.nodes>=64 || parent!=0 && s.expression>=parent){return 0;}
    walk.nodes+=1;
    if(s.kind==8 || s.kind==14){return control_loop(b,p,row,parent,arity,active.versions,walk,out,work);}
    if(s.kind==9 || s.kind==10){
        return if(s.expression==0 && s.first==0 && s.second==0 && walk.loop_depth!=0 &&
            identity::loop_control(out,s.kind==9,work)){1}else{0};
    }
    if(s.expression==0){return 0;}
    if(s.kind==6){return control_if(b,p,s.expression,arity,active.versions,walk,out,work);}
    if(s.kind==3){
        return if(identity::branch(out,b"return",work) && expression(b,p,s.expression,arity,0,active.versions,walk,out,work) &&
            json::raw(out,b"]}",work)){2}else{0};
    }
    if(s.kind==2){
        let binding=ir::local(b,s.entry);
        if(!declare(walk,binding,arity)){return 0;}
        let local=ir::local_at(b,binding);
        if(local.initializer!=s.expression ||
            !identity::branch(out,if(local.kind==3){b"var"}else{b"let"},work) ||
            !expression(b,p,s.expression,arity,0,active.versions,walk,out,work) || !json::raw(out,b"]}",work)){return 0;}
        active.versions[binding-1]=identity::arm_position(walk.depth)+1;
        walk.depth+=1;
        return 1;
    }
    if(s.kind!=4 || !bytes::equal(ir::assignment_operator(b,item-1),b"=")){return 0;}
    let target=ir::node(b,s.target);
    if(target==0){return 0;}
    let e=ir::node_at(b,target-1).value;
    if(e.kind!=2 || e.state!=1 || !scalar(p,e.type) || e.binding<=arity || e.binding>ir::local_count(b) ||
        ir::local_at(b,e.binding).kind!=3 || e.place.kind!=1 || e.place.root!=e.binding || !e.place.mutable){return 0;}
    let version=active.versions[e.binding-1];
    return if(version>identity::arm_position(0) && identity::assignment(out,version-1,work) &&
        expression(b,p,s.expression,arity,0,active.versions,walk,out,work) && json::raw(out,b"]}",work)){1}else{0};
}
fn u32 control_block(ref ir::Body b,ref clean::Program p,usize block,usize parent,bool root,usize arity,usize[20] versions,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work){
    let index=ir::statement(b,block);
    if(index==0 || !visit_statement(walk,index)){return 0;}
    let s=ir::statement_at(b,index-1).value;
    if(s.kind!=1 || s.first>=block || s.second>64-walk.nodes || (s.first==0)!=(s.second==0) ||
        parent!=0 && s.expression>=parent){return 0;}
    if(s.second==0 && s.expression==0){return if(identity::empty_block(out,work)){1}else{0};}
    if(!identity::branch(out,b"block",work)){return 0;}
    let depth=walk.depth;
    var Slots active=Slots{.versions};
    var u32 flow=1;
    var usize i=0;
    while(i<s.second){
        if(i!=0 && !json::raw(out,b",",work)){return 0;}
        let item=block_item(b,s,i,walk);
        if(item==0){return 0;}
        let current=control_item(b,p,item,parent,arity,ref mut active,walk,out,work);
        if(current==0){return 0;}
        if(current==2){flow=2;}
        i+=1;
    }
    if(s.expression!=0){
        if(i!=0 && !json::raw(out,b",",work)){return 0;}
        if(root){
            if(!identity::branch(out,b"return",work) || !expression(b,p,s.expression,arity,0,active.versions,walk,out,work) ||
                !json::raw(out,b"]}",work)){return 0;}
            flow=2;
        }else{
            let current=control_if(b,p,s.expression,arity,active.versions,walk,out,work);
            if(current==0){return 0;}
            if(current==2){flow=2;}
        }
    }
    walk.depth=depth;
    return if(json::raw(out,b"]}",work)){flow}else{0};
}
fn bool uses_control(ref ir::Body b,syntax::Statement root){
    var usize i=0;
    while(i<root.second){
        let link=block_link(b,root,i);
        if(link==0){return false;}
        let item=ir::statement(b,ir::statement_at(b,link-1).value.first);
        if(item==0){return false;}
        let s=ir::statement_at(b,item-1).value;
        if(s.kind==6){
            let index=ir::node(b,s.expression);
            if(index!=0 && ir::node_at(b,index-1).value.kind==13){return true;}
        }
        i+=1;
    }
    return false;
}

fn bool has_loop(ref ir::Body b){
    var usize i=0;
    while(i<ir::statement_count(b)){
        let kind=ir::statement_at(b,i).value.kind;
        if(kind==8 || kind==14){return true;}
        i+=1;
    }
    return false;
}

// An absent marker is ordinary IR. A present marker must describe this exact
// pure detached body, including its result and original lexical parameter order.
pub fn u32 check(ref ir::Body b,ref clean::Program p,usize frame){
    if(identity::zero(ir::inline_hash(b))){return 0;}let f=clean::frame_at(p,frame);if(f.kind!=3 || f.effects!=0 || !scalar(p,f.result) || ir::local_count(b)>20 || ir::call_count(b)>64 || ir::argument_count(b)>256 || ir::region_count(b)!=0 || ir::placement_count(b)!=0 || ir::reset_count(b)!=0){return 1001;}
    var usize arity=0;var usize i=1;var bool mutating=false;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(!scalar(p,l.type)){return 1001;}if(l.kind==1 && i==arity+1){if(l.origin.mutable){return 1001;}arity+=1;}else if(l.kind==2 || l.kind==3){mutating=mutating || l.kind==3;}else{return 1001;}i+=1;}if(arity>4){return 1001;}let locals=ir::local_count(b)-arity;if(locals>16){return 1001;}
    let root_index=ir::statement(b,ir::body(b));if(root_index==0){return 1001;}let root=ir::statement_at(b,root_index-1).value;if(root.kind!=1 || root.second<if(root.expression==0){1}else{0} || root.second-if(root.expression==0){1}else{0}>64){return 1001;}
    let min_statements=1+2*root.second;if(ir::statement_count(b)<min_statements){return 1001;}
    var bytes::Buffer out=bytes::empty();
    var json::Ledger work=json::empty();
    var Walk walk=Walk{.nodes=0,.calls=0,.block_count=0,.blocks=[0;64],.arm_statements=0,.depth=0,.declarations=0,.declared=0,.statement_mode=false,.loop_depth=0,.visited=[0;4],.visited_count=0};
    let loop=has_loop(b);
    if(loop || uses_control(b,root)){
        walk.statement_mode=true;
        if(!identity::statement_header(ref mut out,arity,loop,ref mut work) ||
            control_block(b,p,ir::body(b),0,true,arity,[0;20],ref mut walk,ref mut out,ref mut work)!=2 ||
            walk.declarations!=locals || walk.calls!=ir::call_count(b) || walk.visited_count!=ir::statement_count(b)){
            return 1001;
        }
        let marker=switch(identity::finish(ref mut out,ref mut work)){case .ok(h):{h}case .err(_):{return 1001;}};
        return if(identity::equal(marker,ir::inline_hash(b))){0}else{1001};
    }
    let conditional=ir::statement_count(b)>min_statements;
    let steps=root.second-if(root.expression==0){1}else{0};
    if(!if(conditional){identity::conditional_header(ref mut out,arity,steps!=0,mutating,ref mut work)}
        else if(mutating){identity::steps_header(ref mut out,arity,ref mut work)}
        else if(ir::call_count(b)!=0){identity::calls_header(ref mut out,arity,steps!=0,ref mut work)}
        else if(steps==0){identity::header(ref mut out,arity,ref mut work)}
        else{identity::locals_header(ref mut out,arity,ref mut work)}){return 1001;}
    var usize returned=root.expression;
    var usize[20] versions=[0;20];
    i=0;
    while(i<root.second){
        let item=block_item(b,root,i,ref mut walk);
        if(item==0){return 1001;}
        let s=ir::statement_at(b,item-1).value;
        if(i==steps){
            if(s.kind!=3){return 1001;}
            returned=s.expression;
            break;
        }
        var usize binding=0;
        if(s.kind==2){
            binding=ir::local(b,s.entry);
            if(!declare(ref mut walk,binding,arity) || ir::local_at(b,binding).initializer!=s.expression){return 1001;}
        }else if(s.kind==4){
            let target=ir::node(b,s.target);
            if(target==0 || !bytes::equal(ir::assignment_operator(b,item-1),b"=")){return 1001;}
            let e=ir::node_at(b,target-1).value;
            binding=e.binding;
            if(e.kind!=2 || e.state!=1 || !scalar(p,e.type) || binding<=arity || binding>ir::local_count(b) ||
                ir::local_at(b,binding).kind!=3 || versions[binding-1]==0){return 1001;}
        }else{return 1001;}
        if(!expression(b,p,s.expression,arity,i,versions,ref mut walk,ref mut out,ref mut work) ||
            !json::raw(ref mut out,b",",ref mut work)){return 1001;}
        versions[binding-1]=i+1;
        i+=1;
    }
    if(!expression(b,p,returned,arity,steps,versions,ref mut walk,ref mut out,ref mut work) ||
        walk.declarations!=locals || walk.calls!=ir::call_count(b) ||
        walk.block_count+walk.arm_statements!=ir::statement_count(b)-min_statements ||
        steps!=0 && !json::raw(ref mut out,b"]}",ref mut work)){return 1001;}
    let hash=switch(identity::finish(ref mut out,ref mut work)){
        case .ok(h):{h}
        case .err(_):{return 1001;}
    };
    return if(identity::equal(hash,ir::inline_hash(b))){0}else{1001};
}
