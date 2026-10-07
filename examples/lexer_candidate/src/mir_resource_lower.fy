module mir_resource_lower;
import hir_model as h;
import hir_shapes as shapes;
import mir_model as m;
import mir_state as state;
import std::vec as v;

pub fn bool any_live(ref state::Context c){var usize i=1;while(i<1153){if(c.live[i]!=0){return true;}i+=1;}return false;}
// Follow retained lexical parents; loop exits include nested body scopes but
// preserve roots owned by the loop's enclosing block and function.
fn bool within(ref h::Program input,usize scope,usize ancestor){
    var usize current=scope;var usize depth=0;
    while(current!=0 && depth<64){
        if(current==ancestor){return true;}
        var usize i=1;var usize parent=0;
        while(i<=h::block_count(input)){let b=h::block_at(input,i);if(b.scope==current){parent=b.parent_scope;break;}i+=1;}
        current=parent;depth+=1;
    }return false;
}
pub fn bool exited(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize scope,usize start,usize end){
    var usize binding=h::binding_count(input);while(binding!=0){let b=h::binding_at(input,binding);
        if(c.live[binding]!=0 && within(input,b.scope,scope)){
            let usize local=c.locals[binding];c.live[binding]=0;if(!drop(input,locals,blocks,operations,edges,c,local,start,end)){return false;}
        }binding-=1;
    }return true;
}
pub fn bool drop(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize local,usize start,usize end){
    state::release(c,local);let u32 type=v::at<m::Local>(locals,local-1).type;var usize i=0;
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
// Resolved binding IDs preserve lexical declaration order. A scope selects
// only roots it owns; an empty scope selector means every active function root.
pub fn bool scope(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize scope,usize start,usize end){
    if(scope==0){var usize temporary=v::len<m::Local>(locals);while(temporary>c.base){
        if(state::pending(c,temporary)==1 && temporary!=c.keep){if(!drop(input,locals,blocks,operations,edges,c,temporary,start,end)){return false;}}temporary-=1;
    }}
    var usize binding=h::binding_count(input);while(binding!=0){let b=h::binding_at(input,binding);
        if(c.live[binding]!=0 && (scope==0 || b.scope==scope)){
            let usize local=c.locals[binding];c.live[binding]=0;if(!drop(input,locals,blocks,operations,edges,c,local,start,end)){return false;}
        }binding-=1;
    }return true;
}
