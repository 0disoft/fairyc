module hir_view_check;
import borrow_origins as owners;
import hir_borrow_cache as cached;
import hir_borrow_paths as paths;
import borrow_shapes as borrows;
import tagged_types as tagged;
import hir_model as h;
import hir_mutable_check as exclusive;
import std::vec as v;

// Zero means Static; leaves name one owner and canonical sets retain all
// possible local owners across selection. Return summaries still require one
// exact input parameter; recursive/ambiguous returns remain closed.
struct Context {usize work;usize origin_depth;bool cache_origins;h::Problem problem;}
// Four initialized binding lanes live in one owned Vec: initializer, last use,
// return summary and visit color. Summary retains Unknown (65538). No large
// array values enter generated-C expression temporaries.
fn Result<v::Vec<usize>,u32> table(){var v::Vec<usize> out=v::empty<usize>();if(!v::reserve<usize>(ref mut out,4612).is_ok){return .err(3);}var usize i=0;while(i<4612){if(!v::push<usize>(ref mut out,if(i>=2306 && i<3459){65538}else{0}).is_ok){return .err(3);}i+=1;}return .ok(out);}
fn usize read(ref v::Vec<usize> slots,usize lane,usize id){assert(lane<4 && id<1153);return *v::at<usize>(slots,lane*1153+id);}
fn void write(ref mut v::Vec<usize> slots,usize lane,usize id,usize value){assert(lane<4 && id<1153);let cell=v::at_mut<usize>(slots,lane*1153+id);*cell=value;}
fn void reset_colors(ref mut v::Vec<usize> slots){var usize i=0;while(i<1153){write(slots,3,i,0);i+=1;}}
fn bool byte_view(u32 type) { return borrows::count(type)!=0; }
fn usize fail(ref mut Context c, u32 code, usize start, usize end) {
    if (c.problem.code == 0) { c.problem = h::Problem { .code,.start,.end }; } return 0;
}
fn bool within(ref h::Program p, usize child, usize parent) {
    var usize scope = child;
    var usize depth = 0;
    while (scope != 0 && depth < 128) {
        if (scope == parent) { return true; }
        var usize i = 1; var usize next = 0;
        while (i <= h::block_count(p)) { let b = h::block_at(p,i); if (b.scope == scope) { next = b.parent_scope; break; } i += 1; }
        scope = next; depth += 1;
    } return false;
}
// Unknown is distinct from Static; cycles never bootstrap a guessed origin.
fn usize joined(ref mut Context c,ref mut v::Vec<owners::Entry> sets,ref mut v::Vec<usize> members,usize a,usize b,usize start,usize end){return switch(owners::join(sets,members,a,b,ref mut c.work)){case .ok(id):{id}case .err(code):{fail(c,code,start,end)}};}
fn bool scope_check(ref h::Program p,ref v::Vec<owners::Entry> sets,ref v::Vec<usize> members,usize origin,usize scope,bool escape){
    var usize i=0;let usize count=owners::count(sets,origin);
    while(i<count){let b=h::binding_at(p,owners::at(sets,members,origin,i));
        if(escape && within(p,b.scope,scope)){return true;}if(!escape && !within(p,scope,b.scope)){return false;}i+=1;
    }return !escape;
}
fn usize origin(ref h::Program p,ref mut Context c,ref mut v::Vec<usize> tables,ref mut v::Vec<owners::Entry> sets,ref mut v::Vec<usize> members,ref mut v::Vec<cached::Slot> cache,usize id,usize depth){
    if(id==0 || c.problem.code!=0){return 0;}
    if(!c.cache_origins){return trace_origin(p,c,tables,sets,members,cache,id,depth);}
    let e=h::expression_at(p,id);let u32 kind=h::expression_code(e.kind);
    if(!byte_view(h::type_code(e.type)) || kind==15 || kind==21){return 0;}
    let old=*v::at<cached::Slot>(cache,id);
    if(old.origin_state==2){
        c.work+=1;if(c.work>1048576){return fail(c,2,e.start,e.end);}
        if(old.expression_reach==1){let usize required=depth+old.origin_depth;if(required>=64){return fail(c,2,e.start,e.end);}if(required>c.origin_depth){c.origin_depth=required;}}
        return old.origin;
    }
    if(old.origin_state==1){let e=h::expression_at(p,id);return fail(c,2103,e.start,e.end);}
    {let ref mut cached::Slot stored=v::at_mut<cached::Slot>(cache,id);stored.origin_state=1;}
    let usize previous_depth=c.origin_depth;c.origin_depth=depth;
    let usize result=trace_origin(p,c,tables,sets,members,cache,id,depth);let usize required=c.origin_depth;
    {let ref mut cached::Slot stored=v::at_mut<cached::Slot>(cache,id);stored.origin=result;stored.origin_depth=required-depth;stored.origin_state=2;}
    if(previous_depth>c.origin_depth){c.origin_depth=previous_depth;}
    return result;
}
fn usize trace_origin(ref h::Program p, ref mut Context c,ref mut v::Vec<usize> tables, ref mut v::Vec<owners::Entry> sets, ref mut v::Vec<usize> members, ref mut v::Vec<cached::Slot> cache, usize id, usize depth) {
    if (id == 0 || c.problem.code != 0) { return 0; }
    let e = h::expression_at(p,id); let u32 kind = h::expression_code(e.kind);
    if (!byte_view(h::type_code(e.type)) || kind == 15 || kind == 21) { return 0; }
    if(!cached::reachable(cache,id,false,ref mut c.work)){return 0;}
    c.work += 1; if (depth >= 64 || c.work > 1048576) { return fail(c,2,e.start,e.end); }
    if(depth>c.origin_depth){c.origin_depth=depth;}
    if(kind==18){return if(e.edge_count==1){origin(p,c,tables,sets,members,cache,h::edge_at(p,e.edge_start),depth+1)}else{0};}
    if(kind==22 || kind==11){return origin(p,c,tables,sets,members,cache,e.left,depth+1);}
    if (kind == 16) { let usize owner=root(p,e.left); if(owner!=0 && byte_view(h::type_code(h::binding_at(p,owner).type))){return origin(p,c,tables,sets,members,cache,base(p,e.left),depth+1);}return owner; }
    if (kind == 17 || kind==25 || kind==29 || kind==30) { return origin(p,c,tables,sets,members,cache,e.left,depth + 1); }
    if (kind == 3) {
        let b = h::binding_at(p,e.binding);
        if (h::binding_code(b.kind) == 2) { return e.binding; }
        let usize initial = read(tables,0,e.binding); return origin(p,c,tables,sets,members,cache,initial,depth + 1);
    }
    if ((kind == 7 || kind==20)) { return origin(p,c,tables,sets,members,cache,h::block_at(p,e.block).tail,depth + 1); }
    if(kind==19){var usize result=0;var bool seen=false;var usize i=0;while(i<e.edge_count){let arm=h::expression_at(p,h::edge_at(p,e.edge_start+i));let usize tail=h::block_at(p,arm.block).tail;if(tail!=0){let usize value=origin(p,c,tables,sets,members,cache,tail,depth+1);result=if(seen){joined(c,sets,members,result,value,e.start,e.end)}else{value};seen=true;}i+=1;}return result;}
    if (kind == 8) {
        let usize yes = origin(p,c,tables,sets,members,cache,h::block_at(p,e.block).tail,depth + 1);
        let usize no = origin(p,c,tables,sets,members,cache,e.alternative,depth + 1);
        if(h::block_at(p,e.block).tail==0){return no;}if(!byte_view(h::type_code(h::expression_at(p,e.alternative).type))){return yes;}return joined(c,sets,members,yes,no,e.start,e.end);
    }
    if (kind == 6) {
        let usize returned = summary(p,c,tables,sets,members,cache,e.binding,depth + 1); if (returned == 0 || c.problem.code != 0) { return 0; }if(returned==65538){return 65538;}
        let usize parameter=origin_parameter(p,e.binding,returned);return if(parameter!=0){origin(p,c,tables,sets,members,cache,h::edge_at(p,e.edge_start+parameter-1),depth+1)}else{fail(c,2103,e.start,e.end)};
    } return fail(c,2103,e.start,e.end);
}
fn usize summary(ref h::Program p, ref mut Context c,ref mut v::Vec<usize> tables, ref mut v::Vec<owners::Entry> sets, ref mut v::Vec<usize> members, ref mut v::Vec<cached::Slot> cache, usize binding, usize depth) {
    if (read(tables,3,binding) == 2) { return read(tables,2,binding); }
    let b = h::binding_at(p,binding);
    if(read(tables,3,binding)==1){return read(tables,2,binding);}
    if (depth >= 64 || h::binding_code(b.kind) != 1) { return fail(c,2103,b.start,b.end); }
    write(tables,3,binding,1); var usize fid = 0;
    while (fid < h::function_count(p) && h::function_at(p,fid).binding != binding) { fid += 1; }
    if (fid == h::function_count(p)) { return fail(c,2103,b.start,b.end); }
    let f = h::function_at(p,fid); var usize result = 65538; var bool seen = false; var usize i = 1;
    while (i <= h::statement_count(p)) {
        let s = h::statement_at(p,i);
        // Specializations retain the same source bytes, but own distinct scopes.
        if (h::statement_code(s.kind) == 2 && within(p,s.scope,f.scope) && s.expression != 0 && cached::reachable(cache,i,true,ref mut c.work)) {
            let usize value = origin(p,c,tables,sets,members,cache,s.expression,depth + 1);
            result=joined(c,sets,members,result,value,s.start,s.end);seen=seen || value!=65538;
        } i += 1;
    }
    i=1;while(i<=h::expression_count(p)){let e=h::expression_at(p,i);if(h::expression_code(e.kind)==22 && e.binding==binding && tagged::result(h::type_code(h::expression_at(p,e.left).type)) && borrows::count(tagged::failure(h::type_code(h::expression_at(p,e.left).type)))!=0){let usize value=origin(p,c,tables,sets,members,cache,e.left,depth+1);result=joined(c,sets,members,result,value,e.start,e.end);}i+=1;}
    let body = h::block_at(p,f.body);
    if (body.tail != 0) { let usize value = origin(p,c,tables,sets,members,cache,body.tail,depth + 1); result=joined(c,sets,members,result,value,body.start,body.end); }
    if (result != 0 && result!=65538) {
        if(owners::mixed(result)){return fail(c,2103,f.start,f.end);}
        let owner = h::binding_at(p,result);
        if (h::binding_code(owner.kind) != 2 || !byte_view(h::type_code(owner.type)) || owner.owner != binding) { return fail(c,2102,f.start,f.end); }
    }
    write(tables,2,binding,result); write(tables,3,binding,2); return result;
}
fn usize loop_end(ref h::Program p, h::Expression e,ref mut Context c) {
    var usize end = e.end;
    if (h::expression_code(e.kind) == 3) {
        let binding = h::binding_at(p,e.binding); var usize i = 1;
        while (i <= h::statement_count(p)) {
            let s = h::statement_at(p,i); let u32 kind = h::statement_code(s.kind);
            if ((kind == 5 || kind == 6) && e.start >= s.start && e.end <= s.end && binding.start < s.start && s.end > end && paths::reenters(p,s,ref mut c.work)) { end = s.end; } i += 1;
        }
    } return end;
}
fn usize held_end(ref h::Program p, usize id, h::Expression e,ref mut Context c) {
    var usize end=loop_end(p,e,c);var usize i=1;
    while(i<=h::expression_count(p)){
        let parent=h::expression_at(p,i);let u32 kind=h::expression_code(parent.kind);
        if((kind==13 || kind==17 || kind==25 || kind==28 || kind==29 || kind==30 || kind==4 || kind==10 || kind==14) && parent.left==id && parent.end>end){end=parent.end;}
        if(kind==6){var usize j=0;while(j<parent.edge_count){if(h::edge_at(p,parent.edge_start+j)==id && parent.end>end){end=parent.end;}j+=1;}}i+=1;
    }
    i = 1; while (i <= h::statement_count(p)) {
        let s = h::statement_at(p,i);
        if (h::statement_code(s.kind) == 3) {
            let target = h::expression_at(p,s.target);
            if (e.start >= target.start && e.end <= target.end && s.end > end) { end = s.end; }
        } i += 1;
    }
    return end;
}
fn usize first(ref h::Program p, ref mut Context c,ref mut v::Vec<usize> tables, ref mut v::Vec<owners::Entry> sets, ref mut v::Vec<usize> members, ref mut v::Vec<cached::Slot> cache, usize id, usize depth) {
    let e=h::expression_at(p,id);let u32 kind=h::expression_code(e.kind);
    if(depth>=64){discard fail(c,2,e.start,e.end);return e.start;}
    if(kind==17 || kind==25 || kind==29 || kind==30 || kind==11 || kind==18 && e.edge_count==1){return first(p,c,tables,sets,members,cache,if(kind==18){h::edge_at(p,e.edge_start)}else{e.left},depth+1);}
    if(kind==7){let usize tail=h::block_at(p,e.block).tail;return if(tail==0){e.start}else{first(p,c,tables,sets,members,cache,tail,depth+1)};}
    if(kind==8){let usize yes=h::block_at(p,e.block).tail;if(yes==0 || e.alternative==0){return e.start;}let usize a=first(p,c,tables,sets,members,cache,yes,depth+1);let usize b=first(p,c,tables,sets,members,cache,e.alternative,depth+1);return if(a<b){a}else{b};}
    if(kind==6){
        let usize returned=summary(p,c,tables,sets,members,cache,e.binding,depth+1);var usize i=0;while(i<h::function_count(p)){
            let f=h::function_at(p,i);if(f.binding==e.binding){var usize j=0;while(j<f.parameter_count){if(h::edge_at(p,f.parameter_start+j)==returned){return first(p,c,tables,sets,members,cache,h::edge_at(p,e.edge_start+j),depth+1);}j+=1;}}i+=1;
        }
    }return e.start;
}
pub fn usize base(ref h::Program p,usize id){var usize at=id;var usize depth=0;while(at!=0 && depth<64){let e=h::expression_at(p,at);let u32 kind=h::expression_code(e.kind);if(kind==3){return at;}if(kind!=10 && kind!=13 && !(kind==4 && h::operator_code(e.operator)==5)){return 0;}at=e.left;depth+=1;}return 0;}
fn usize root(ref h::Program p, usize id) {
    var usize current = id; var usize depth = 0; while (current != 0 && depth < 64) {
        let e = h::expression_at(p,current); let u32 kind = h::expression_code(e.kind);
        if (kind == 3) { return e.binding; }
        if (kind != 10 && kind != 13 && !(kind==4 && h::operator_code(e.operator)==5)) { return 0; } current = e.left; depth += 1;
    } return 0;
}
pub fn h::Problem check(ref h::Program p) {
    // Parameter-origin summaries also apply when no explicit borrow/slice
    // occurs in this module. Static literals alone retain the previous lane.
    var bool needed = h::has_views(p); var usize parameter = 1;
    while (!needed && parameter <= h::binding_count(p)) {
        let b = h::binding_at(p,parameter);
        needed = h::binding_code(b.kind) == 2 && byte_view(h::type_code(b.type));
        parameter += 1;
    }
    if (!needed) { return h::Problem { .code=0,.start=0,.end=0 }; }
    var v::Vec<owners::Entry> sets=v::empty<owners::Entry>();var v::Vec<usize> members=v::empty<usize>();
    var usize cache_work=0;var v::Vec<cached::Slot> cache=switch(cached::create(p,ref mut cache_work)){case .ok(value):{value}case .err(code):{return h::Problem{.code,.start=0,.end=0};}};
    var Context c=Context{.work=cache_work,.origin_depth=0,.cache_origins=false,.problem=h::Problem{.code=0,.start=0,.end=0}};var tables=switch(table()){case .ok(p):{p}case .err(code):{return h::Problem{.code,.start=0,.end=0};}};
    var usize i = 1; while (i <= h::statement_count(p)) {
        let s = h::statement_at(p,i); if (h::statement_code(s.kind) == 1) { write(ref mut tables,0,s.binding,s.expression); } i += 1;
    }
    i=1;while(i<=h::expression_count(p)){let e=h::expression_at(p,i);if(h::expression_code(e.kind)==19){var usize a=0;while(a<e.edge_count){let arm=h::expression_at(p,h::edge_at(p,e.edge_start+a));var usize j=0;while(j<arm.edge_count){let capture=h::expression_at(p,h::edge_at(p,arm.edge_start+j));if(capture.edge_count==1){write(ref mut tables,0,h::edge_at(p,capture.edge_start),e.left);}j+=1;}a+=1;}}i+=1;}
    i = 1; while (i <= h::expression_count(p)) {
        let e = h::expression_at(p,i); if (h::expression_code(e.kind) == 3 && byte_view(h::type_code(e.type)) && cached::reachable(ref cache,i,false,ref mut c.work)) {
            let usize end = held_end(p,i,e,ref mut c); if (end > read(ref tables,1,e.binding)) { write(ref mut tables,1,e.binding,end); }
        } i += 1;
    }
    var bool changed=true;var usize rounds=0;while(changed && c.problem.code==0){changed=false;rounds+=1;if(rounds>64){return h::Problem{.code=2,.start=0,.end=0};}reset_colors(ref mut tables);i=0;while(i<h::function_count(p)){let f=h::function_at(p,i);if(byte_view(h::type_code(h::binding_at(p,f.binding).type))){let usize old=read(ref tables,2,f.binding);let usize next=summary(p,ref mut c,ref mut tables,ref mut sets,ref mut members,ref mut cache,f.binding,0);changed=changed || old!=next;}i+=1;}}
    i=0;while(i<h::function_count(p)){let f=h::function_at(p,i);if(byte_view(h::type_code(h::binding_at(p,f.binding).type)) && read(ref tables,2,f.binding)==65538){discard fail(ref mut c,2103,f.start,f.end);}i+=1;}
    // Initializers and return summaries are fixed before this pass. Reuse the
    // same origins for assignment/move conflicts instead of retracing every
    // loan for every scalar store in the complete authenticated SDK.
    c.cache_origins=true;
    var v::Vec<usize> origins = v::empty<usize>(); i = 0;
    while (i <= h::expression_count(p)) {
        let usize value = origin(p,ref mut c,ref mut tables,ref mut sets,ref mut members,ref mut cache,i,0);
        if (!v::push<usize>(ref mut origins,value).is_ok) { return h::Problem { .code=3,.start=0,.end=0 }; } i += 1;
    }
    if(c.work>1048576){return h::Problem{.code=2,.start=0,.end=0};}
    if (c.problem.code != 0) { return c.problem; }
    i = 1; while (i <= h::binding_count(p)) {
        let b = h::binding_at(p,i); if (byte_view(h::type_code(b.type)) && read(ref tables,0,i) != 0) {
            let usize initial = read(ref tables,0,i); let usize owner = *v::at<usize>(ref origins,initial);
            if (owner != 0 && (h::binding_code(b.kind) == 4 || !scope_check(p,ref sets,ref members,owner,b.scope,false))) { discard fail(ref mut c,2102,b.start,b.end); }
        } i += 1;
    }
    i = 1; while (i <= h::block_count(p)) {
        let b = h::block_at(p,i); let usize owner = *v::at<usize>(ref origins,b.tail);
        if (owner != 0 && scope_check(p,ref sets,ref members,owner,b.scope,true)) { discard fail(ref mut c,2102,b.start,b.end); } i += 1;
    }
    i = 1; while (i <= h::statement_count(p) && c.problem.code == 0) {
        let s = h::statement_at(p,i);
        if(h::statement_code(s.kind)==11){let call=h::expression_at(p,s.expression);var usize j=0;while(j<call.edge_count){let arg=h::expression_at(p,h::edge_at(p,call.edge_start+j));let u32 type=h::type_code(arg.type);if(type==9 || byte_view(type)){discard fail(ref mut c,4,arg.start,arg.end);}j+=1;}}
        if (h::statement_code(s.kind) == 3 && cached::reachable(ref cache,i,true,ref mut c.work)) {
            let usize owner = root(p,s.target);
            let bool through_mutable = owner != 0 && (borrows::mutable(h::type_code(h::binding_at(p,owner).type)));
            if (*v::at<usize>(ref origins,s.expression) != 0 && byte_view(h::type_code(h::expression_at(p,s.target).type))) { discard fail(ref mut c,2102,s.start,s.end); }
            var usize j = 1; while (!through_mutable && j <= h::expression_count(p) && c.problem.code == 0) {
                let e = h::expression_at(p,j); let usize loan = *v::at<usize>(ref origins,j);
                if (loan != 0 && owners::contains(ref sets,ref members,loan,owner,ref mut c.work)) {
                    let usize start=first(p,ref mut c,ref mut tables,ref mut sets,ref mut members,ref mut cache,j,0);var usize end = held_end(p,j,e,ref mut c); var usize k = 1;
                    while (k <= h::binding_count(p)) { if (read(ref tables,0,k) == j && read(ref tables,1,k) > end) { end = read(ref tables,1,k); } k += 1; }
                    // Arguments stay held through later arguments and the call.
                    k = 1; while (k <= h::expression_count(p)) {
                        let call = h::expression_at(p,k); if (h::expression_code(call.kind) == 6) {
                            var usize a = 0; while (a < call.edge_count) { if (h::edge_at(p,call.edge_start + a) == j && call.end > end) { end = call.end; } a += 1; }
                        } k += 1;
                    }
                    // Source spans are half-open: an assignment ending at the borrow
                    // start completes before the loan begins.
                    if (start < s.end && end > s.start && !paths::separate(p,start,s.start,s.end,ref mut c.work) && !paths::separate(p,end-1,s.start,s.end,ref mut c.work)) { discard fail(ref mut c,2101,s.start,s.end); }
                } j += 1;
            }
        } i += 1;
    }
    // A whole resource move/burn is a competing access, just like replacement.
    i=1;while(i<=h::expression_count(p) && c.problem.code==0){let moved=h::expression_at(p,i);if(h::expression_code(moved.kind)==24){var usize j=1;while(j<=h::expression_count(p)){let loan=h::expression_at(p,j);if(byte_view(h::type_code(loan.type)) && owners::contains(ref sets,ref members,*v::at<usize>(ref origins,j),moved.binding,ref mut c.work)){let usize begin=first(p,ref mut c,ref mut tables,ref mut sets,ref mut members,ref mut cache,j,0);var usize end=held_end(p,j,loan,ref mut c);var usize local=1;while(local<=h::binding_count(p)){if(read(ref tables,0,local)==j && read(ref tables,1,local)>end){end=read(ref tables,1,local);}local+=1;}if(begin<moved.end && moved.start<end && cached::reachable(ref cache,i,false,ref mut c.work) && !paths::separate(p,begin,moved.start,moved.end,ref mut c.work) && !paths::separate(p,end-1,moved.start,moved.end,ref mut c.work)){discard fail(ref mut c,2101,moved.start,moved.end);}}j+=1;}}i+=1;}
    if(c.work>1048576){return h::Problem{.code=2,.start=0,.end=0};}
    if (c.problem.code != 0) { return c.problem; }
    return exclusive::check(p,v::view_usize(ref tables)[..1153],v::view_usize(ref tables)[1153..2306],v::view_usize(ref tables)[2306..3459],ref origins,ref sets,ref members,ref cache);
}

// One-based parameter position; zero means no proven input-origin substitution.
fn usize origin_parameter(ref h::Program p,usize callee,usize origin){var usize i=0;while(i<h::function_count(p)){let f=h::function_at(p,i);if(f.binding==callee){var usize j=0;while(j<f.parameter_count){if(h::edge_at(p,f.parameter_start+j)==origin){return j+1;}j+=1;}return 0;}i+=1;}return 0;}
