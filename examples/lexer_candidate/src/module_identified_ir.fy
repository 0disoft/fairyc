module module_identified_ir;
import module_source_specializations as selection;
import generic_source_plan as source;
import generic_source_plan_input as input;
import generic_source_catalog as catalog;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_interfaces as typed;
import module_signatures as signatures;
import module_unit_c_callable_index as callables;
import module_unit_c_frame_index as targets;
import module_public_interface_model as records;
import project_path as budget;
import syntax as syntax;
import std::vec as v;

pub struct Problem{u32 code;u32 kind;usize id;usize module_index;usize start;usize end;}
pub resource Program{ir::Program program;selection::Selection selection;v::Vec<selection::Row> frames;targets::Index targets;cleanup{}}
pub fn ref ir::Program ir_program(ref Program p){return ref p.program;}
pub fn usize frame_count(ref Program p){return v::len<selection::Row>(ref p.frames);}
pub fn selection::Row frame_at(ref Program p,usize id){assert(id>0 && id<=frame_count(p));return *v::at<selection::Row>(ref p.frames,id-1);}
fn selection::Row empty(){return selection::Row{.node=0,.id=records::Text{.start=0,.end=0}};}
pub fn selection::Row type_at(ref Program p,usize id){assert(id>0 && id<=clean::type_count(ir::cleanup_ir(ref p.program)));return if(id<=selection::type_count(ref p.selection)){selection::type_at(ref p.selection,id)}else{empty()};}
pub fn ref u8[] identity(ref Program p,selection::Row row){return selection::identity(ref p.selection,row);}
fn Problem fail(u32 code,u32 kind,usize id,usize module_index,usize start,usize end){return Problem{.code,.kind,.id,.module_index,.start,.end};}
// The canonical graph is a prefix: value typing may append ordinary descriptors,
// but an unselected late Vec cannot silently acquire an empty source identity.
pub fn Result<Program,Problem> build(ir::Program program,ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref mut usize work){
    let selected=switch(selection::build(plan,sources,metadata,graph,work)){case .ok(p):{p}case .err(e):{return .err(fail(e.code,e.kind,e.id,0,0,0));}};let cleaned=ir::cleanup_ir(ref program);
    if(clean::type_count(cleaned)<typed::type_count(graph) || clean::frame_count(cleaned)!=ir::count(ref program)){return .err(fail(9,0,0,0,0,0));}var usize i=1;
    while(i<=clean::type_count(cleaned)){if(!budget::charge(work,1)){return .err(fail(10,2,i,0,0,0));}let t=clean::type_at(cleaned,i);if(i<=typed::type_count(graph)){let original=typed::type_at(graph,i);if(t.kind!=original.kind || t.scalar!=original.scalar || t.module_index!=original.module_index || t.declaration!=original.declaration || t.first!=original.first || t.second!=original.second){return .err(fail(12,2,i,t.module_index,0,0));}}else if(t.kind==13){return .err(fail(9,2,i,t.module_index,0,0));}i+=1;}
    var v::Vec<selection::Row> frames=v::empty<selection::Row>();i=1;
    while(i<=ir::count(ref program)){if(!budget::charge(work,1)){return .err(fail(10,4,i,0,0,0));}let frame=clean::frame_at(cleaned,i);if(frame.module_index>=v::len<syntax::Parsed>(sources) || frame.declaration==0 || frame.declaration>syntax::declaration_count(v::at<syntax::Parsed>(sources,frame.module_index))){return .err(fail(9,4,i,frame.module_index,0,0));}let d=syntax::declaration_at(v::at<syntax::Parsed>(sources,frame.module_index),frame.declaration-1);var selection::Row row=empty();
        if(frame.kind==1){if(frame.context==0 || frame.context>selection::function_count(ref selected)){return .err(fail(9,4,i,frame.module_index,d.start,d.end));}row=selection::function_at(ref selected,frame.context);}
        else if(frame.kind==2){if(frame.context==0 || frame.context>selection::cleanup_count(ref selected)){return .err(fail(9,4,i,frame.module_index,d.start,d.end));}row=selection::cleanup_at(ref selected,frame.context);if(frame.receiver_type==0 || frame.receiver_type>selection::type_count(ref selected) || row.node!=selection::type_at(ref selected,frame.receiver_type).node){return .err(fail(9,4,i,frame.module_index,d.start,d.end));}}
        else if(frame.kind!=3){return .err(fail(9,4,i,frame.module_index,d.start,d.end));}
        if((row.node==0)!=(d.generic_parameters==0)){return .err(fail(9,4,i,frame.module_index,d.start,d.end));}if(row.node!=0){let node=source::at(plan,row.node-1);let original=catalog::template_at(input::templates(source::input(plan)),node.template_index);if(original.module_index!=frame.module_index || original.declaration+1!=frame.declaration){return .err(fail(9,4,i,frame.module_index,d.start,d.end));}}
        if(!v::push<selection::Row>(ref mut frames,row).is_ok){return .err(fail(11,4,i,frame.module_index,d.start,d.end));}i+=1;
    }
    var usize measured=0;let contracts=switch(callables::build(ref program,ref mut measured)){case .ok(p):{p}case .err(code):{discard budget::charge(work,measured);return .err(fail(code,0,0,0,0,0));}};let indexed=switch(targets::build(ref program,ref contracts,ref mut measured)){case .ok(p):{p}case .err(code):{discard budget::charge(work,measured);return .err(fail(code,0,0,0,0,0));}};if(!budget::charge(work,measured)){return .err(fail(10,0,0,0,0,0));}
    return .ok(Program{.program,.selection=selected,.frames,.targets=indexed});
}
// Foreign/builtin calls have no owned frame. Owned ordinary calls yield a zero
// identity row; specialized calls use the exact target frame's source identity.
pub fn Result<usize,u32> call_frame(ref Program p,usize frame,usize at,ref mut usize work){assert(frame>0 && frame<=frame_count(p));let body=ir::at(ref p.program,frame);assert(at<ir::call_count(body));let call=ir::call_at(body,at);if(!call.checked){return .err(9);}if(call.kind!=1 && call.kind!=2){return .ok(0);}var usize measured=0;var u32 code=0;let target=targets::find(ref p.program,ref p.targets,call.target_module,call.target_declaration,call.instance,call.kind==2,ref mut measured,ref mut code);if(!budget::charge(work,measured)){return .err(10);}if(code!=0){return .err(code);}if(call.kind==2 && (target==0 || frame_at(p,target).node==0)){return .err(9);}return .ok(target);}
