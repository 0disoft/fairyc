module module_unit_interpreter;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_project_identified_ir as identified;
import module_constant_arithmetic as math;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;
import project_compiler_core as compiler;
import project_test_options as test_options;
import project_compiler_input as input;

// An interpreter consumes checked owned IR and verified CFGs only. It cannot
// emit C, launch a process, or invoke a host adapter. Values are scalar bits.
// Errors: 1 unsupported, 2 step limit, 3 call depth, 4 invariant/allocation.
pub resource Plan{v::Vec<cfg::Graph> graphs;cleanup{}}
pub struct Outcome{u32 error;u32 trap;usize bits;}
fn Outcome result(usize bits){return Outcome{.error=0,.trap=0,.bits};}
fn Outcome failure(u32 error){return Outcome{.error,.trap=0,.bits=0};}
fn Outcome trapped(u32 trap){return Outcome{.error=0,.trap,.bits=0};}
pub fn ref u8[] detail(u32 error){let text=b"interpreter step limit exceededinterpreter call-depth limit exceededcandidate interpreter supports scalar values and calls only";return if(error==2){text[..31]}else if(error==3){text[31..68]}else{text[68..]};}
pub fn Result<Plan,u32> prepare(ref ir::Program p){let graphs=switch(flow::lower_all(p)){case .ok(g):{g}case .err(e):{return .err(if(e.code==1002){1}else{4});}};return .ok(Plan{.graphs});}
fn u32 scalar(ref ir::Program p,usize type){let t=clean::type_at(ir::cleanup_ir(p),type);return if(t.kind==1){t.scalar}else{0};}
fn bool slots(ref mut v::Vec<usize> values,usize count){if(!v::reserve<usize>(values,count).is_ok){return false;}while(v::len<usize>(values)<count){if(!v::push<usize>(values,0).is_ok){return false;}}return true;}
fn Outcome computed(math::RuntimeValue r){return if(!r.supported){failure(1)}else if(r.trap!=0){trapped(r.trap)}else{result(r.bits)};}
fn Outcome literal(ref ir::Body b,usize id,u32 type){
    if(type==6){return result(if(bytes::equal(ir::node_text(b,id-1),b"true")){1}else{0});}
    let text=ir::node_cooked(b,id-1);var usize bits=0;var usize i=0;if(text.len==0){return failure(4);}while(i<text.len){let digit=text[i] as usize;if(digit<48 || digit>57 || bits>(18446744073709551615-(digit-48))/10){return failure(4);}bits=bits*10+(digit-48);i+=1;}return result(bits);
}
fn Outcome invoke(ref identified::Program owned,ref Plan plan,usize frame,usize invocation,usize site,ref v::Vec<usize> values,ref v::Vec<usize> captures,ref mut usize steps,usize depth){
    let p=identified::ir_program(owned);let b=ir::at(p,frame);if(invocation==0 || invocation>ir::call_count(b)){return failure(4);}let call=ir::call_at(b,invocation-1);var usize work=0;
    let target=switch(identified::call_frame(owned,frame,invocation-1,ref mut work)){case .ok(n):{n}case .err(_):{return failure(4);}};if(target==0){return failure(1);}
    var v::Vec<usize> arguments=v::empty<usize>();var usize i=0;while(i<call.argument_count){let a=ir::argument_at(b,call.argument_start+i);if(a.shared || scalar(p,a.actual)==0){return failure(1);}let value=if(site==0){*v::at<usize>(values,a.expression)}else{*v::at<usize>(captures,cfg::deferred_at(v::at<cfg::Graph>(ref plan.graphs,frame-1),site).capture_start+i+1)};if(!v::push<usize>(ref mut arguments,value).is_ok){return failure(4);}i+=1;}
    return execute(owned,plan,target,ref arguments,steps,depth+1);
}
fn Outcome expression(ref identified::Program owned,ref Plan plan,usize frame,usize id,ref v::Vec<usize> values,ref v::Vec<usize> locals,ref v::Vec<usize> captures,ref mut usize steps,usize depth){
    let p=identified::ir_program(owned);let b=ir::at(p,frame);let e=ir::node_at(b,id-1).value;let type=scalar(p,e.type);if(type==0 || type>7){return failure(1);}let left=lookup::node(b,e.left);let right=lookup::node(b,e.right);let op=ir::node_text(b,id-1);
    if(e.kind==1){return literal(b,id,type);}if(e.kind==2){if(e.binding==0){return failure(1);}return result(*v::at<usize>(locals,e.binding));}if(e.kind==3){return result(*v::at<usize>(values,left));}
    if(e.kind==4){if(e.minimum){return result(if(type==1){2147483648}else{9223372036854775808});}return computed(math::runtime_prefix(type,op,*v::at<usize>(values,left)));}
    if(e.kind==5){return computed(math::runtime_binary(scalar(p,ir::node_at(b,left-1).value.type),op,*v::at<usize>(values,left),*v::at<usize>(values,right)));}
    if(e.kind==23){return computed(math::runtime_cast(scalar(p,ir::node_at(b,left-1).value.type),type,*v::at<usize>(values,left)));}
    if(e.kind==6){if(e.builtin.kind!=0){return failure(1);}var usize i=0;while(i<ir::call_count(b)){if(ir::call_at(b,i).expression==id){return invoke(owned,plan,frame,i+1,0,values,captures,steps,depth);}i+=1;}return failure(4);}
    return failure(1);
}
fn Outcome execute(ref identified::Program owned,ref Plan plan,usize frame,ref v::Vec<usize> arguments,ref mut usize steps,usize depth){
    if(depth>=256){return failure(3);}let p=identified::ir_program(owned);let b=ir::at(p,frame);let g=v::at<cfg::Graph>(ref plan.graphs,frame-1);
    if(scalar(p,clean::frame_at(ir::cleanup_ir(p),frame).result)==0){return failure(1);}
    var v::Vec<usize> values=v::empty<usize>();var v::Vec<usize> locals=v::empty<usize>();var v::Vec<usize> captures=v::empty<usize>();var v::Vec<usize> snapshots=v::empty<usize>();
    if(!slots(ref mut values,ir::node_count(b)+1) || !slots(ref mut locals,ir::local_count(b)+1) || !slots(ref mut captures,cfg::capture_count(g)+1) || !slots(ref mut snapshots,ir::statement_count(b)+1)){return failure(4);}
    var usize parameter=0;var usize i=1;while(i<=ir::local_count(b)){let l=ir::local_at(b,i);if(l.kind==1){if(parameter>=v::len<usize>(arguments)){return failure(4);}if(scalar(p,l.type)==0){return failure(1);}let slot=v::at_mut<usize>(ref mut locals,i);*slot=*v::at<usize>(arguments,parameter);parameter+=1;}i+=1;}if(parameter!=v::len<usize>(arguments)){return failure(4);}
    var usize current=cfg::entry(g);while(current!=0){if(*steps==0){return failure(2);};*steps=*steps-1;let n=cfg::at(g,current);var Outcome r=result(0);var usize next=n.first;
        if(n.kind==1){r=expression(owned,plan,frame,n.source,ref values,ref locals,ref captures,steps,depth);if(r.error==0 && r.trap==0){let slot=v::at_mut<usize>(ref mut values,n.source);*slot=r.bits;}}
        else if(n.kind==2){let stmt=ir::statement_at(b,n.target-1).value;let destination=lookup::node(b,stmt.target);let usize local=if(stmt.kind==2){lookup::local(b,stmt.entry)}else if(destination!=0 && ir::node_at(b,destination-1).value.kind==2){ir::node_at(b,destination-1).value.binding}else{0};if(local==0){return failure(1);}var usize bits=*v::at<usize>(ref values,n.source);let op=ir::assignment_operator(b,n.target-1);if(stmt.kind==4 && !bytes::equal(op,b"=")){r=computed(math::runtime_binary(scalar(p,ir::local_at(b,local).type),op[..op.len-1],*v::at<usize>(ref snapshots,n.target),bits));bits=r.bits;}if(r.error==0 && r.trap==0){let slot=v::at_mut<usize>(ref mut locals,local);*slot=bits;}}
        else if(n.kind==3){next=if(*v::at<usize>(ref values,n.source)!=0){n.first}else{n.second};}
        else if(n.kind==6){return result(if(n.source==0){0}else{*v::at<usize>(ref values,n.source)});}
        else if(n.kind==8){if(*v::at<usize>(ref values,n.source)==0){return trapped(6);}}
        else if(n.kind==10 || n.kind==11){if(n.target!=0){let bits=*v::at<usize>(ref values,n.target);let slot=v::at_mut<usize>(ref mut values,n.source);*slot=bits;}}
        else if(n.kind==12){return trapped(7);}
        else if(n.kind==13){let bits=*v::at<usize>(ref values,n.source);let slot=v::at_mut<usize>(ref mut captures,n.target);*slot=bits;}
        else if(n.kind==14){r=invoke(owned,plan,frame,cfg::deferred_at(g,n.target).invocation,n.target,ref values,ref captures,steps,depth);}
        else if(n.kind==17){let e=ir::node_at(b,n.source-1).value;if(e.kind!=2 || e.binding==0){return failure(1);}let bits=*v::at<usize>(ref locals,e.binding);let slot=v::at_mut<usize>(ref mut snapshots,n.target);*slot=bits;}
        else if(n.kind!=7){return failure(1);}
        if(r.error!=0 || r.trap!=0){return r;}current=next;
    }return failure(4);
}
pub fn Outcome execute_void(ref identified::Program owned,ref Plan plan,usize frame){
    let p=identified::ir_program(owned);if(frame==0 || frame>ir::count(p) || v::len<cfg::Graph>(ref plan.graphs)!=ir::count(p)){return failure(4);}if(scalar(p,clean::frame_at(ir::cleanup_ir(p),frame).result)!=7){return failure(4);}
    let arguments=v::empty<usize>();var usize steps=1000000;return execute(owned,plan,frame,ref arguments,ref mut steps,0);
}

@effects(console) fn i32 report_failure(ref str message,i32 code){return if(io::write_stderr(message).is_ok){code}else{4};}
@effects(console) pub fn i32 tests(ref compiler::Complete complete,ref test_options::TestOptions options,ref v::Vec<test_options::TestIdentity> tests,ref v::Vec<usize> chosen,ref input::Input projected,u32 lock_code){
    var v::Vec<u32> traps=v::empty<u32>();if(v::len<usize>(chosen)==0){return test_options::test_semantic_report(options,tests,chosen,projected,ref traps,lock_code);}
    let owned=compiler::checked_program(complete);let p=identified::ir_program(owned);let plan=switch(prepare(p)){case .ok(p):{p}case .err(code):{return report_failure(if(code==4){"fairyc: internal semantic test preparation failed\n"}else{"fairyc: requested workspace is outside the semantic test interpreter subset: candidate interpreter CFG preparation limit exceeded\n"},if(code==4){4}else{2});}};
    var usize i=0;while(i<v::len<usize>(chosen)){let test=v::at<test_options::TestIdentity>(tests,*v::at<usize>(chosen,i));var usize frame=1;while(frame<=ir::count(p)){let f=clean::frame_at(ir::cleanup_ir(p),frame);if(f.kind==3 && f.module_index==test_options::test_module(test) && f.declaration==test_options::test_declaration(test)+1){break;}frame+=1;}
        let outcome=execute_void(owned,ref plan,frame);if(outcome.error!=0){if(outcome.error==4){return report_failure("fairyc: internal semantic test execution failed\n",4);}var bytes::Buffer message=bytes::empty();if(!test_options::test_append(ref mut message,b"fairyc: requested workspace is outside the semantic test interpreter subset: ") || !test_options::test_append(ref mut message,test_options::test_name(test)) || !test_options::test_append(ref mut message,b": ") || !test_options::test_append(ref mut message,detail(outcome.error)) || !test_options::test_append(ref mut message,b"\n")){return 4;}return if(io::write_stderr_bytes(bytes::view(ref message)).is_ok){2}else{4};}
        if(!v::push<u32>(ref mut traps,outcome.trap).is_ok){return 4;}i+=1;
    }return test_options::test_semantic_report(options,tests,chosen,projected,ref traps,lock_code);
}
