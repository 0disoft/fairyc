module mir_verify_views;
import mir_model as m;
import array_types as arrays;
import std::vec as v;

fn bool local(ref m::Program p, usize id, usize function, u32 type) { return id>0 && id<=m::local_count(p) && m::local_at(p,id).function==function && m::local_at(p,id).type==type; }
pub fn bool operation(ref m::Program p, m::Operation o) {
    if(o.type!=10 || !local(p,o.result,o.function,10) || o.operator!=0 || o.boolean || o.payload_start!=0 || o.payload_end!=0){return false;}
    if(o.kind==18){return arrays::valid(o.owner_type) && arrays::element(o.owner_type)==3 && local(p,o.left,o.function,o.owner_type) && m::local_at(p,o.left).binding!=0 && o.right==0 && o.argument_start==0 && o.argument_count==0;}
    return o.kind==19 && o.owner_type==10 && local(p,o.left,o.function,10) && local(p,o.right,o.function,5) && o.argument_count==1 && local(p,m::edge_at(p,o.argument_start),o.function,5);
}
fn usize get(ref v::Vec<usize> values, usize id){return *v::at<usize>(values,id);}
fn bool merge(ref mut v::Vec<usize> values, usize id, usize origin, ref mut bool changed, ref mut m::Problem problem, usize start, usize end) {
    if(origin==65538){return true;} let usize old=get(values,id);
    if(old!=65538 && old!=origin){*problem=m::Problem{.code=2103,.start,.end};return false;}
    if(old!=origin){let ref mut usize slot=v::at_mut<usize>(values,id);*slot=origin;*changed=true;}return true;
}
// Origin fixed point: 0 Static, 1..16384 exact function-local backing array,
// 131072+parameter-index caller origin, 65538 reserved unknown. A function
// may return only Static or one byte
// parameter. Actual call arguments substitute that summary before any return.
// This verifier recomputes provenance from MIR; it does not trust source spans
// or the earlier HIR approval to prove pointer lifetime.
pub fn m::Problem check(ref m::Program p) {
    var bool needed = m::has_views(p); var usize parameter = 1;
    while (!needed && parameter <= m::local_count(p)) {
        let l = m::local_at(p,parameter); needed = l.parameter && l.type == 10;
        parameter += 1;
    }
    if (!needed) { return m::Problem { .code=0,.start=0,.end=0 }; }
    var v::Vec<usize> values=v::empty<usize>();var v::Vec<usize> summaries=v::empty<usize>();
    var usize i=0;while(i<=m::local_count(p)){if(!v::push<usize>(ref mut values,65538).is_ok){return m::Problem{.code=3,.start=0,.end=0};}i+=1;}
    i=0;while(i<=m::target_count(p)){if(!v::push<usize>(ref mut summaries,65538).is_ok){return m::Problem{.code=3,.start=0,.end=0};}i+=1;}
    i=1;while(i<=m::function_count(p)){
        let f=m::function_at(p,i);var usize j=0;while(j<f.parameter_count){let usize id=m::edge_at(p,f.parameter_start+j);if(m::local_at(p,id).type==10){let ref mut usize slot=v::at_mut<usize>(ref mut values,id);*slot=131072+j;}j+=1;}i+=1;
    }
    var usize work=0;var bool changed=true;var m::Problem problem=m::Problem{.code=0,.start=0,.end=0};
    while(changed){
        changed=false;work+=m::operation_count(p)+m::block_count(p);if(work>1048576){return m::Problem{.code=2,.start=0,.end=0};}
        i=1;while(i<=m::operation_count(p)){
            let o=m::operation_at(p,i);if(o.type==10){
                let usize origin=if(o.kind==14){0}else if(o.kind==18){o.left}else if(o.kind==2 || o.kind==19){get(ref values,o.left)}else{65538};
                if(!merge(ref mut values,o.result,origin,ref mut changed,ref mut problem,o.start,o.end)){return problem;}
            }i+=1;
        }
        i=1;while(i<=m::block_count(p)){
            let b=m::block_at(p,i);
            if(b.kind==5 && b.destination!=0 && m::local_at(p,b.destination).type==10){
                let usize summary=get(ref summaries,b.callee);var usize origin=summary;
                if(summary>=131072){let usize index=summary-131072;if(index>=b.argument_count){return m::Problem{.code=1001,.start=b.start,.end=b.end};}origin=get(ref values,m::edge_at(p,b.argument_start+index));}
                if(!merge(ref mut values,b.destination,origin,ref mut changed,ref mut problem,b.start,b.end)){return problem;}
            }
            if(b.kind==1 && b.value!=0 && m::local_at(p,b.value).type==10){
                let usize origin=get(ref values,b.value);
                if(origin>0 && origin<65538){return m::Problem{.code=2102,.start=b.start,.end=b.end};}
                if(!merge(ref mut summaries,m::function_at(p,b.function).target,origin,ref mut changed,ref mut problem,b.start,b.end)){return problem;}
            }i+=1;
        }
    }
    // Early returns may leave unused declaration slots without definitions.
    // Only consumed descriptors need an origin, just as definite initialization
    // checks only actual reads after CFG reachability has been established.
    i=1;while(i<=m::operation_count(p)){
        let o=m::operation_at(p,i);if(o.left!=0 && m::local_at(p,o.left).type==10 && get(ref values,o.left)==65538){return m::Problem{.code=2103,.start=o.start,.end=o.end};}i+=1;
    }
    i=1;while(i<=m::block_count(p)){
        let b=m::block_at(p,i);if(b.value!=0 && m::local_at(p,b.value).type==10 && get(ref values,b.value)==65538){return m::Problem{.code=2103,.start=b.start,.end=b.end};}
        if(b.kind==5){var usize j=0;while(j<b.argument_count){let usize id=m::edge_at(p,b.argument_start+j);if(m::local_at(p,id).type==10 && get(ref values,id)==65538){return m::Problem{.code=2103,.start=b.start,.end=b.end};}j+=1;}}i+=1;
    }
    return problem;
}
