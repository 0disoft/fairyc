module module_unit_body_ir_model;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_value_model as values;
import module_unit_interfaces as interfaces;
import module_unit_region_model as regions;
import module_unit_body_ir_verify as proof;
import module_unit_sdk_sources as sdk_sources;
import syntax as syntax;
import module_state_layout as state;
import std::bytes as bytes;
import std::vec as v;

pub struct Node {values::Expression value;syntax::Expression syntax;u32 literal_kind;usize text_start;usize text_end;usize cooked_start;usize cooked_end;}
pub struct Statement {usize source_index;syntax::Statement value;usize operator_start;usize operator_end;}
pub struct Layout {interfaces::Shape value;usize name_start;usize name_end;}
pub resource Body {
    usize body;usize source_bytes;usize start;usize end;
    v::Vec<Node> nodes;v::Vec<Statement> statements;v::Vec<values::Local> locals;
    v::Vec<values::Invocation> calls;v::Vec<values::Argument> arguments;
    v::Vec<regions::Scope> regions;v::Vec<regions::Placement> placements;v::Vec<regions::Exit> resets;bytes::Buffer text;
    cleanup{}
}
pub resource Program {
    state::Plan states;
    sdk_sources::Sources sdk_sources;
    clean_ir::Program cleanup_ir;v::Vec<Body> bodies;v::Vec<Layout> layouts;
    v::Vec<interfaces::Field> fields;v::Vec<interfaces::Callable> callables;
    v::Vec<usize> parameters;bytes::Buffer names;usize work;cleanup{}
}
// Private checked body/cleanup/layout model compatibility, separate from the
// Go MIR codec and from any future independently versioned artifact envelope.
pub fn ref u8[] cache_revision(){return b"fairyc.candidate-owned-body-ir/0";}
pub fn ref clean_ir::Program cleanup_ir(ref Program p){return ref p.cleanup_ir;}
pub fn ref state::Plan state_layouts(ref Program p){return ref p.states;}
pub fn ref sdk_sources::Sources sdk_units(ref Program p){return ref p.sdk_sources;}
pub fn usize count(ref Program p){return v::len<Body>(ref p.bodies);}
pub fn ref Body at(ref Program p,usize frame){assert(frame!=0 && frame<=count(p));return v::at<Body>(ref p.bodies,frame-1);}
pub fn usize body(ref Body p){return p.body;}
pub fn usize source_bytes(ref Body p){return p.source_bytes;}
pub fn usize start(ref Body p){return p.start;}
pub fn usize end(ref Body p){return p.end;}
pub fn usize node_count(ref Body p){return v::len<Node>(ref p.nodes);}
pub fn Node node_at(ref Body p,usize index){return *v::at<Node>(ref p.nodes,index);}
pub fn ref u8[] node_text(ref Body p,usize index){let n=node_at(p,index);return bytes::view(ref p.text)[n.text_start..n.text_end];}
pub fn ref u8[] node_cooked(ref Body p,usize index){let n=node_at(p,index);return bytes::view(ref p.text)[n.cooked_start..n.cooked_end];}
pub fn usize text_bytes(ref Body p){return bytes::len(ref p.text);}
pub fn ref u8[] text_view(ref Body p){return bytes::view(ref p.text);}
pub fn usize name_bytes(ref Program p){return bytes::len(ref p.names);}
pub fn ref u8[] names_view(ref Program p){return bytes::view(ref p.names);}
pub fn usize statement_count(ref Body p){return v::len<Statement>(ref p.statements);}
pub fn Statement statement_at(ref Body p,usize index){return *v::at<Statement>(ref p.statements,index);}
pub fn ref u8[] assignment_operator(ref Body p,usize index){let s=statement_at(p,index);return bytes::view(ref p.text)[s.operator_start..s.operator_end];}
pub fn usize local_count(ref Body p){return v::len<values::Local>(ref p.locals);}
pub fn values::Local local_at(ref Body p,usize id){assert(id!=0 && id<=local_count(p));return *v::at<values::Local>(ref p.locals,id-1);}
pub fn usize call_count(ref Body p){return v::len<values::Invocation>(ref p.calls);}
pub fn values::Invocation call_at(ref Body p,usize index){return *v::at<values::Invocation>(ref p.calls,index);}
pub fn usize argument_count(ref Body p){return v::len<values::Argument>(ref p.arguments);}
pub fn values::Argument argument_at(ref Body p,usize index){return *v::at<values::Argument>(ref p.arguments,index);}
pub fn usize region_count(ref Body p){return v::len<regions::Scope>(ref p.regions);}
pub fn regions::Scope region_at(ref Body p,usize index){return *v::at<regions::Scope>(ref p.regions,index);}
pub fn usize placement_count(ref Body p){return v::len<regions::Placement>(ref p.placements);}
pub fn regions::Placement placement_at(ref Body p,usize index){return *v::at<regions::Placement>(ref p.placements,index);}
pub fn usize reset_count(ref Body p){return v::len<regions::Exit>(ref p.resets);}
pub fn regions::Exit reset_at(ref Body p,usize index){return *v::at<regions::Exit>(ref p.resets,index);}
pub fn usize layout_count(ref Program p){return v::len<Layout>(ref p.layouts);}
pub fn Layout layout_at(ref Program p,usize index){return *v::at<Layout>(ref p.layouts,index);}
pub fn ref u8[] layout_name(ref Program p,usize index){let s=layout_at(p,index);return bytes::view(ref p.names)[s.name_start..s.name_end];}
pub fn usize field_count(ref Program p){return v::len<interfaces::Field>(ref p.fields);}
pub fn interfaces::Field field_at(ref Program p,usize index){return *v::at<interfaces::Field>(ref p.fields,index);}
pub fn ref u8[] field_name(ref Program p,usize index){let s=field_at(p,index);return bytes::view(ref p.names)[s.name_start..s.name_end];}
pub fn usize callable_count(ref Program p){return v::len<interfaces::Callable>(ref p.callables);}
pub fn interfaces::Callable callable_at(ref Program p,usize index){return *v::at<interfaces::Callable>(ref p.callables,index);}
pub fn ref u8[] linker_name(ref Program p,usize index){let s=callable_at(p,index);return bytes::view(ref p.names)[s.link_start..s.link_end];}
pub fn usize parameter_count(ref Program p){return v::len<usize>(ref p.parameters);}
pub fn usize parameter_at(ref Program p,usize index){return *v::at<usize>(ref p.parameters,index);}
pub fn usize work(ref Program p){return p.work;}
pub fn Body assemble_body_regions(usize body,usize source_bytes,usize start,usize end,v::Vec<Node> nodes,v::Vec<Statement> statements,v::Vec<values::Local> locals,v::Vec<values::Invocation> calls,v::Vec<values::Argument> arguments,v::Vec<regions::Scope> regions,v::Vec<regions::Placement> placements,v::Vec<regions::Exit> resets,bytes::Buffer text){return Body{.body,.source_bytes,.start,.end,.nodes,.statements,.locals,.calls,.arguments,.regions,.placements,.resets,.text};}
pub fn Body assemble_body(usize body,usize source_bytes,usize start,usize end,v::Vec<Node> nodes,v::Vec<Statement> statements,v::Vec<values::Local> locals,v::Vec<values::Invocation> calls,v::Vec<values::Argument> arguments,v::Vec<regions::Scope> regions,v::Vec<regions::Exit> resets,bytes::Buffer text){return assemble_body_regions(body,source_bytes,start,end,nodes,statements,locals,calls,arguments,regions,v::empty<regions::Placement>(),resets,text);}
pub fn Result<Program,clean_ir::Problem> assemble_sdk(sdk_sources::Sources sdk_sources,clean_ir::Program cleanup_ir,v::Vec<Body> bodies,v::Vec<Layout> layouts,v::Vec<interfaces::Field> fields,v::Vec<interfaces::Callable> callables,v::Vec<usize> parameters,bytes::Buffer names,usize work){
    return assemble_sdk_states(state::empty(),sdk_sources,cleanup_ir,bodies,layouts,fields,callables,parameters,names,work);
}
pub fn Result<Program,clean_ir::Problem> assemble_sdk_states(state::Plan states,sdk_sources::Sources sdk_sources,clean_ir::Program cleanup_ir,v::Vec<Body> bodies,v::Vec<Layout> layouts,v::Vec<interfaces::Field> fields,v::Vec<interfaces::Callable> callables,v::Vec<usize> parameters,bytes::Buffer names,usize work){
    let p=Program{.states,.sdk_sources,.cleanup_ir,.bodies,.layouts,.fields,.callables,.parameters,.names,.work};let checked=proof::check(ref p);if(checked.code!=0){return .err(checked);}return .ok(p);
}
pub fn Result<Program,clean_ir::Problem> assemble(clean_ir::Program cleanup_ir,v::Vec<Body> bodies,v::Vec<Layout> layouts,v::Vec<interfaces::Field> fields,v::Vec<interfaces::Callable> callables,v::Vec<usize> parameters,bytes::Buffer names,usize work){
    return assemble_sdk(sdk_sources::empty(),cleanup_ir,bodies,layouts,fields,callables,parameters,names,work);
}
