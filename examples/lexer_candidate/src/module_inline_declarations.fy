module module_inline_declarations;
import syntax as syntax;
import type_state as scalars;
import module_inline_identity as identity;
import generic_source_json as json;
import std::bytes as bytes;
import std::vec as v;

pub struct Info{bool annotated;u32 code;usize start;usize end;usize expression;usize parameters;u32[8] hash;}
struct Row{usize declaration;usize edges;Info info;}
struct Walk { usize nodes; usize declarations; usize depth; usize[16] bindings; }

fn usize arm_binding(ref syntax::Parsed p,ref Walk walk,ref u8[] name){
    var usize i=walk.depth;
    while(i>0){
        i-=1;
        if(bytes::equal(name,local_name(p,syntax::statement_at(p,walk.bindings[i])))){return i+1;}
    }
    return 0;
}
fn Info invalid(Info info){var Info out=info;out.code=3;return out;}
fn bool i32_type(ref syntax::Parsed p,usize root){return root!=0 && syntax::type_at(p,root).kind==1 && scalars::primitive(p,root)==1;}
fn ref u8[] local_name(ref syntax::Parsed p,syntax::Statement s){if(s.kind==2 && s.entry!=0){let a=syntax::entry_at(p,s.entry-1);return syntax::source_view(p)[a.name_start..a.name_end];}if(s.kind==4 && s.target!=0 && syntax::expression_at(p,s.target).kind==2){return syntax::expression_text(p,s.target);}return b"";}
fn bool bound(ref syntax::Parsed p,syntax::Declaration d,ref u8[] name,usize visible){var usize i=0;while(i<visible){let s=syntax::statement_at(p,syntax::block_statement(p,d.body,i));if(bytes::equal(name,local_name(p,s))){return true;}i+=1;}i=d.entry_start;while(i<d.entry_start+d.entry_count){let a=syntax::entry_at(p,i);if(a.kind==1 && bytes::equal(name,syntax::source_view(p)[a.name_start..a.name_end])){return true;}i+=1;}return false;}
fn bool has_kind(ref syntax::Parsed p,syntax::Declaration d,u32 kind){let b=syntax::statement_at(p,d.body);var usize i=1;while(i<=syntax::expression_count(p)){let e=syntax::expression_at(p,i);if(e.kind==kind && e.start>=b.start && e.end<=b.end){return true;}i+=1;}return false;}
fn bool edge(ref syntax::Parsed p,syntax::Declaration d,usize target,ref mut v::Vec<Row> graph){var usize owner=64;var usize callee=64;var usize i=0;while(i<v::len<Row>(graph)){let row=*v::at<Row>(graph,i);if(syntax::declaration_at(p,row.declaration).name_start==d.name_start){owner=i;}if(row.declaration==target){callee=i;}i+=1;}if(owner==64 || callee==64){return false;}let usize one=1;let row=v::at_mut<Row>(graph,owner);row.edges=row.edges|(one<<callee);return true;}
fn bool condition(ref syntax::Parsed p,syntax::Declaration d,usize id,usize visible,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work,ref mut v::Vec<Row> graph){
    if(id==0 || id>syntax::expression_count(p) || walk.nodes>=64){return json::reject(work,9);}
    walk.nodes+=1;
    let e=syntax::expression_at(p,id);
    if(e.kind==3){return e.left<id && condition(p,d,e.left,visible,walk,out,work,graph);}
    let op=syntax::expression_text(p,id);
    if(e.kind==1){return identity::boolean(out,op,work);}
    if(e.kind==4 && bytes::equal(op,b"!")){
        return e.left<id && identity::branch(out,op,work) &&
            condition(p,d,e.left,visible,walk,out,work,graph) && json::raw(out,b"]}",work);
    }
    if(e.kind==5 && identity::logical(op)){
        return e.left<id && e.right<id && identity::branch(out,op,work) &&
            condition(p,d,e.left,visible,walk,out,work,graph) && json::raw(out,b",",work) &&
            condition(p,d,e.right,visible,walk,out,work,graph) && json::raw(out,b"]}",work);
    }
    return e.kind==5 && e.left<id && e.right<id && identity::comparison(op) &&
        identity::branch(out,op,work) && expression(p,d,e.left,visible,walk,out,work,graph) &&
        json::raw(out,b",",work) && expression(p,d,e.right,visible,walk,out,work,graph) && json::raw(out,b"]}",work);
}
fn bool value_block(ref syntax::Parsed p,syntax::Declaration d,usize block,usize parent,usize visible,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work,ref mut v::Vec<Row> graph){
    if(block==0 || block>syntax::statement_count(p)){return json::reject(work,9);}
    let row=syntax::statement_at(p,block);
    if(row.kind!=1 || row.expression==0 || row.expression>=parent){return json::reject(work,9);}
    if(row.second==0){
        return row.first==0 && expression(p,d,row.expression,visible,walk,out,work,graph);
    }
    if(row.second>16-walk.declarations || !identity::branch(out,b"arm-bindings",work)){return json::reject(work,9);}
    let depth=walk.depth;
    var usize i=0;
    while(i<row.second){
        let id=syntax::block_statement(p,block,i);
        let binding=syntax::statement_at(p,id);
        if(binding.kind!=2 || binding.entry==0 || binding.expression>=parent){return json::reject(work,9);}
        let entry=syntax::entry_at(p,binding.entry-1);
        if(entry.kind!=2 || entry.type_root!=0 && !i32_type(p,entry.type_root)){return json::reject(work,9);}
        if(walk.declarations>=16){return json::reject(work,9);}
        walk.declarations+=1;
        if(!expression(p,d,binding.expression,visible,walk,out,work,graph) || !json::raw(out,b",",work)){
            return json::reject(work,9);
        }
        let slot=walk.depth;
        walk.bindings[slot]=id;
        walk.depth+=1;
        i+=1;
    }
    let ok=expression(p,d,row.expression,visible,walk,out,work,graph) && json::raw(out,b"]}",work);
    walk.depth=depth;
    return ok;
}
fn bool expression(ref syntax::Parsed p,syntax::Declaration d,usize id,usize visible,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work,ref mut v::Vec<Row> graph){
    if(id==0 || id>syntax::expression_count(p) || walk.nodes>=64){return json::reject(work,9);}walk.nodes+=1;let e=syntax::expression_at(p,id);
    if(e.kind==3){return e.left<id && expression(p,d,e.left,visible,walk,out,work,graph);}
    if(e.kind==13){
        if(e.left>=id || e.right==0 || e.right>=id){return json::reject(work,9);}
        let arms=syntax::expression_at(p,e.right);
        if(arms.kind!=14 || arms.right==0 || arms.right>=e.right){return json::reject(work,9);}
        if(!identity::branch(out,b"if",work) || !condition(p,d,e.left,visible,walk,out,work,graph) ||
            walk.nodes>=64 || !json::raw(out,b",",work)){return json::reject(work,9);}
        walk.nodes+=1;
        if(!value_block(p,d,arms.left,e.right,visible,walk,out,work,graph) || !json::raw(out,b",",work)){return json::reject(work,9);}
        let other=syntax::expression_at(p,arms.right);
        if(other.kind==12){
            if(walk.nodes>=64){return json::reject(work,9);}
            walk.nodes+=1;
            if(!value_block(p,d,other.left,arms.right,visible,walk,out,work,graph)){return json::reject(work,9);}
        }else if(other.kind!=13 || !expression(p,d,arms.right,visible,walk,out,work,graph)){return json::reject(work,9);}
        return json::raw(out,b"]}",work);
    }
    if(e.kind==1){let n=switch(identity::integer(syntax::expression_text(p,id))){case .ok(n):{n}case .err(_):{return json::reject(work,9);}};return identity::leaf(out,b"literal",n,work);}
    if(e.kind==2){let name=syntax::expression_text(p,id);let arm=arm_binding(p,walk,name);if(arm!=0){return identity::leaf(out,b"local",identity::arm_position(arm-1),work);}var usize local=visible;while(local>0){local-=1;let st=syntax::statement_at(p,syntax::block_statement(p,d.body,local));if(bytes::equal(name,local_name(p,st))){return identity::leaf(out,b"local",local,work);}}var usize i=d.entry_start;var usize position=0;while(i<d.entry_start+d.entry_count){let a=syntax::entry_at(p,i);if(a.kind==1){if(bytes::equal(name,syntax::source_view(p)[a.name_start..a.name_end])){return identity::leaf(out,b"path",position,work);}position+=1;}i+=1;}return json::reject(work,9);}
    if(e.kind==6){
        if(e.left==0 || e.left>=id || syntax::expression_at(p,e.left).kind!=2){return json::reject(work,9);}let name=syntax::expression_text(p,e.left);if(arm_binding(p,walk,name)!=0 || bound(p,d,name,visible)){return json::reject(work,9);}
        var usize target=syntax::declaration_count(p);var usize i=0;while(i<syntax::declaration_count(p)){let candidate=syntax::declaration_at(p,i);if(candidate.kind==3 && candidate.owner_end==0 && bytes::equal(name,syntax::source_view(p)[candidate.name_start..candidate.name_end])){if(target!=syntax::declaration_count(p)){return json::reject(work,9);}target=i;}i+=1;}if(target==syntax::declaration_count(p)){return json::reject(work,9);}
        let callee=read_graph(p,target,graph);let arguments=syntax::call_argument_count(p,id);if(!callee.annotated || callee.code!=0 || arguments!=callee.parameters || !edge(p,d,target,graph) || !identity::call(out,name,arguments!=0,work)){return json::reject(work,9);}i=0;while(i<arguments){let argument=syntax::call_argument(p,id,i);if(syntax::call_argument_labeled(p,syntax::call_argument_entry(p,id,i)) || argument==0 || argument>=id || i!=0 && !json::raw(out,b",",work) || !expression(p,d,argument,visible,walk,out,work,graph)){return json::reject(work,9);}i+=1;}return arguments==0 || json::raw(out,b"]}",work);
    }
    if(e.kind!=4 && e.kind!=5){return json::reject(work,9);}let op=syntax::expression_text(p,id);
    if(e.kind==4 && bytes::equal(op,b"-") && e.left!=0 && e.left<id && syntax::expression_at(p,e.left).kind==1){let n=switch(identity::integer(syntax::expression_text(p,e.left))){case .ok(n):{n}case .err(_):{return json::reject(work,9);}};if(n==2147483648){return identity::minimum(out,work);}}
    if(!identity::operator(op,e.kind==5) || e.left>=id || !identity::branch(out,op,work) || !expression(p,d,e.left,visible,walk,out,work,graph)){return json::reject(work,9);}
    if(e.kind==5 && (e.right>=id || !json::raw(out,b",",work) || !expression(p,d,e.right,visible,walk,out,work,graph))){return json::reject(work,9);}return json::raw(out,b"]}",work);
}
// 0 is invalid, 1 falls through, 2 returns on every path.
fn u32 control_if(ref syntax::Parsed p,syntax::Declaration d,usize id,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work,ref mut v::Vec<Row> graph){
    if(id==0 || id>syntax::expression_count(p) || walk.nodes>=64){return 0;}
    let e=syntax::expression_at(p,id);
    if(e.kind!=13 || e.left>=id || e.right==0 || e.right>=id){return 0;}
    let arms=syntax::expression_at(p,e.right);
    if(arms.kind!=14 || arms.right>=e.right){return 0;}
    walk.nodes+=1;
    if(!identity::branch(out,b"if-statement",work) || !condition(p,d,e.left,0,walk,out,work,graph) ||
        !json::raw(out,b",",work) || walk.nodes>=64){return 0;}
    walk.nodes+=1;
    let left=control_block(p,d,arms.left,false,walk,out,work,graph);
    if(left==0){return 0;}
    var u32 right=1;
    if(arms.right!=0){
        if(!json::raw(out,b",",work)){return 0;}
        let other=syntax::expression_at(p,arms.right);
        if(other.kind==13){right=control_if(p,d,arms.right,walk,out,work,graph);}
        else if(other.kind==12 && walk.nodes<64){
            walk.nodes+=1;
            right=control_block(p,d,other.left,false,walk,out,work,graph);
        }else{return 0;}
    }
    return if(right==0 || !json::raw(out,b"]}",work)){0}else if(left==2 && right==2){2}else{1};
}
fn u32 control_item(ref syntax::Parsed p,syntax::Declaration d,usize id,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work,ref mut v::Vec<Row> graph){
    if(walk.nodes>=64){return 0;}
    walk.nodes+=1;
    let s=syntax::statement_at(p,id);
    if(s.kind==6){return control_if(p,d,s.expression,walk,out,work,graph);}
    if(s.kind==3){
        return if(identity::branch(out,b"return",work) && expression(p,d,s.expression,0,walk,out,work,graph) &&
            json::raw(out,b"]}",work)){2}else{0};
    }
    if(s.kind==2){
        if(s.entry==0 || walk.declarations>=16){return 0;}
        let entry=syntax::entry_at(p,s.entry-1);
        if((entry.kind!=2 && entry.kind!=3) || entry.type_root!=0 && !i32_type(p,entry.type_root)){return 0;}
        walk.declarations+=1;
        if(!identity::branch(out,if(entry.kind==3){b"var"}else{b"let"},work) ||
            !expression(p,d,s.expression,0,walk,out,work,graph) || !json::raw(out,b"]}",work)){return 0;}
        let slot=walk.depth;
        walk.bindings[slot]=id;
        walk.depth+=1;
        return 1;
    }
    if(s.kind!=4 || !bytes::equal(syntax::assignment_operator(p,id),b"=")){return 0;}
    let name=local_name(p,s);
    let slot=arm_binding(p,walk,name);
    if(name.len==0 || slot==0){return 0;}
    let binding=syntax::statement_at(p,walk.bindings[slot-1]);
    if(syntax::entry_at(p,binding.entry-1).kind!=3){return 0;}
    return if(identity::assignment(out,identity::arm_position(slot-1),work) &&
        expression(p,d,s.expression,0,walk,out,work,graph) && json::raw(out,b"]}",work)){1}else{0};
}
fn u32 control_block(ref syntax::Parsed p,syntax::Declaration d,usize block,bool root,ref mut Walk walk,ref mut bytes::Buffer out,ref mut json::Ledger work,ref mut v::Vec<Row> graph){
    if(block==0 || block>syntax::statement_count(p)){return 0;}
    let b=syntax::statement_at(p,block);
    if(b.kind!=1 || b.second>64-walk.nodes){return 0;}
    if(b.second==0 && b.expression==0){return if(identity::empty_block(out,work)){1}else{0};}
    if(!identity::branch(out,b"block",work)){return 0;}
    let depth=walk.depth;
    var u32 flow=1;
    var usize i=0;
    while(i<b.second){
        if(i!=0 && !json::raw(out,b",",work)){return 0;}
        let current=control_item(p,d,syntax::block_statement(p,block,i),walk,out,work,graph);
        if(current==0){return 0;}
        if(current==2){flow=2;}
        i+=1;
    }
    if(b.expression!=0){
        if(i!=0 && !json::raw(out,b",",work)){return 0;}
        if(root){
            if(!identity::branch(out,b"return",work) || !expression(p,d,b.expression,0,walk,out,work,graph) ||
                !json::raw(out,b"]}",work)){return 0;}
            flow=2;
        }else{
            let current=control_if(p,d,b.expression,walk,out,work,graph);
            if(current==0){return 0;}
            if(current==2){flow=2;}
        }
    }
    walk.depth=depth;
    return if(json::raw(out,b"]}",work)){flow}else{0};
}
fn Info read_control(ref syntax::Parsed p,syntax::Declaration d,Info info,ref mut v::Vec<Row> graph){
    var bytes::Buffer out=bytes::empty();
    var json::Ledger work=json::empty();
    var Walk walk=Walk{.nodes=0,.declarations=0,.depth=0,.bindings=[0;16]};
    if(!identity::statement_header(ref mut out,info.parameters,ref mut work) ||
        control_block(p,d,d.body,true,ref mut walk,ref mut out,ref mut work,graph)!=2){return invalid(info);}
    var Info result=info;
    result.hash=switch(identity::finish(ref mut out,ref mut work)){case .ok(h):{h}case .err(_):{return invalid(info);}};
    return result;
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
    let b=syntax::statement_at(p,d.body);if(b.kind!=1){return invalid(info);}let count=syntax::block_statement_count(p,d.body);
    if(count>65){return invalid(info);}
    while(i<count){
        let s=syntax::statement_at(p,syntax::block_statement(p,d.body,i));
        if(s.kind==6 && s.expression!=0 && syntax::expression_at(p,s.expression).kind==13){return read_control(p,d,info,graph);}
        i+=1;
    }
    var usize locals=count;if(b.expression!=0){info.expression=b.expression;}else{if(count==0){return invalid(info);}let last=syntax::statement_at(p,syntax::block_statement(p,d.body,count-1));if(last.kind!=3 || last.expression==0){return invalid(info);}info.expression=last.expression;locals-=1;}if(locals>64){return invalid(info);}var usize declarations=0;var bool mutating=false;
    i=0;while(i<locals){let id=syntax::block_statement(p,d.body,i);let s=syntax::statement_at(p,id);if(s.expression==0){return invalid(info);}if(s.kind==2){if(s.entry==0){return invalid(info);}let a=syntax::entry_at(p,s.entry-1);if((a.kind!=2 && a.kind!=3) || a.type_root!=0 && !i32_type(p,a.type_root)){return invalid(info);}declarations+=1;mutating=mutating || a.kind==3;}
        else if(s.kind==4){let name=local_name(p,s);if(name.len==0 || !bytes::equal(syntax::assignment_operator(p,id),b"=")){return invalid(info);}var bool writable=false;var usize j=0;while(j<i){let old=syntax::statement_at(p,syntax::block_statement(p,d.body,j));if(old.kind==2 && bytes::equal(name,local_name(p,old))){writable=syntax::entry_at(p,old.entry-1).kind==3;}j+=1;}if(!writable){return invalid(info);}mutating=true;}else{return invalid(info);}i+=1;}if(declarations>16){return invalid(info);}
    var bytes::Buffer out=bytes::empty();var json::Ledger work=json::empty();var Walk walk=Walk{.nodes=0,.declarations,.depth=0,.bindings=[0;16]};
    if(!if(has_kind(p,d,13)){identity::conditional_header(ref mut out,info.parameters,locals!=0,mutating,ref mut work)}else if(mutating){identity::steps_header(ref mut out,info.parameters,ref mut work)}else if(has_kind(p,d,6)){identity::calls_header(ref mut out,info.parameters,locals!=0,ref mut work)}else if(locals==0){identity::header(ref mut out,info.parameters,ref mut work)}else{identity::locals_header(ref mut out,info.parameters,ref mut work)}){return invalid(info);}
    i=0;while(i<locals){let s=syntax::statement_at(p,syntax::block_statement(p,d.body,i));if(!expression(p,d,s.expression,i,ref mut walk,ref mut out,ref mut work,graph) || !json::raw(ref mut out,b",",ref mut work)){return invalid(info);}i+=1;}
    if(!expression(p,d,info.expression,locals,ref mut walk,ref mut out,ref mut work,graph) || locals!=0 && !json::raw(ref mut out,b"]}",ref mut work)){return invalid(info);}info.hash=switch(identity::finish(ref mut out,ref mut work)){case .ok(h):{h}case .err(_):{return invalid(info);}};return info;
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
