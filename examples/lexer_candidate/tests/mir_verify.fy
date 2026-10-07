module tests::mir_verify;
import mir_model as m;
import mir_verify as verify;
import mir_verify_flow as flow;
import mir_output as output;
import mir_state as state;
import std::bytes as bytes;
import std::vec as v;

fn m::Local local(u32 type, bool parameter) { return m::Local { .function = 1, .binding = 0, .type, .parameter, .mutable = false, .start = 0, .end = 1 }; }
fn m::Target target(u32 result, u32 parameter) { return m::Target { .foreign=false,.link_start=0,.link_end=0, .unsafe_call=false, .effects = 0, .binding = 1, .module_index = 0, .declaration = 1, .result, .signature_start = 0, .signature_count = if (parameter == 0) { 0 } else { 1 }, .start = 0, .end = 1, .external = false }; }
fn m::Function function(usize locals, usize blocks, usize entry, usize parameters) {
    return m::Function { .target = 1, .entry, .local_start = 0, .local_count = locals, .block_start = 0, .block_count = blocks, .parameter_start = 0, .parameter_count = parameters, .start = 0, .end = 1, .public = false };
}
fn m::Operation constant(usize block, usize result) {
    var m::Operation o = state::operation(1,result,1,1,0,0,0,0,1); o.function = 1; o.block = block; o.payload_end = 1; return o;
}
fn m::Program finish(bytes::Buffer payload, v::Vec<m::Target> targets, v::Vec<m::Signature> signatures, v::Vec<m::Function> functions,
    v::Vec<m::Local> locals, v::Vec<m::Block> blocks, v::Vec<m::Operation> operations, v::Vec<usize> edges) {
    var bytes::Buffer source = bytes::empty(); assert(bytes::append(ref mut source,b"fg").is_ok);
    return switch(m::assemble(source,payload,targets,signatures,functions,locals,blocks,operations,edges,0)) {
        case .ok(p): { p } case .err(_): { assert(false); return finish(bytes::empty(),v::empty<m::Target>(),v::empty<m::Signature>(),v::empty<m::Function>(),v::empty<m::Local>(),v::empty<m::Block>(),v::empty<m::Operation>(),v::empty<usize>()); }
    };
}
fn m::Program arithmetic(u32 mutation) {
    var bytes::Buffer payload = bytes::empty(); assert(bytes::append(ref mut payload,if (mutation == 12) { b"01" } else if (mutation == 13) { b"2147483648" } else { b"1" }).is_ok);
    var v::Vec<m::Target> targets = v::empty<m::Target>(); assert(v::push<m::Target>(ref mut targets,target(1,1)).is_ok);
    var v::Vec<m::Signature> signatures = v::empty<m::Signature>(); assert(v::push<m::Signature>(ref mut signatures,m::Signature { .type = 1 }).is_ok);
    var v::Vec<m::Function> functions = v::empty<m::Function>(); assert(v::push<m::Function>(ref mut functions,function(4,if (mutation == 7) { 4 } else { 3 },if (mutation == 7) { 4 } else { 1 },1)).is_ok);
    var v::Vec<m::Local> locals = v::empty<m::Local>(); assert(v::push<m::Local>(ref mut locals,local(1,true)).is_ok);
    var usize i = 0; while (i < 3) { assert(v::push<m::Local>(ref mut locals,local(1,mutation == 11 && i == 2)).is_ok); i += 1; }
    var v::Vec<usize> edges = v::empty<usize>(); assert(v::push<usize>(ref mut edges,1).is_ok);
    if (mutation == 10) { assert(v::push<usize>(ref mut edges,1).is_ok); }
    var m::Operation first = constant(1,2); first.payload_end = bytes::len(ref payload); first.next = 2;
    var m::Operation marker = state::operation(5,0,1,1,1,1,if (mutation == 2) { 4 } else { 2 },0,1); marker.function = 1; marker.block = 1;
    var m::Operation add = state::operation(4,3,if (mutation == 1) { 6 } else { 1 },1,1,1,if (mutation == 2) { 4 } else if (mutation == 3) { 1 } else { 2 },0,1); add.function = 1; add.block = 2;
    if (mutation == 4) { marker.kind = 2; marker.result = 4; marker.operator = 0; marker.left = 2; marker.right = 0; }
    var v::Vec<m::Operation> operations = v::empty<m::Operation>();
    assert(v::push<m::Operation>(ref mut operations,first).is_ok); assert(v::push<m::Operation>(ref mut operations,marker).is_ok); assert(v::push<m::Operation>(ref mut operations,add).is_ok);
    if (mutation == 6 || mutation == 9) { assert(v::push<m::Operation>(ref mut operations,constant(if (mutation == 6) { 3 } else { 2 },4)).is_ok); }
    var m::Block header = state::block(1,0,1); header.first = 1; header.last = 2; header.kind = 4; header.check = 2; header.left = if (mutation == 5) { 2 } else { 1 };
    header.right = marker.right; header.operator = 1; header.type = 1; header.target = 2; header.alternative = 3; header.trap_mask = if (mutation == 8) { 3 } else { 1 };
    if (mutation == 4) { header.kind = 2; header.check = 0; header.left = 0; header.right = 0; header.operator = 0; header.type = 0; header.alternative = 0; header.trap_mask = 0; }
    var m::Block normal = state::block(1,0,1); normal.first = 3; normal.last = 3; normal.kind = 1; normal.value = 3;
    var m::Block trap = state::block(1,0,1); trap.kind = 6; trap.trap_mask = header.trap_mask;
    if (mutation == 4) { trap.kind = 1; trap.value = 1; trap.trap_mask = 0; }
    if (mutation == 6) { trap.first = 4; trap.last = 4; }
    var v::Vec<m::Block> blocks = v::empty<m::Block>(); assert(v::push<m::Block>(ref mut blocks,header).is_ok); assert(v::push<m::Block>(ref mut blocks,normal).is_ok); assert(v::push<m::Block>(ref mut blocks,trap).is_ok);
    if (mutation == 7) { var m::Block bypass = state::block(1,0,1); bypass.kind = 2; bypass.target = 2; assert(v::push<m::Block>(ref mut blocks,bypass).is_ok); }
    return finish(payload,targets,signatures,functions,locals,blocks,operations,edges);
}
fn m::Program diamond(u32 mutation) {
    let bool missing = mutation == 1;
    var bytes::Buffer payload = bytes::empty(); assert(bytes::append(ref mut payload,b"1").is_ok);
    var v::Vec<m::Target> targets = v::empty<m::Target>(); assert(v::push<m::Target>(ref mut targets,target(1,6)).is_ok);
    var v::Vec<m::Signature> signatures = v::empty<m::Signature>(); assert(v::push<m::Signature>(ref mut signatures,m::Signature { .type = 6 }).is_ok);
    var v::Vec<m::Function> functions = v::empty<m::Function>(); assert(v::push<m::Function>(ref mut functions,function(2,4,1,1)).is_ok);
    var v::Vec<m::Local> locals = v::empty<m::Local>(); assert(v::push<m::Local>(ref mut locals,local(6,true)).is_ok); assert(v::push<m::Local>(ref mut locals,local(1,false)).is_ok);
    var v::Vec<usize> edges = v::empty<usize>(); assert(v::push<usize>(ref mut edges,1).is_ok);
    var v::Vec<m::Operation> operations = v::empty<m::Operation>(); assert(v::push<m::Operation>(ref mut operations,constant(2,2)).is_ok);
    if (!missing) { assert(v::push<m::Operation>(ref mut operations,constant(3,2)).is_ok); }
    var m::Block entry = state::block(1,0,1); entry.kind = 3; entry.type = 6; entry.value = 1; entry.target = 2; entry.alternative = 3;
    var m::Block yes = state::block(1,0,1); yes.kind = 2; yes.target = 4; yes.first = 1; yes.last = 1;
    var m::Block no = state::block(1,0,1); no.kind = 2; no.target = 4; if (!missing) { no.first = 2; no.last = 2; }
    var m::Block join = state::block(1,0,1); join.kind = if (mutation == 2) { 7 } else { 1 }; join.value = if (mutation == 2) { 0 } else { 2 };
    var v::Vec<m::Block> blocks = v::empty<m::Block>(); assert(v::push<m::Block>(ref mut blocks,entry).is_ok); assert(v::push<m::Block>(ref mut blocks,yes).is_ok); assert(v::push<m::Block>(ref mut blocks,no).is_ok); assert(v::push<m::Block>(ref mut blocks,join).is_ok);
    return finish(payload,targets,signatures,functions,locals,blocks,operations,edges);
}
// One native process per profile runs the whole mutation inventory. Every
// negative seed passes assemble, so rejection belongs to the new verifier.
fn m::Program call(u32 mutation) {
    var bytes::Buffer payload = bytes::empty(); assert(bytes::append(ref mut payload,b"1").is_ok);
    var v::Vec<m::Target> targets = v::empty<m::Target>(); assert(v::push<m::Target>(ref mut targets,target(1,0)).is_ok);
    var m::Target callee = target(1,1); callee.binding = if (mutation == 5) { 1 } else { 2 }; callee.declaration = 2; callee.start = 1; callee.end = 2;
    assert(v::push<m::Target>(ref mut targets,callee).is_ok);
    var v::Vec<m::Signature> signatures = v::empty<m::Signature>(); assert(v::push<m::Signature>(ref mut signatures,m::Signature { .type = 1 }).is_ok);
    var v::Vec<m::Function> functions = v::empty<m::Function>(); assert(v::push<m::Function>(ref mut functions,function(2,2,1,0)).is_ok);
    if (mutation != 6) {
        var m::Function body = function(1,1,3,1); body.target = 2; body.local_start = 2; body.block_start = 2; body.parameter_start = 1; body.start = 1; body.end = 2;
        assert(v::push<m::Function>(ref mut functions,body).is_ok);
    }
    var v::Vec<m::Local> locals = v::empty<m::Local>(); assert(v::push<m::Local>(ref mut locals,local(if (mutation == 3) { 6 } else { 1 },false)).is_ok);
    assert(v::push<m::Local>(ref mut locals,local(if (mutation == 2) { 6 } else { 1 },false)).is_ok);
    if (mutation != 6) { var m::Local argument = local(1,true); argument.function = 2; argument.start = 1; argument.end = 2; assert(v::push<m::Local>(ref mut locals,argument).is_ok); }
    var v::Vec<usize> edges = v::empty<usize>(); assert(v::push<usize>(ref mut edges,if (mutation == 1) { 2 } else { 1 }).is_ok);
    if (mutation != 6) { assert(v::push<usize>(ref mut edges,3).is_ok); }
    var m::Operation value = constant(1,1);
    if (mutation == 3) { value.type = 6; value.owner_type = 6; value.payload_end = 0; }
    var v::Vec<m::Operation> operations = v::empty<m::Operation>(); assert(v::push<m::Operation>(ref mut operations,value).is_ok);
    var m::Block invoke = state::block(1,0,1); invoke.first = 1; invoke.last = 1; invoke.kind = 5; invoke.callee = 2; invoke.destination = if (mutation == 4) { 0 } else { 2 }; invoke.argument_count = 1; invoke.target = 2;
    var m::Block done = state::block(1,0,1); done.kind = 1; done.value = if (mutation == 2 || mutation == 4) { 1 } else { 2 };
    var v::Vec<m::Block> blocks = v::empty<m::Block>(); assert(v::push<m::Block>(ref mut blocks,invoke).is_ok); assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    if (mutation != 6) { var m::Block returned = state::block(2,1,2); returned.kind = 1; returned.value = 3; assert(v::push<m::Block>(ref mut blocks,returned).is_ok); }
    return finish(payload,targets,signatures,functions,locals,blocks,operations,edges);
}
fn m::Program loop(bool missing) {
    var bytes::Buffer payload = bytes::empty(); assert(bytes::append(ref mut payload,b"1").is_ok);
    var v::Vec<m::Target> targets = v::empty<m::Target>(); assert(v::push<m::Target>(ref mut targets,target(1,6)).is_ok);
    var v::Vec<m::Signature> signatures = v::empty<m::Signature>(); assert(v::push<m::Signature>(ref mut signatures,m::Signature { .type = 6 }).is_ok);
    var v::Vec<m::Function> functions = v::empty<m::Function>(); assert(v::push<m::Function>(ref mut functions,function(2,3,1,1)).is_ok);
    var v::Vec<m::Local> locals = v::empty<m::Local>(); assert(v::push<m::Local>(ref mut locals,local(6,true)).is_ok); assert(v::push<m::Local>(ref mut locals,local(1,false)).is_ok);
    var v::Vec<usize> edges = v::empty<usize>(); assert(v::push<usize>(ref mut edges,1).is_ok);
    var v::Vec<m::Operation> operations = v::empty<m::Operation>();
    if (!missing) { assert(v::push<m::Operation>(ref mut operations,constant(1,2)).is_ok); }
    assert(v::push<m::Operation>(ref mut operations,constant(2,2)).is_ok);
    var m::Block entry = state::block(1,0,1); entry.kind = 3; entry.type = 6; entry.value = 1; entry.target = 2; entry.alternative = 3;
    if (!missing) { entry.first = 1; entry.last = 1; }
    var m::Block body = state::block(1,0,1); body.kind = 2; body.target = 1; body.first = if (missing) { 1 } else { 2 }; body.last = body.first;
    var m::Block exit = state::block(1,0,1); exit.kind = 1; exit.value = 2;
    var v::Vec<m::Block> blocks = v::empty<m::Block>(); assert(v::push<m::Block>(ref mut blocks,entry).is_ok); assert(v::push<m::Block>(ref mut blocks,body).is_ok); assert(v::push<m::Block>(ref mut blocks,exit).is_ok);
    return finish(payload,targets,signatures,functions,locals,blocks,operations,edges);
}
@test
fn void scalar_verifier_mutations_and_predecessor_intersections() {
    let m::Program good = arithmetic(0); assert(verify::check(ref good).code == 0);
    var u32 mutation = 1;
    while (mutation <= 13) {
        let m::Program bad = arithmetic(mutation);
        let u32 expected = if (mutation == 2) { 1002 } else if (mutation >= 3 && mutation <= 8) { 1501 } else { 1001 };
        assert(verify::check(ref bad).code == expected); mutation += 1;
    }
    let m::Program joined = diamond(0); assert(verify::check(ref joined).code == 0);
    let m::Program one_path = diamond(1); assert(verify::check(ref one_path).code == 1002);
    let m::Program unreachable = diamond(2); assert(verify::check(ref unreachable).code == 1501);
    let m::Program forged = asserted_parameter(); assert(verify::check(ref forged).code == 1501);
    let m::Program calls = call(0); assert(verify::check(ref calls).code == 0);
    mutation = 1; while (mutation <= 6) { let m::Program bad = call(mutation); assert(verify::check(ref bad).code == if (mutation == 1) { 1002 } else { 1001 }); mutation += 1; }
    let m::Program loop_initialized = loop(false); assert(verify::check(ref loop_initialized).code == 0);
    let m::Program loop_body_only = loop(true); assert(verify::check(ref loop_body_only).code == 1002);
    var flow::Budget exhausted = flow::Budget { .work = 67108864 }; assert(flow::function(ref good,1,ref mut exhausted).code == 2);
    var flow::Budget invalid = flow::Budget { .work = 67108865 }; assert(flow::function(ref good,1,ref mut invalid).code == 2);
    var bytes::Buffer rendered = bytes::empty(); assert(bytes::append(ref mut rendered,b"prefix").is_ok);
    assert(output::render(ref one_path,ref mut rendered) == 1002 && bytes::equal(bytes::view(ref rendered),b"prefix"));
}

// A false assignment in dead code cannot prove that a bool parameter is false
// at an earlier assertion. Entry initialization comes from the caller.
fn m::Program asserted_parameter() {
    var v::Vec<m::Target> targets = v::empty<m::Target>(); assert(v::push<m::Target>(ref mut targets,target(1,6)).is_ok);
    var v::Vec<m::Signature> signatures = v::empty<m::Signature>(); assert(v::push<m::Signature>(ref mut signatures,m::Signature { .type = 6 }).is_ok);
    var v::Vec<m::Function> functions = v::empty<m::Function>(); assert(v::push<m::Function>(ref mut functions,function(2,4,1,1)).is_ok);
    var v::Vec<m::Local> locals = v::empty<m::Local>(); assert(v::push<m::Local>(ref mut locals,local(6,true)).is_ok); assert(v::push<m::Local>(ref mut locals,local(1,false)).is_ok);
    var v::Vec<usize> edges = v::empty<usize>(); assert(v::push<usize>(ref mut edges,1).is_ok);
    var m::Operation check = state::operation(5,0,6,6,0,1,0,0,1); check.function = 1; check.block = 1;
    var m::Operation false_value = state::operation(1,1,6,6,0,0,0,0,1); false_value.function = 1; false_value.block = 4;
    var v::Vec<m::Operation> operations = v::empty<m::Operation>(); assert(v::push<m::Operation>(ref mut operations,check).is_ok); assert(v::push<m::Operation>(ref mut operations,false_value).is_ok);
    var m::Block entry = state::block(1,0,1); entry.kind = 4; entry.first = 1; entry.last = 1; entry.check = 1; entry.left = 1; entry.type = 6; entry.target = 2; entry.alternative = 3; entry.trap_mask = 16;
    var m::Block unreachable = state::block(1,0,1); unreachable.kind = 7;
    var m::Block trap = state::block(1,0,1); trap.kind = 6; trap.trap_mask = 16;
    var m::Block dead = state::block(1,0,1); dead.kind = 1; dead.value = 2; dead.first = 2; dead.last = 2;
    var v::Vec<m::Block> blocks = v::empty<m::Block>(); assert(v::push<m::Block>(ref mut blocks,entry).is_ok); assert(v::push<m::Block>(ref mut blocks,unreachable).is_ok); assert(v::push<m::Block>(ref mut blocks,trap).is_ok); assert(v::push<m::Block>(ref mut blocks,dead).is_ok);
    return finish(bytes::empty(),targets,signatures,functions,locals,blocks,operations,edges);
}
