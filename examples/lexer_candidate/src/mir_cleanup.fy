module mir_cleanup;
import hir_model as h;
import mir_resource_lower as lifetime;
import mir_model as m;
import mir_state as state;
import std::vec as v;

// Emit only the selected exiting suffix. Registration metadata stays intact
// while sibling branches are compiled; their lexical block restores its bases.
pub fn bool emit_deferred(ref mut v::Vec<m::Block> blocks, ref mut v::Vec<usize> edges, ref mut state::Context c, usize from) {
    if (c.problem.code != 0) { return false; }
    if (from > c.defer_count || c.current == 0) { discard state::fail(c,4,0,0); return false; }
    var usize i = c.defer_count;
    while (i > from) {
        i -= 1;
        let usize start = c.deferred_starts[i]; let usize end = c.deferred_ends[i];
        let usize count = c.deferred_arg_counts[i]; let usize captured = c.deferred_arg_starts[i];
        let usize callee = c.deferred_targets[i];
        if (count > 65536 - v::len<usize>(edges)) { discard state::fail(c,2,start,end); return false; }
        let usize arguments = v::len<usize>(edges); var usize j = 0;
        while (j < count) {
            let usize value = c.captures[captured + j];
            if (!state::edge(edges,value,c)) { return false; } j += 1;
        }
        let usize next = state::new_block(blocks,c,start,end);
        var m::Block call = state::block(c.function,start,end);
        call.kind = 5; call.unsafe_call=c.deferred_unsafe[i]; call.callee = callee; call.argument_start = arguments;
        call.argument_count = count; call.target = next;
        if (!state::terminate(blocks,c,call)) { return false; }
        c.current = next;
    }
    return true;
}

// One exit policy: evaluate/freeze the result at the caller, invoke the exiting
// lexical defer suffix, then discharge resources. Traps have no cleanup edge.
pub fn bool exit(ref h::Program input,ref mut v::Vec<m::Local> locals,ref mut v::Vec<m::Block> blocks,ref mut v::Vec<m::Operation> operations,ref mut v::Vec<usize> edges,ref mut state::Context c,usize from,usize scope,bool crossed,usize start,usize end){
    if(!emit_deferred(blocks,edges,c,from)){return false;}
    return if(crossed){lifetime::exited(input,locals,blocks,operations,edges,c,scope,start,end)}else{lifetime::scope(input,locals,blocks,operations,edges,c,scope,start,end)};
}
