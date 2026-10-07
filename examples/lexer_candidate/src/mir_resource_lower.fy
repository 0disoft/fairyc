module mir_resource_lower;
import hir_model as h;
import hir_shapes as shapes;
import resource_identity as resources;
import mir_model as m;
import mir_state as state;
import std::vec as v;

pub fn bool any_live(ref state::Context c){var usize i=1;while(i<1153){if(c.live[i]!=0){return true;}i+=1;}return false;}
fn usize parent(ref h::Program p,usize scope){var usize i=1;while(i<=h::block_count(p)){let b=h::block_at(p,i);if(b.scope==scope){return b.parent_scope;}i+=1;}return 0;}
pub fn bool exited(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize scope,usize start,usize end){
    var usize at=c.scope;var usize depth=0;while(at!=0 && depth<64){
        if(!scope_roots(input,locals,blocks,operations,edges,c,at,start,end)){return false;}if(at==scope){return true;}at=parent(input,at);depth+=1;
    }discard state::fail(c,4,start,end);return false;
}
pub fn bool drop(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize local,usize start,usize end){
    state::release(c,local);let u32 type=v::at<m::Local>(locals,local-1).type;
    if(shapes::needs_drop(type) && !resources::valid(type)){return state::emit(operations,blocks,c,state::operation(26,0,type,type,0,local,0,start,end))!=0;}
    var usize i=0;
    while(i<h::drop_count(input) && h::drop_at(input,i).type!=type){i+=1;}
    if(i==h::drop_count(input)){discard state::fail(c,4,start,end);return false;}
    let d=h::drop_at(input,i);
    let shape=shapes::at(h::nominal(input),shapes::find(h::nominal(input),type)-1);var usize field=0;
    while(field<shape.field_count){if(shapes::needs_drop(shapes::field_at(h::nominal(input),shape.field_start+field+1).type)){
        return state::emit(operations,blocks,c,state::operation(26,0,type,type,0,local,0,start,end))!=0;
    }field+=1;}
    if(!d.nonempty){return state::emit(operations,blocks,c,state::operation(25,0,type,type,0,local,0,start,end))!=0;}
    let usize callee=c.targets[d.function];if(callee==0){discard state::fail(c,4,start,end);return false;}
    let usize arguments=v::len<usize>(edges);if(!state::edge(edges,local,c)){return false;}
    let usize next=state::new_block(blocks,c,start,end);var m::Block term=state::block(c.function,start,end);
    term.kind=5;term.type=type;term.callee=callee;term.argument_start=arguments;term.argument_count=1;term.target=next;
    if(!state::terminate(blocks,c,term)){return false;}c.current=next;return true;
}
// One scope drops newer partial temporaries before its older declared roots.
// Full exits walk inner scopes first, preserving lexical ordering around ?.
fn bool scope_roots(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize scope,usize start,usize end){
    var usize work=0;while(true){var usize selected=0;var usize binding=0;var usize latest=0;
        var usize temporary=v::len<m::Local>(locals);while(temporary>c.base){
            if(work>=67108864){discard state::fail(c,2,start,end);return false;}work+=1;
            if(state::pending(c,temporary)==1 && temporary!=c.keep && state::temporary_scope(c,temporary)==scope){let usize at=state::temporary_start(c,temporary);if(selected==0 || at>latest){selected=temporary;latest=at;}}
            temporary-=1;
        }
        var usize root=h::binding_count(input);while(root!=0){let b=h::binding_at(input,root);
            if(c.live[root]!=0 && b.scope==scope && (selected==0 || b.start>latest)){selected=c.locals[root];binding=root;latest=b.start;}root-=1;
        }
        if(selected==0){return true;}if(binding!=0){c.live[binding]=0;}if(!drop(input,locals,blocks,operations,edges,c,selected,start,end)){return false;}
    }return false;
}
pub fn bool scope(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize scope,usize start,usize end){
    if(scope!=0){return scope_roots(input,locals,blocks,operations,edges,c,scope,start,end);}
    var usize at=c.scope;var usize depth=0;while(at!=0 && depth<64){if(!scope_roots(input,locals,blocks,operations,edges,c,at,start,end)){return false;}at=parent(input,at);depth+=1;}
    if(at!=0){discard state::fail(c,2,start,end);return false;}return true;
}
