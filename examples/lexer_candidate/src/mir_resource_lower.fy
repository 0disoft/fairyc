module mir_resource_lower;
import hir_model as h;
import mir_model as m;
import mir_state as state;
import std::vec as v;

pub fn bool any_live(ref state::Context c){var usize i=1;while(i<1153){if(c.live[i]!=0){return true;}i+=1;}return false;}
pub fn bool drop(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize local,usize start,usize end){
    let u32 type=v::at<m::Local>(locals,local-1).type;var usize i=0;
    while(i<h::drop_count(input) && h::drop_at(input,i).type!=type){i+=1;}
    if(i==h::drop_count(input)){discard state::fail(c,4,start,end);return false;}
    let d=h::drop_at(input,i);
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
    var usize binding=h::binding_count(input);while(binding!=0){let b=h::binding_at(input,binding);
        if(c.live[binding]!=0 && (scope==0 || b.scope==scope)){
            let usize local=c.locals[binding];c.live[binding]=0;if(!drop(input,locals,blocks,operations,edges,c,local,start,end)){return false;}
        }binding-=1;
    }return true;
}
