module module_unit_c_emit;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_body_ir_lookup as lookup;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import module_unit_c_literals as literals;
import module_unit_c_switches as switches;
import module_unit_c_sdk as sdk;
import module_unit_c_bytes as byte_compare;
import module_unit_c_callable_index as contracts;
import module_unit_c_frame_index as targets;
import module_unit_c_reachability as reachable;
import module_unit_c_region as lifecycle;
import module_unit_c_raw_views as raw_views;
import module_unit_c_storage as storage;
import module_unit_c_iteration as iteration;
import module_unit_c_borrowed_arrays as borrowed_arrays;
import module_unit_c_names as names;
import module_unit_c_calls as calls;
import module_identified_ir as identified;
import module_project_identified_ir as project;
import module_project_c_owner as owner;
import project_compiler_core as compiler;
import ffi_symbols as ffi;
import candidate_limits as limits;
import c_text as w;
import c_abi_contract as abi;
import std::bytes as bytes;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
fn bool snapshot(ref mut bytes::Buffer out,ref mut w::State s,usize id){return w::text(out,s,b"fy_snapshot_") && w::number(out,s,id);}
fn bool store(ref ir::Program p,ref ir::Body b,cfg::Block n,ref mut bytes::Buffer out,ref mut w::State s){
    let stmt=ir::statement_at(b,n.target-1).value;let type=scalar::scalar(p,ir::node_at(b,n.source-1).value.type);var usize local=0;
    if(stmt.kind==2){local=lookup::local(b,stmt.entry);}else{let id=lookup::node(b,stmt.target);if(id!=0){let e=ir::node_at(b,id-1).value;if(e.kind==2){local=e.binding;}}}
    if(stmt.kind==2 && (local==0 || ir::local_at(b,local).type!=ir::node_at(b,n.source-1).value.type) || type==7){s.code=1001;return false;}
    let target=lookup::node(b,stmt.target);if(stmt.kind==4 && (target==0 || ir::node_at(b,target-1).value.type!=ir::node_at(b,n.source-1).value.type)){s.code=1001;return false;}
    let op=ir::assignment_operator(b,n.target-1);let plain=stmt.kind==2 || bytes::equal(op,b"=");
    if(!(if(stmt.kind==2){w::local(out,s,local)}else{aggregate::place(p,b,target,out,s,0)}) || !w::text(out,s,b"=")){return false;}
    if(plain){if(!scalar::value(out,s,n.source)){return false;}}
    else{if(op.len<2 || op[op.len-1]!=61){s.code=1001;return false;}let code=scalar::operator(op[..op.len-1]);if(code==0 || code==3 || code==4 || code>12 || type==0 || type==6){s.code=1004;return false;}if(!scalar::math_start(out,s,type,code) || !snapshot(out,s,n.target) || !w::text(out,s,b",") || !scalar::value(out,s,n.source) || !scalar::math_end(out,s,type)){return false;}}
    return w::text(out,s,b";\n");
}
fn bool function(ref ir::Program p,ref names::Table named,ref sdk::Plan plan,ref contracts::Index indexed,ref targets::Index body_index,ref cfg::Graph g,ref mut bytes::Buffer out,ref mut w::State s,ref mut usize work,ref mut usize failed_start,ref mut usize failed_end,bool shared){
    let b=ir::at(p,cfg::frame(g));if(!calls::signature(p,named,cfg::frame(g),out,s,shared) || !w::text(out,s,b"{\n")){return false;}var usize i=1;
    if(cfg::frame(g)==sdk::byte_equal_frame(plan)){return byte_compare::body(b,out,s);}
    let arrays=switch(borrowed_arrays::build(p,b,g,work)){case .ok(a):{a}case .err(code):{s.code=code;return false;}};
    while(i<=ir::local_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let l=ir::local_at(b,i);if(scalar::scalar(p,l.type)==7){s.code=1004;return false;}if(l.kind!=1 && l.kind!=10 && l.kind!=6 && (!aggregate::type(p,l.type,out,s) || !w::text(out,s,b" ") || !w::local(out,s,i) || !w::text(out,s,b";\n"))){return false;}i+=1;}
    if(!iteration::locals(g,out,s,work) || !lifecycle::locals(b,out,s,work)){return false;}
    i=1;while(i<=ir::node_count(b)){if(!budget::charge(work,1)){s.code=1002;return false;}let e=ir::node_at(b,i-1).value;if(e.state==1 && !borrowed_arrays::omitted(ref arrays,i)){let type=scalar::scalar(p,e.type);if(type!=7 && (!aggregate::type(p,e.type,out,s) || !w::text(out,s,b" ") || !scalar::value(out,s,i) || !w::text(out,s,b";\n"))){return false;}}i+=1;}
    i=1;while(i<=cfg::capture_count(g)){let c=cfg::capture_at(g,i);if(!w::type(out,s,scalar::scalar(p,c.type)) || !w::text(out,s,b" ") || !scalar::capture(out,s,i) || !w::text(out,s,b";\n")){return false;}i+=1;}
    i=1;while(i<=cfg::count(g)){let n=cfg::at(g,i);if(n.kind==17 && (!aggregate::type(p,ir::node_at(b,n.source-1).value.type,out,s) || !w::text(out,s,b" ") || !snapshot(out,s,n.target) || !w::text(out,s,b";\n"))){return false;}i+=1;}
    if(!w::jump(out,s,cfg::entry(g))){return false;}i=1;
    while(i<=cfg::count(g)){if(!budget::charge(work,1)){s.code=1002;return false;}let n=cfg::at(g,i);*failed_start=n.start;*failed_end=n.end;if(!w::text(out,s,b"fy_b_") || !w::number(out,s,i) || !w::text(out,s,b":;\n")){return false;}
        if(n.kind==1){let e=ir::node_at(b,n.source-1).value;let left=lookup::node(b,e.left);let variant=e.kind==6 && left!=0 && ir::node_at(b,left-1).value.kind==28;
            if(borrowed_arrays::omitted(ref arrays,n.source)){}
            else if(e.kind==8 && borrowed_arrays::field_omitted(ref arrays,left)){if(!borrowed_arrays::read_field(p,b,n.source,out,s)){return false;}}
            else if(e.kind==9 && borrowed_arrays::omitted(ref arrays,left)){if(!borrowed_arrays::read(p,b,n.source,out,s)){return false;}}
            else if(e.kind==6 && e.builtin.kind>=101 && e.builtin.kind<=107){if(!storage::expression(p,b,n.source,out,s,work)){return false;}}
            else if(e.kind==6 && e.builtin.kind>=9 && e.builtin.kind<=12){if(!raw_views::expression(p,b,n.source,out,s,work)){return false;}}
            else if(e.kind==6 && e.builtin.kind==300){if(!lifecycle::placement(p,plan,b,n.source,out,s,work)){return false;}}
            else if(clean::type_at(ir::cleanup_ir(p),e.type).kind==19){if(!lifecycle::value(b,n.source,out,s,work)){return false;}}
            else if(e.kind==6 && !variant){if(e.builtin.kind!=0){s.code=1004;return false;}if(!calls::call(p,named,plan,indexed,body_index,g,n.source,0,out,s,work)){return false;}}
            else if(e.kind==1 && literals::descriptor(clean::type_at(ir::cleanup_ir(p),e.type))){if(!literals::expression(p,b,n.source,out,s,work)){return false;}}
            else if((e.kind==1 || e.kind==23) && clean::type_at(ir::cleanup_ir(p),e.type).kind==12 || e.kind==2 || e.kind==3 || e.kind==8 || e.kind==9 || e.kind==10 || e.kind==21 || e.kind==22 || e.kind==24 || e.kind==26 || e.kind==28 || e.kind==29 || e.kind==30 || e.kind==4 && bytes::equal(ir::node_text(b,n.source-1),b"*") || variant){if(!aggregate::expression(p,b,n.source,out,s,work)){return false;}}
            else if(!scalar::expression(p,b,n.source,out,s)){return false;}}
        else if(n.kind==2){if(!store(p,b,n,out,s)){return false;}}
        else if(n.kind==3){if(!w::text(out,s,b"if(") || !scalar::value(out,s,n.source) || !w::text(out,s,b"){") || !w::jump(out,s,n.first) || !w::text(out,s,b"}else{") || !w::jump(out,s,n.second) || !w::text(out,s,b"}\n")){return false;}}
        else if(n.kind==4){if(!w::text(out,s,b"if(") || !scalar::value(out,s,n.source) || !w::text(out,s,b".fy_tag){") || !w::jump(out,s,n.first) || !w::text(out,s,b"}else{") || !w::jump(out,s,n.second) || !w::text(out,s,b"}\n")){return false;}}
        else if(n.kind==5){let d=clean::drop_at(ir::cleanup_ir(p),n.drop-1);let o=clean::operand_at(ir::cleanup_ir(p),d.operand);if(!aggregate::drop_call(p,o.type,out,s) || !w::text(out,s,b"&") || !(if(o.kind==1){w::local(out,s,o.id)}else{scalar::value(out,s,lookup::node(b,o.id))}) || !w::text(out,s,b");\n")){return false;}}
        else if(n.kind==6){if(!w::text(out,s,b"return") || n.source!=0 && (!w::text(out,s,b" ") || !scalar::value(out,s,n.source)) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==8){if(!w::text(out,s,b"if(!") || !scalar::value(out,s,n.source) || !w::text(out,s,b")fy_runtime_trap(5);\n")){return false;}}
        else if(n.kind==9){let f=clean::frame_at(ir::cleanup_ir(p),cfg::frame(g));if(!w::text(out,s,b"return (") || !aggregate::type(p,f.result,out,s) || !w::text(out,s,b"){.fy_tag=0,.fy_payload.fy_err=") || !scalar::value(out,s,n.source) || !w::text(out,s,b".fy_payload.fy_err};\n")){return false;}}
        else if(n.kind==10 || n.kind==11){let type=scalar::scalar(p,ir::node_at(b,n.source-1).value.type);if(type!=7){if(n.target==0){s.code=1001;return false;}if(!scalar::value(out,s,n.source) || !w::text(out,s,b"=") || !scalar::value(out,s,n.target) || !w::text(out,s,b";\n")){return false;}}}
        else if(n.kind==12){if(!w::text(out,s,b"fy_runtime_trap(7);\n")){return false;}}
        else if(n.kind==13){if(!scalar::capture(out,s,n.target) || !w::text(out,s,b"=") || !scalar::value(out,s,n.source) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==14){if(!calls::call(p,named,plan,indexed,body_index,g,n.source,n.target,out,s,work)){return false;}}
        else if(n.kind==15 || n.kind==16){if(!lifecycle::block(b,n,out,s)){return false;}}
        else if(n.kind==17){if(!snapshot(out,s,n.target) || !w::text(out,s,b"=") || !aggregate::place(p,b,n.source,out,s,0) || !w::text(out,s,b";\n")){return false;}}
        else if(n.kind==18){if(!switches::branch(p,b,n,out,s,work)){return false;}}
        else if(n.kind==19){if(!switches::seed(p,b,n,out,s,work)){return false;}}
        else if(n.kind==20 || n.kind==21){if(!iteration::block(p,g,n,out,s,work)){return false;}}
        else if(n.kind!=7){s.code=1004;return false;}
        if(n.kind!=3 && n.kind!=4 && n.kind!=6 && n.kind!=9 && n.kind!=12 && n.kind!=18 && n.kind!=21 && !w::jump(out,s,n.first)){return false;}i+=1;
    }return w::text(out,s,b"}\n");
}
fn bool prologue(ref ir::Program p,ref mut bytes::Buffer text,ref mut w::State s,ref mut usize work){
    var usize i=1;var bool posix=false;while(i<=clean::type_count(ir::cleanup_ir(p))){if(!budget::charge(work,1)){s.code=1002;return false;}let t=clean::type_at(ir::cleanup_ir(p),i);if(t.kind==1 && t.scalar==138){posix=true;}i+=1;}
    if(posix && !abi::posix_header(text,s)){return false;}
    if(!w::text(text,s,b"/* fairyc.candidate-module-owned-c/v1 */\n#include <stdint.h>\n#include <limits.h>\n#include <string.h>\n_Static_assert(CHAR_BIT==8 && UINTPTR_MAX==UINT64_MAX,\"requires 64-bit target\");\n_Static_assert(INT32_MIN==(-INT32_MAX-1) && INT64_MIN==(-INT64_MAX-1),\"requires two's complement\");\n_Noreturn void fy_runtime_trap(uint32_t category);\n") || !scalar::helpers(text,s)){return false;}
    // Keep aggregate reads from live owner storage behind an opaque CRT copy
    // on MSVC. Inline aggregate copies cached a pre-view-write array element
    // under /O2; the scalar helpers keep their normal intrinsic behavior.
    if(!w::text(text,s,b"#ifdef _MSC_VER\n#pragma function(memcpy)\n__declspec(noinline)\n#endif\nstatic void fy_unit_copy(void *destination,const void *source,uintptr_t count){memcpy(destination,source,count);}\n#ifdef _MSC_VER\n#pragma intrinsic(memcpy)\n#endif\n")){return false;}
    return true;
}
// A private transport from owned bodies through freshly lowered
// CFGs. No parser/source owner or source join is used. Output is transactional.
pub fn clean::Problem render(ref ir::Program p,usize entry,ref mut bytes::Buffer out){let named=names::empty();return render_named(p,ref named,entry,out);}
pub fn clean::Problem render_identified(ref identified::Program program,usize entry,ref mut bytes::Buffer out){var usize work=0;let named=switch(names::build(program,ref mut work)){case .ok(p):{p}case .err(e):{return e;}};return render_named(identified::ir_program(program),ref named,entry,out);}
fn clean::Problem render_named(ref ir::Program p,ref names::Table named,usize entry,ref mut bytes::Buffer out){
    let checked=proof::check(p);if(checked.code!=0){return checked;}if(entry==0 || entry>ir::count(p)){return fail(1001,0,0,0);}let f=clean::frame_at(ir::cleanup_ir(p),entry);if(scalar::scalar(p,f.result)!=1 || calls::parameters(ir::at(p,entry))!=0){return fail(1001,f.module_index,0,0);}
    let graphs=switch(flow::lower_all(p)){case .ok(g):{g}case .err(e):{return e;}};var usize work=0;if(!budget::charge(ref mut work,names::work(named)) || !names::matches(named,p,ref mut work)){return fail(if(work>=1048576){1002}else{1001},0,0,0);}var usize i=1;while(i<=v::len<cfg::Graph>(ref graphs)){if(!budget::charge(ref mut work,cfg::count(v::at<cfg::Graph>(ref graphs,i-1)))){return fail(1002,0,0,0);}i+=1;}
    let sdk_plan=switch(sdk::build(p,ref mut work)){case .ok(p):{p}case .err(e):{return e;}};
    let indexed=switch(contracts::build(p,ref mut work)){case .ok(index):{index}case .err(e):{return fail(e,0,0,0);}};
    var bytes::Buffer text=bytes::empty();var w::State s=w::State{.code=0};
    if(!prologue(p,ref mut text,ref mut s,ref mut work)){return fail(if(s.code==3){1003}else{1002},0,0,0);}
    let marks=switch(aggregate::definitions(p,ref mut text,ref mut s,ref mut work)){case .ok(m):{m}case .err(e):{return fail(if(e<1000){if(e==3){1003}else{1002}}else{e},0,0,0);}};
    if(!literals::definitions(p,ref mut text,ref mut s,ref mut work)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},0,0,0);}
    if(!w::text(ref mut text,ref mut s,b"#include <stddef.h>\n") || !sdk::definitions(p,ref sdk_plan,ref mut text,ref mut s,ref mut work) || !lifecycle::definitions(p,ref sdk_plan,ref mut text,ref mut s,ref mut work) || !calls::foreign_signatures(p,ref sdk_plan,ref indexed,ref mut text,ref mut s,ref mut work)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},0,0,0);}
    i=1;while(i<=ir::count(p)){if(!calls::signature(p,named,i,ref mut text,ref mut s,false) || !w::text(ref mut text,ref mut s,b";\n")){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},clean::frame_at(ir::cleanup_ir(p),i).module_index,ir::start(ir::at(p,i)),ir::end(ir::at(p,i)));}i+=1;}
    if(!aggregate::drops_named(p,named,ref marks,ref mut text,ref mut s,ref mut work)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},0,0,0);}
    let body_index=switch(targets::build(p,ref indexed,ref mut work)){case .ok(index):{index}case .err(e):{return fail(e,0,0,0);}};
    i=1;while(i<=ir::count(p)){var usize failed_start=ir::start(ir::at(p,i));var usize failed_end=ir::end(ir::at(p,i));if(!function(p,named,ref sdk_plan,ref indexed,ref body_index,v::at<cfg::Graph>(ref graphs,i-1),ref mut text,ref mut s,ref mut work,ref mut failed_start,ref mut failed_end,false)){return fail(if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code},clean::frame_at(ir::cleanup_ir(p),i).module_index,failed_start,failed_end);}i+=1;}
    if(!w::text(ref mut text,ref mut s,b"int32_t fy_candidate_entry(void){return ") || !names::function(named,entry,ref mut text,ref mut s) || !w::text(ref mut text,ref mut s,b"();}\n#ifndef FY_CANDIDATE_NO_MAIN\nint main(void){return (int)fy_candidate_entry();}\n#endif\n")){return fail(if(s.code==3){1003}else{1002},0,0,0);}
    if(bytes::len(out)>2097152 || bytes::len(ref text)>2097152-bytes::len(out)){return fail(1002,0,0,0);}return fail(if(bytes::append(out,bytes::view(ref text)).is_ok){0}else{1003},0,0,0);
}

resource Prepared {owner::Program program;names::Table named;v::Vec<cfg::Graph> graphs;sdk::Plan sdk_plan;contracts::Index indexed;targets::Index body_index;reachable::Plan selected;bytes::Buffer header;bytes::Buffer literal_source;usize entry;usize work;cleanup{}}
resource Unit {usize first;usize stop;bytes::Buffer text;cleanup{}}
pub resource Bundle {Prepared prepared;v::Vec<Unit> units;usize frames;usize work;usize retries;cleanup{}}
fn u32 error_code(w::State s){return if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code};}
fn Result<Prepared,clean::Problem> prepare_project(owner::Program p,names::Table named,usize entry,bool entry_only){
    let raw=owner::ir_program(ref p);
    let checked=proof::check(raw);if(checked.code!=0){return .err(checked);}if(!clean::project(ir::cleanup_ir(raw)) || entry==0 || entry>ir::count(raw)){return .err(fail(1001,0,0,0));}let f=clean::frame_at(ir::cleanup_ir(raw),entry);if(scalar::scalar(raw,f.result)!=1 || calls::parameters(ir::at(raw,entry))!=0){return .err(fail(1001,f.module_index,0,0));}
    let graphs=switch(flow::lower_all(raw)){case .ok(g):{g}case .err(e):{return .err(e);}};var usize work=0;var usize i=0;while(i<v::len<cfg::Graph>(ref graphs)){if(!budget::charge(ref mut work,cfg::count(v::at<cfg::Graph>(ref graphs,i)))){return .err(fail(1002,0,0,0));}i+=1;}
    let sdk_plan=switch(sdk::build(raw,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};let indexed=switch(contracts::build(raw,ref mut work)){case .ok(p):{p}case .err(e):{return .err(fail(e,0,0,0));}};
    let body_index=switch(targets::build(raw,ref indexed,ref mut work)){case .ok(p):{p}case .err(e):{return .err(fail(e,0,0,0));}};
    // Index/SDK preparation and the reachability fixed point each retain their
    // own work bound. Every completed history enters the whole-project ledger.
    var usize total=0;var usize name_work=0;if(!names::matches(ref named,raw,ref mut name_work)){return .err(fail(if(name_work>=1048576){1002}else{1001},0,0,0));}if(!limits::project_charge(ref mut total,names::work(ref named)+name_work)){return .err(fail(1002,0,0,0));}if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}work=0;
    let selected=switch(if(entry_only){reachable::build(raw,ref graphs,ref indexed,ref body_index,entry,ref mut work)}else{reachable::all(raw,ref mut work)}){case .ok(p):{p}case .err(e):{return .err(fail(e,0,0,0));}};
    if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}work=0;
    var bytes::Buffer header=bytes::empty();var w::State s=w::State{.code=0};if(!w::text(ref mut header,ref mut s,b"#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#define FY_CANDIDATE_PROJECT_HEADER_V1\n") || !prologue(raw,ref mut header,ref mut s,ref mut work)){return .err(fail(error_code(s),0,0,0));}
    let marks=switch(aggregate::definitions(raw,ref mut header,ref mut s,ref mut work)){case .ok(m):{m}case .err(e):{return .err(fail(if(e<1000){if(e==3){1003}else{1002}}else{e},0,0,0));}};
    if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}work=0;
    var bytes::Buffer literal_source=bytes::empty();if(!literals::shared_storage(raw,ref mut header,ref mut literal_source,ref mut s,ref mut work)){return .err(fail(error_code(s),0,0,0));}
    if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}work=0;
    if(!w::text(ref mut header,ref mut s,b"#include <stddef.h>\n") || !sdk::definitions(raw,ref sdk_plan,ref mut header,ref mut s,ref mut work) || !lifecycle::definitions(raw,ref sdk_plan,ref mut header,ref mut s,ref mut work) || !calls::foreign_signatures(raw,ref sdk_plan,ref indexed,ref mut header,ref mut s,ref mut work)){return .err(fail(error_code(s),0,0,0));}
    i=1;while(i<=ir::count(raw)){if(reachable::frame(ref selected,i) && (!calls::signature(raw,ref named,i,ref mut header,ref mut s,true) || !w::text(ref mut header,ref mut s,b";\n"))){return .err(fail(error_code(s),clean::frame_at(ir::cleanup_ir(raw),i).module_index,ir::start(ir::at(raw,i)),ir::end(ir::at(raw,i))));}i+=1;}
    if(!aggregate::drops_named(raw,ref named,if(entry_only){reachable::drops(ref selected)}else{ref marks},ref mut header,ref mut s,ref mut work) || !w::text(ref mut header,ref mut s,b"#endif\n")){return .err(fail(error_code(s),0,0,0));}
    if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}
    return .ok(Prepared{.program=p,.named,.graphs,.sdk_plan,.indexed,.body_index,.selected,.header,.literal_source,.entry,.work=total});
}
fn Result<bytes::Buffer,clean::Problem> project_unit(ref Prepared p,usize first,usize stop,ref mut usize work){
    let raw=owner::ir_program(ref p.program);
    var bytes::Buffer text=bytes::empty();var w::State s=w::State{.code=0};if(!w::text(ref mut text,ref mut s,b"/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\n")){return .err(fail(error_code(s),0,0,0));}var usize i=first;
    while(i<stop){if(reachable::frame(ref p.selected,i)){var usize start=ir::start(ir::at(raw,i));var usize end=ir::end(ir::at(raw,i));if(!function(raw,ref p.named,ref p.sdk_plan,ref p.indexed,ref p.body_index,v::at<cfg::Graph>(ref p.graphs,i-1),ref mut text,ref mut s,work,ref mut start,ref mut end,true)){return .err(fail(error_code(s),clean::frame_at(ir::cleanup_ir(raw),i).module_index,start,end));}}i+=1;}
    if(first<=p.entry && p.entry<stop && (!w::text(ref mut text,ref mut s,b"int32_t fy_candidate_entry(void){return ") || !names::function(ref p.named,p.entry,ref mut text,ref mut s) || !w::text(ref mut text,ref mut s,b"();}\n#ifndef FY_CANDIDATE_NO_MAIN\nint main(void){return (int)fy_candidate_entry();}\n#endif\n"))){return .err(fail(error_code(s),0,0,0));}return .ok(text);
}
fn clean::Problem append_project_units(ref Prepared p,usize first,usize stop,usize depth,ref mut v::Vec<Unit> units,ref mut usize total,ref mut usize retries){
    var usize work=0;switch(project_unit(p,first,stop,ref mut work)){
        case .ok(text):{if(!limits::project_charge(total,work)){return fail(1002,0,0,0);}if(v::len<Unit>(units)>=limits::project_frames()){return fail(1002,0,0,0);}if(!v::push<Unit>(units,Unit{.first,.stop,.text}).is_ok){return fail(1003,0,0,0);}return fail(0,0,0,0);}
        case .err(e):{if(!limits::project_charge(total,work)){return fail(1002,0,0,0);}if(e.code!=1002 || stop-first<=1 || depth>=12){return e;};*retries=*retries+1;let middle=first+(stop-first)/2;let left=append_project_units(p,first,middle,depth+1,units,total,retries);if(left.code!=0){return left;}return append_project_units(p,middle,stop,depth+1,units,total,retries);}
    };return fail(1001,0,0,0);
}
// A complete project owns its validated IR and never exposes a successful prefix.
// Analysis, common declarations and each body attempt retain 1 MiB work;
// output buffers retain 2 MiB. All histories share limits::project_work().
fn Result<Bundle,clean::Problem> partition_scoped(owner::Program program,names::Table named,usize entry,usize max_frames,bool entry_only){
    if(max_frames==0 || max_frames>limits::project_frames()){return .err(fail(1001,0,0,0));}let p=switch(prepare_project(program,named,entry,entry_only)){case .ok(p):{p}case .err(e):{return .err(e);}};let frames=ir::count(owner::ir_program(ref p.program));var usize total=p.work;var usize retries=0;var v::Vec<Unit> units=v::empty<Unit>();var usize first=1;
    while(first<=frames){let stop=if(max_frames<frames+1-first){first+max_frames}else{frames+1};let e=append_project_units(ref p,first,stop,0,ref mut units,ref mut total,ref mut retries);if(e.code!=0){return .err(e);}first=stop;}
    let usize count=v::len<Unit>(ref units);var usize index=0;var usize next=1;while(index<count){let unit=v::at<Unit>(ref units,index);if(unit.first!=next || unit.first>=unit.stop){return .err(fail(1001,0,0,0));}next=unit.stop;index+=1;}if(next!=frames+1){return .err(fail(1001,0,0,0));}
    return .ok(Bundle{.prepared=p,.units,.frames,.work=total,.retries});
}
fn Result<Bundle,clean::Problem> partition_plain(ir::Program program,usize entry,usize max_frames,bool entry_only){if(max_frames==0 || max_frames>limits::project_frames()){return .err(fail(1001,0,0,0));}let owned=switch(owner::plain(program)){case .ok(p):{p}case .err(e):{return .err(e);}};return partition_scoped(owned,names::empty(),entry,max_frames,entry_only);}
pub fn Result<Bundle,clean::Problem> partition(ir::Program program,usize entry,usize max_frames){return partition_plain(program,entry,max_frames,false);}
pub fn Result<Bundle,clean::Problem> partition_entry(ir::Program program,usize entry,usize max_frames){return partition_plain(program,entry,max_frames,true);}
fn Result<Bundle,clean::Problem> partition_named(project::Program program,usize entry,usize max_frames,bool entry_only){if(max_frames==0 || max_frames>limits::project_frames()){return .err(fail(1001,0,0,0));}var usize work=0;let named=switch(names::build_project(ref program,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};let owned=switch(owner::named(program)){case .ok(p):{p}case .err(e):{return .err(e);}};return partition_scoped(owned,named,entry,max_frames,entry_only);}
pub fn Result<Bundle,clean::Problem> partition_project(project::Program program,usize entry,usize max_frames){return partition_named(program,entry,max_frames,false);}
pub fn Result<Bundle,clean::Problem> partition_project_entry(project::Program program,usize entry,usize max_frames){return partition_named(program,entry,max_frames,true);}
// Keep the checked IR and its complete detached interfaces in the same owner.
// Name preparation borrows that IR before ownership passes to the C bundle.
fn Result<Bundle,clean::Problem> partition_complete(compiler::Complete program,usize entry,usize max_frames,bool entry_only){if(max_frames==0 || max_frames>limits::project_frames()){return .err(fail(1001,0,0,0));}var usize work=0;let named=switch(names::build_project(compiler::checked_program(ref program),ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};let owned=switch(owner::complete(program)){case .ok(p):{p}case .err(e):{return .err(e);}};return partition_scoped(owned,named,entry,max_frames,entry_only);}
pub fn Result<Bundle,clean::Problem> partition_complete_project(compiler::Complete program,usize entry,usize max_frames){return partition_complete(program,entry,max_frames,false);}
pub fn Result<Bundle,clean::Problem> partition_complete_project_entry(compiler::Complete program,usize entry,usize max_frames){return partition_complete(program,entry,max_frames,true);}
pub fn bool project_has_complete(ref Bundle p){return owner::has_complete(ref p.prepared.program);}
pub fn ref compiler::Complete project_complete(ref Bundle p){return owner::complete_project(ref p.prepared.program);}
pub fn ref u8[] project_header(ref Bundle p){return bytes::view(ref p.prepared.header);}
pub fn ref u8[] project_literals(ref Bundle p){return bytes::view(ref p.prepared.literal_source);}
pub fn usize project_unit_count(ref Bundle p){return v::len<Unit>(ref p.units);}
pub fn usize project_frames(ref Bundle p){return p.frames;}
pub fn usize project_selected_frames(ref Bundle p){return reachable::count(ref p.prepared.selected);}
pub fn usize project_work(ref Bundle p){return p.work;}
pub fn usize project_retries(ref Bundle p){return p.retries;}
pub fn usize project_first(ref Bundle p,usize index){return v::at<Unit>(ref p.units,index).first;}
pub fn usize project_stop(ref Bundle p,usize index){return v::at<Unit>(ref p.units,index).stop;}
pub fn ref u8[] project_source(ref Bundle p,usize index){return bytes::view(ref v::at<Unit>(ref p.units,index).text);}
