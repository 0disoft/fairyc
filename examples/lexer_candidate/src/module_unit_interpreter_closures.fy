module module_unit_interpreter_closures;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_closures as proof;
import module_unit_interpreter_values as runtime;
import std::vec as v;

// Private immutable per-test cells: [tag, creator, construction node, helper,
// capture count, c0, c1, c2, c3]. These are checked IR handles, never native
// pointers or references into a creator's locals. Copies share frozen cells.
pub resource Invocation{usize target;v::Vec<usize> arguments;cleanup{}}
pub fn usize frame(ref Invocation p){return p.target;}
pub fn ref v::Vec<usize> arguments(ref Invocation p){return ref p.arguments;}
fn usize tag(){return 1129074518;}
fn runtime::Outcome target(ref ir::Program p,usize frame,usize id,usize count,ref mut usize steps){
    if(frame==0 || frame>ir::count(p) || count>4){return runtime::failure(4);}let b=ir::at(p,frame);
    if(id==0 || id>ir::node_count(b)){return runtime::failure(4);}let e=ir::node_at(b,id-1).value;
    if(e.kind!=36 || !proof::leaf(ir::cleanup_ir(p),e.type)){return runtime::failure(4);}
    var usize link=e.right;var usize actual=0;while(link!=0){
        if(*steps==0){return runtime::failure(2);};*steps=*steps-1;
        let at=ir::node(b,link);if(at==0 || actual>=4){return runtime::failure(4);}let pair=ir::node_at(b,at-1).value;
        if(pair.kind!=7 || pair.right>=link){return runtime::failure(4);}actual+=1;link=pair.right;
    }if(actual!=count){return runtime::failure(4);}
    var usize work=0;let helper=proof::helper(p,frame,e.target,count,ref mut work);
    if(work>*steps){return runtime::failure(2);};*steps=*steps-work;
    return if(helper!=0){runtime::result(helper)}else{runtime::failure(4)};
}
pub fn runtime::Outcome construct(ref ir::Program p,usize frame,usize id,ref v::Vec<usize> values,ref mut v::Vec<usize> heap,ref mut usize steps){
    if(frame==0 || frame>ir::count(p)){return runtime::failure(4);}let b=ir::at(p,frame);if(id==0 || id>ir::node_count(b)){return runtime::failure(4);}let e=ir::node_at(b,id-1).value;
    var usize[4] captures=[0;4];var usize count=0;var usize link=e.right;
    // IR argument links are reversed. CFG has already evaluated every snapshot
    // once in source order. Read those results, never the current binding roots.
    while(link!=0){
        if(*steps==0){return runtime::failure(2);};*steps=*steps-1;
        let at=ir::node(b,link);if(at==0 || count>=4){return runtime::failure(4);}let pair=ir::node_at(b,at-1).value;let child=ir::node(b,pair.left);
        if(pair.kind!=7 || pair.right>=link || child==0 || child>=v::len<usize>(values)){return runtime::failure(4);}let bits=*v::at<usize>(values,child);
        if(bits>4294967295){return runtime::failure(4);}captures[count]=bits;count+=1;link=pair.right;
    }
    let helper=target(p,frame,id,count,steps);if(helper.error!=0){return helper;}let cell=runtime::allocate(heap,9,steps);if(cell.error!=0){return cell;}
    {let slot=v::at_mut<usize>(heap,cell.bits);*slot=tag();}{let slot=v::at_mut<usize>(heap,cell.bits+1);*slot=frame;}{let slot=v::at_mut<usize>(heap,cell.bits+2);*slot=id;}{let slot=v::at_mut<usize>(heap,cell.bits+3);*slot=helper.bits;}{let slot=v::at_mut<usize>(heap,cell.bits+4);*slot=count;}
    var usize i=0;while(i<count){{let slot=v::at_mut<usize>(heap,cell.bits+5+i);*slot=captures[count-i-1];}i+=1;}return cell;
}
pub fn Result<Invocation,u32> invocation(ref ir::Program p,ref ir::Body b,usize invocation,ref v::Vec<usize> values,ref v::Vec<usize> heap,ref mut usize steps){
    if(invocation==0 || invocation>ir::call_count(b)){return .err(4);}let call=ir::call_at(b,invocation-1);
    if(call.kind!=4 || !call.checked || call.argument_count!=1 || call.argument_start>=ir::argument_count(b) || call.expression==0 || call.expression>ir::node_count(b)){return .err(4);}
    let e=ir::node_at(b,call.expression-1).value;let operand=ir::node(b,e.left);let arg=ir::argument_at(b,call.argument_start);
    if(operand==0 || operand>=v::len<usize>(values) || arg.expression==0 || arg.expression>=v::len<usize>(values)){return .err(4);}
    // Both operands are frozen expression results from the checked CFG.
    let cell=*v::at<usize>(values,operand);if(cell==0 || cell>v::len<usize>(heap) || *v::at<usize>(heap,cell-1)!=9 || v::len<usize>(heap)-cell<9){return .err(4);}
    if(runtime::item(heap,cell,0).bits!=tag()){return .err(4);}let creator=runtime::item(heap,cell,1).bits;let node=runtime::item(heap,cell,2).bits;let count=runtime::item(heap,cell,4).bits;
    let helper=target(p,creator,node,count,steps);if(helper.error!=0){return .err(helper.error);}if(helper.bits!=runtime::item(heap,cell,3).bits){return .err(4);}
    var arguments=v::empty<usize>();var usize i=0;while(i<4){
        if(*steps==0){return .err(2);};*steps=*steps-1;let bits=runtime::item(heap,cell,5+i).bits;
        if(bits>4294967295 || i>=count && bits!=0){return .err(4);}if(i<count && !v::push<usize>(ref mut arguments,bits).is_ok){return .err(4);}i+=1;
    }
    let bits=*v::at<usize>(values,arg.expression);if(bits>4294967295 || !v::push<usize>(ref mut arguments,bits).is_ok){return .err(4);}
    return .ok(Invocation{.target=helper.bits,.arguments});
}
