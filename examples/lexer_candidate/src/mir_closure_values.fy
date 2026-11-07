module mir_closure_values;
import mir_model as m;
import hir_shapes as shapes;
import nominal_identity as nominal;
import pointer_types as pointers;
import reference_types as refs;
import std::bytes as bytes;
import std::vec as v;

fn m::Problem fail(u32 code,usize start,usize end){return m::Problem{.code,.start,.end};}
fn bool local(ref m::Program p,usize id,usize function,u32 type){return id>0 && id<=m::local_count(p) && m::local_at(p,id).function==function && m::local_at(p,id).type==type;}
// The only direct closure identity is 13. Composite layouts are inspected
// separately; a pointer/reference must not hide a closure leaf from old gates.
fn bool leaf(u32 type){if(type==13){return true;}if(pointers::valid(type)){return pointers::inner(type)==13;}if(refs::valid(type)){return refs::inner(type)==13;}return false;}
pub fn bool present(ref m::Program p){
    if(m::closure_count(p)!=0){return true;}var usize i=1;
    while(i<=m::local_count(p)){if(leaf(m::local_at(p,i).type)){return true;}i+=1;}
    i=1;while(i<=m::target_count(p)){if(leaf(m::target_at(p,i).result)){return true;}i+=1;}
    i=0;while(i<m::signature_count(p)){if(leaf(m::signature_at(p,i))){return true;}i+=1;}
    i=1;while(i<=shapes::field_count(m::nominal(p))){if(leaf(shapes::field_at(m::nominal(p),i).type)){return true;}i+=1;}
    i=1;while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.kind==40 || leaf(o.type) || leaf(o.owner_type)){return true;}i+=1;}
    i=1;while(i<=m::block_count(p)){if(m::block_at(p,i).kind==8){return true;}i+=1;}return false;
}
pub fn usize find(ref m::Program p,usize target){var usize i=0;while(i<m::closure_count(p)){if(m::closure_at(p,i).target==target){return i+1;}i+=1;}return 0;}
// Arena bounds have already been checked. Admission is explicit, never inferred
// from a native pointer or an arbitrary function with a matching signature.
pub fn bool assembly(ref m::Program p,bool family){
    if(!family){return !present(p);}
    // Signature-only consumers have no literal target to register. Every
    // actual construction still needs a matching row before publication.
    var usize op=1;while(op<=m::operation_count(p)){let o=m::operation_at(p,op);if(o.kind==40){let usize slot=find(p,o.operator as usize);if(slot==0 || m::closure_at(p,slot-1).captures!=o.argument_count){return false;}}op+=1;}
    var usize i=0;while(i<m::closure_count(p)){let c=m::closure_at(p,i);if(c.target==0 || c.target>m::target_count(p) || c.captures>4){return false;}
        var usize j=0;while(j<i){if(m::closure_at(p,j).target==c.target){return false;}j+=1;}
        var bool used=false;j=1;while(j<=m::operation_count(p)){let o=m::operation_at(p,j);if(o.kind==40 && o.operator as usize==c.target){if(o.argument_count!=c.captures){return false;}used=true;}j+=1;}
        if(!used){return false;}i+=1;
    }return true;
}
pub fn bool operation(ref m::Program p,m::Operation o){
    let usize slot=find(p,o.operator as usize);
    if(o.kind!=40 || slot==0 || o.type!=13 || o.owner_type!=13 || !local(p,o.result,o.function,13) || o.left!=0 || o.right!=0 || o.payload_start!=0 || o.payload_end!=0 || o.boolean || o.argument_count!=m::closure_at(p,slot-1).captures || o.argument_count==0 && o.argument_start!=0){return false;}
    var usize i=0;while(i<o.argument_count){if(!local(p,m::edge_at(p,o.argument_start+i),o.function,1)){return false;}i+=1;}return true;
}
pub fn bool invocation(ref m::Program p,m::Block b){
    return b.kind==8 && !b.unsafe_call && b.type==1 && local(p,b.value,b.function,13) && local(p,b.left,b.function,1) && local(p,b.destination,b.function,1)
        && b.right==0 && b.operator==0 && b.alternative==0 && b.check==0 && b.callee==0 && b.argument_start==0 && b.argument_count==0 && b.trap_mask==0;
}
struct Budget {usize work;bool exhausted;}
fn bool charge(ref mut Budget budget,usize count){if(budget.work>67108864 || count>67108864-budget.work){budget.exhausted=true;return false;}budget.work+=count;return true;}
fn bool contains(ref m::Program p,ref bytes::Buffer bearing,u32 type,ref mut Budget budget){
    if(!charge(budget,shapes::count(m::nominal(p))+1)){return false;}
    if(leaf(type)){return true;}let u32 inner=if(pointers::valid(type)){pointers::inner(type)}else if(refs::valid(type)){refs::inner(type)}else{type};
    let usize slot=shapes::find(m::nominal(p),inner);return slot!=0 && bytes::view(bearing)[slot-1]!=0;
}
fn bool allowed(ref m::Program p,ref bytes::Buffer bearing,u32 type,ref mut Budget budget){return !contains(p,bearing,type,budget) || type==13 || nominal::valid(type);}
fn m::Problem transport(ref m::Program p){
    var Budget budget=Budget{.work=0,.exhausted=false};
    var bytes::Buffer bearing=bytes::empty();let usize count=shapes::count(m::nominal(p));if(!bytes::resize_zeroed(ref mut bearing,count).is_ok){return fail(3,0,0);}
    var bool changed=true;while(changed){if(budget.exhausted){return fail(2,0,0);}changed=false;var usize i=0;while(i<count){let s=shapes::at(m::nominal(p),i);if(bytes::view(ref bearing)[i]==0){var usize j=0;while(j<s.field_count){if(contains(p,ref bearing,shapes::field_at(m::nominal(p),s.field_start+j+1).type,ref mut budget)){let flags=bytes::view_mut(ref mut bearing);flags[i]=1;changed=true;break;}j+=1;}}i+=1;}}
    var usize i=0;while(i<count){let s=shapes::at(m::nominal(p),i);if(bytes::view(ref bearing)[i]!=0){if(!nominal::valid(s.type) || shapes::abi_find(m::nominal(p),s.type)!=0){return fail(1001,0,0);}}i+=1;}
    i=1;while(i<=shapes::field_count(m::nominal(p))){let f=shapes::field_at(m::nominal(p),i);if(contains(p,ref bearing,f.type,ref mut budget) && f.type!=13){return fail(1001,f.start,f.end);}i+=1;}
    var bytes::Buffer written=bytes::empty();if(!bytes::resize_zeroed(ref mut written,m::local_count(p)+1).is_ok){return fail(3,0,0);}
    i=1;while(i<=m::local_count(p)){let l=m::local_at(p,i);if(!allowed(p,ref bearing,l.type,ref mut budget) || contains(p,ref bearing,l.type,ref mut budget) && l.mutable){return fail(1001,l.start,l.end);}if(contains(p,ref bearing,l.type,ref mut budget) && l.parameter){let flags=bytes::view_mut(ref mut written);flags[i]=1;}i+=1;}
    i=1;while(i<=m::target_count(p)){let t=m::target_at(p,i);if(!allowed(p,ref bearing,t.result,ref mut budget) || t.foreign && contains(p,ref bearing,t.result,ref mut budget)){return fail(1001,t.start,t.end);}var usize j=0;while(j<t.signature_count){let u32 type=m::signature_at(p,t.signature_start+j);if(!allowed(p,ref bearing,type,ref mut budget) || t.foreign && contains(p,ref bearing,type,ref mut budget)){return fail(1001,t.start,t.end);}j+=1;}i+=1;}
    // Pointer/borrow/cast/storage operations cannot create an alternate route
    // to the private representation, including closure-bearing nominal values.
    i=1;while(i<=m::operation_count(p)){let o=m::operation_at(p,i);if(o.kind!=2 && o.kind!=6 && o.kind!=7 && o.kind!=40 && (contains(p,ref bearing,o.type,ref mut budget) || contains(p,ref bearing,o.owner_type,ref mut budget))){return fail(1001,o.start,o.end);}
        if(o.result!=0 && contains(p,ref bearing,m::local_at(p,o.result).type,ref mut budget)){if(bytes::view(ref written)[o.result]!=0){return fail(1001,o.start,o.end);}let flags=bytes::view_mut(ref mut written);flags[o.result]=1;}i+=1;}
    i=1;while(i<=m::block_count(p)){let b=m::block_at(p,i);if(b.kind==5 && b.destination!=0 && contains(p,ref bearing,m::local_at(p,b.destination).type,ref mut budget)){if(bytes::view(ref written)[b.destination]!=0){return fail(1001,b.start,b.end);}let flags=bytes::view_mut(ref mut written);flags[b.destination]=1;}i+=1;}
    return fail(if(budget.exhausted){2}else{0},0,0);
}
fn bool arithmetic(m::Operation o){
    if(o.type!=1 || o.owner_type!=1){return false;}
    if(o.kind==1 || o.kind==2){return true;}
    if(o.kind==3 || o.kind==5 && o.right==0){return o.operator==1 || o.operator==2;}
    return (o.kind==4 || o.kind==5) && (o.operator==1 || o.operator==2 || o.operator==5 || o.operator==6 || o.operator==7);
}
fn m::Problem body(ref m::Program p,m::ClosureTarget c,usize fid){
    let f=m::function_at(p,fid);let t=m::target_at(p,c.target);
    if(f.public || t.foreign || t.external || t.unsafe_call || t.effects!=0 || t.result!=1 || t.signature_count!=c.captures+1 || m::drop_target(p,c.target)){return fail(1001,f.start,f.end);}
    var usize i=0;while(i<t.signature_count){if(m::signature_at(p,t.signature_start+i)!=1){return fail(1001,t.start,t.end);}i+=1;}
    var bytes::Buffer writes=bytes::empty();var v::Vec<usize> incoming=v::empty<usize>();var v::Vec<usize> ready=v::empty<usize>();
    if(!bytes::resize_zeroed(ref mut writes,f.local_count).is_ok){return fail(3,f.start,f.end);}
    i=0;while(i<f.block_count){if(!v::push<usize>(ref mut incoming,0).is_ok){return fail(3,f.start,f.end);}i+=1;}
    i=f.local_start+1;while(i<=f.local_start+f.local_count){let l=m::local_at(p,i);if(l.type!=1 || l.mutable){return fail(1001,l.start,l.end);}if(l.parameter){let flags=bytes::view_mut(ref mut writes);flags[i-f.local_start-1]=1;}i+=1;}
    // Inspect dead blocks too. A caller-supplied marker is not a body proof.
    i=f.block_start+1;while(i<=f.block_start+f.block_count){let b=m::block_at(p,i);if(b.kind!=1 && b.kind!=2 && b.kind!=4 && b.kind!=6){return fail(1001,b.start,b.end);}var usize op=b.first;
        while(op!=0){let o=m::operation_at(p,op);if(!arithmetic(o)){return fail(1001,o.start,o.end);}if(o.result!=0){let usize at=o.result-f.local_start-1;if(bytes::view(ref writes)[at]!=0){return fail(1001,o.start,o.end);}let flags=bytes::view_mut(ref mut writes);flags[at]=1;}op=o.next;}
        var usize edge=0;while(edge<2){let usize to=if(edge==0){b.target}else{b.alternative};if(to!=0){let n=v::at_mut<usize>(ref mut incoming,to-f.block_start-1);*n+=1;}edge+=1;}i+=1;
    }
    i=0;while(i<f.block_count){if(*v::at<usize>(ref incoming,i)==0 && !v::push<usize>(ref mut ready,f.block_start+i+1).is_ok){return fail(3,f.start,f.end);}i+=1;}
    var usize visited=0;while(visited<v::len<usize>(ref ready)){let b=m::block_at(p,*v::at<usize>(ref ready,visited));var usize edge=0;while(edge<2){let usize to=if(edge==0){b.target}else{b.alternative};if(to!=0){var bool empty=false;{let n=v::at_mut<usize>(ref mut incoming,to-f.block_start-1);*n-=1;empty=*n==0;}if(empty && !v::push<usize>(ref mut ready,to).is_ok){return fail(3,f.start,f.end);}}edge+=1;}visited+=1;}
    return fail(if(visited==f.block_count){0}else{1001},f.start,f.end);
}
pub fn m::Problem check(ref m::Program p){
    if(!assembly(p,true)){return fail(1001,0,0);}if(!present(p)){return fail(0,0,0);}
    let checked=transport(p);if(checked.code!=0){return checked;}
    var usize i=0;while(i<m::closure_count(p)){let c=m::closure_at(p,i);var usize fid=1;while(fid<=m::function_count(p) && m::function_at(p,fid).target!=c.target){fid+=1;}if(fid>m::function_count(p)){return fail(1001,0,0);}let result=body(p,c,fid);if(result.code!=0){return result;}i+=1;}
    return fail(0,0,0);
}
