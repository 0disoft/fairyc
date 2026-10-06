module mir_places;
import hir_model as h;
import mir_model as m;
import mir_state as state;
import mir_nominal as nominal;
import mir_arrays as array;
import array_types as arrays;
import type_state as ids;
import std::vec as v;

// Reverse field/index paths retain immutable, already checked index locals.
// Stores reread the current root after RHS effects and rebuild only this path.
pub struct Step { usize expression; usize index; }
pub fn usize root(ref h::Program p, usize id, bool mutable_only) {
    var usize current=id;var usize depth=0;
    while(current!=0){let e=h::expression_at(p,current);let u32 kind=h::expression_code(e.kind);
        if(kind==10 || kind==13){if(depth>=64 || kind==13 && !arrays::valid(h::type_code(h::expression_at(p,e.left).type))){return 0;}current=e.left;depth+=1;}
        else{return if(kind==3 && (!mutable_only || h::binding_code(h::binding_at(p,e.binding).kind)==4)){e.binding}else{0};}
    }return 0;
}
pub fn Result<v::Vec<Step>,u32> collect(ref h::Program p, usize target) {
    var v::Vec<Step> path=v::empty<Step>();var usize id=target;
    while(h::expression_code(h::expression_at(p,id).kind)==10 || h::expression_code(h::expression_at(p,id).kind)==13){
        if(v::len<Step>(ref path)>=64){return .err(2);}if(!v::push<Step>(ref mut path,Step{.expression=id,.index=0}).is_ok){return .err(3);}id=h::expression_at(p,id).left;
    }return .ok(path);
}
fn usize child(ref h::Program p, Step step, usize owner, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> ops, ref mut state::Context c, usize start, usize end) {
    let e=h::expression_at(p,step.expression);
    return if(h::expression_code(e.kind)==10){nominal::read(locals,blocks,ops,c,owner,h::type_code(e.type),e.binding,start,end)}else{array::read(locals,blocks,ops,c,h::type_code(h::expression_at(p,e.left).type),h::type_code(e.type),owner,step.index,start,end)};
}
pub fn usize read(ref h::Program p, ref v::Vec<Step> path, usize owner, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> ops, ref mut state::Context c, usize start, usize end) {
    var usize value=owner;var usize at=v::len<Step>(path);
    while(at!=0){at-=1;value=child(p,*v::at<Step>(path,at),value,locals,blocks,ops,c,start,end);if(value==0){return 0;}}return value;
}
fn usize update(ref h::Program p, ref v::Vec<Step> path, usize at, usize owner, usize value, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> ops, ref mut v::Vec<usize> edges, ref mut state::Context c, usize start, usize end) {
    let Step step=*v::at<Step>(path,at);let e=h::expression_at(p,step.expression);var usize replacement=value;
    if(at!=0){let usize next=child(p,step,owner,locals,blocks,ops,c,start,end);if(next==0){return 0;}replacement=update(p,path,at-1,next,value,locals,blocks,ops,edges,c,start,end);if(replacement==0){return 0;}}
    let u32 type=v::at<m::Local>(locals,owner-1).type;let usize result=state::local(locals,c,type,0,false,false,start,end);
    var m::Operation o=state::operation(8,result,type,type,ids::small_id(e.binding),owner,replacement,start,end);
    if(h::expression_code(e.kind)==13){o.kind=11;o.operator=0;o.right=step.index;o.argument_start=v::len<usize>(edges);o.argument_count=1;if(!state::edge(edges,replacement,c)){return 0;}}
    if(state::emit(ops,blocks,c,o)==0){return 0;}return result;
}
pub fn usize store(ref h::Program p, ref v::Vec<Step> path, usize owner, usize value, ref mut v::Vec<m::Local> locals, ref mut v::Vec<m::Block> blocks, ref mut v::Vec<m::Operation> ops, ref mut v::Vec<usize> edges, ref mut state::Context c, usize start, usize end) {
    return if(v::len<Step>(path)==0){value}else{update(p,path,v::len<Step>(path)-1,owner,value,locals,blocks,ops,edges,c,start,end)};
}
