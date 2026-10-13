module module_project_cleanup;
import candidate_limits as limits;
import module_project_inputs as inputs_view;
import module_project_identified_compile as named_units;
import module_project_types as types;
import module_project_frames as functions;
import module_project_metadata as metadata;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index){return clean::Problem{.code,.module_index,.start=0,.end=0};}
fn bool chosen(ref functions::Plan frames,usize unit,usize frame){let id=functions::mapped(frames,unit,frame);if(id==0){return false;}let origin=functions::origin(frames,id);return origin.unit==unit && origin.frame==frame;}
resource SourceLayouts {v::Vec<bool> present;cleanup{}}
// Original type IDs stay local. Prepare presence once per source unit; recipe
// equality still compares every selected frame and rebound field below.
fn Result<SourceLayouts,clean::Problem> source_layouts(ref ir::Program p,ref mut usize work){
    let count=clean::type_count(ir::cleanup_ir(p));var v::Vec<bool> present=v::empty<bool>();var usize i=0;
    while(i<=count){if(!limits::project_charge(work,1)){return .err(fail(1002,0));}if(!v::push<bool>(ref mut present,false).is_ok){return .err(fail(1003,0));}i+=1;}
    i=0;while(i<ir::layout_count(p)){if(!limits::project_charge(work,1)){return .err(fail(1002,0));}let type=ir::layout_at(p,i).value.type;if(type==0 || type>count){return .err(fail(1001,0));}let slot=v::at_mut<bool>(ref mut present,type);*slot=true;i+=1;}
    return .ok(SourceLayouts{.present});
}
fn clean::Problem recipes(ref v::Vec<ir::Program> inputs,ref named_units::Plan identified_inputs,ref types::Plan graph,ref functions::Plan frames,ref metadata::Plan layouts,ref mut usize work){var usize unit=0;
    while(unit<inputs_view::count(inputs,identified_inputs)){let p=if(named_units::count(identified_inputs)==0){v::at<ir::Program>(inputs,unit)}else{inputs_view::source_at(identified_inputs,unit)};let c=ir::cleanup_ir(p);let present=switch(source_layouts(p,work)){case .ok(p):{p}case .err(e):{return e;}};var usize type=1;while(type<=clean::type_count(c)){if(!limits::project_charge(work,1)){return fail(1002,0);}let id=types::mapped(graph,unit,type);let choice=metadata::choice(layouts,id);let current=clean::glue_at(c,type);
        if(current.kind==1 && choice.type!=0){if(*v::at<bool>(ref present.present,type)){let selected=ir::cleanup_ir(if(named_units::count(identified_inputs)==0){v::at<ir::Program>(inputs,choice.unit)}else{inputs_view::source_at(identified_inputs,choice.unit)});let old=clean::glue_at(selected,choice.type);if(current.kind!=old.kind || current.nonempty!=old.nonempty || current.effects!=old.effects || current.abilities!=old.abilities || current.field_count!=old.field_count || functions::mapped(frames,unit,current.frame)!=functions::mapped(frames,choice.unit,old.frame)){return fail(1001,clean::type_at(c,type).module_index);}var usize i=0;while(i<current.field_count){if(!limits::project_charge(work,1)){return fail(1002,0);}let a=clean::field_at(c,current.field_start+i);let b=clean::field_at(selected,old.field_start+i);if(a.ordinal!=b.ordinal || types::mapped(graph,unit,a.owner)!=types::mapped(graph,choice.unit,b.owner) || types::mapped(graph,unit,a.type)!=types::mapped(graph,choice.unit,b.type)){return fail(1001,clean::type_at(c,type).module_index);}i+=1;}}
        }type+=1;
    }unit+=1;}return fail(0,0);
}
pub fn Result<clean::Program,clean::Problem> build(ref v::Vec<ir::Program> inputs,ref types::Plan graph,ref functions::Plan functions,ref metadata::Plan layouts,ref mut usize work){let empty=named_units::empty();return build_inputs(inputs,ref empty,graph,functions,layouts,work);}
pub fn Result<clean::Program,clean::Problem> build_inputs(ref v::Vec<ir::Program> inputs,ref named_units::Plan identified_inputs,ref types::Plan graph,ref functions::Plan functions,ref metadata::Plan layouts,ref mut usize work){
    let checked=recipes(inputs,identified_inputs,graph,functions,layouts,work);if(checked.code!=0){return .err(checked);}let descriptors=switch(types::copy_types(graph)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var v::Vec<clean::Frame> frames=v::empty<clean::Frame>();var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();var v::Vec<clean::Field> fields=v::empty<clean::Field>();var v::Vec<clean::Operand> operands=v::empty<clean::Operand>();var v::Vec<clean::Drop> drops=v::empty<clean::Drop>();var usize i=1;
    while(i<=functions::count(functions)){if(!limits::project_charge(work,1)){return .err(fail(1002,0));}if(!v::push<clean::Frame>(ref mut frames,functions::at(functions,i)).is_ok){return .err(fail(1003,0));}i+=1;}
    i=1;while(i<=types::type_count(graph)){if(!limits::project_charge(work,1)){return .err(fail(1002,0));}let t=types::type_at(graph,i);var clean::Glue g=clean::Glue{.kind=if(t.kind==3 || t.kind==13){1}else if(t.kind==5){2}else if(t.kind==6){3}else{0},.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0};let choice=metadata::choice(layouts,i);
        if(g.kind==1 && choice.type!=0){let source=ir::cleanup_ir(if(named_units::count(identified_inputs)==0){v::at<ir::Program>(inputs,choice.unit)}else{inputs_view::source_at(identified_inputs,choice.unit)});g=clean::glue_at(source,choice.type);g.frame=functions::mapped(functions,choice.unit,g.frame);let start=g.field_start;g.field_start=v::len<clean::Field>(ref fields);if(g.field_count>limits::project_fields()-g.field_start){return .err(fail(1002,t.module_index));}var usize j=0;while(j<g.field_count){if(!limits::project_charge(work,1)){return .err(fail(1002,t.module_index));}var f=clean::field_at(source,start+j);f.owner=i;f.type=types::mapped(graph,choice.unit,f.type);if(!v::push<clean::Field>(ref mut fields,f).is_ok){return .err(fail(1003,t.module_index));}j+=1;}}
        if(!v::push<clean::Glue>(ref mut glues,g).is_ok){return .err(fail(1003,t.module_index));}i+=1;
    }
    var usize unit=0;while(unit<inputs_view::count(inputs,identified_inputs)){let source=ir::cleanup_ir(if(named_units::count(identified_inputs)==0){v::at<ir::Program>(inputs,unit)}else{inputs_view::source_at(identified_inputs,unit)});var v::Vec<usize> maps=v::empty<usize>();i=1;
        while(i<=clean::operand_count(source)){if(!limits::project_charge(work,1)){return .err(fail(1002,0));}var o=clean::operand_at(source,i);var usize id=0;if(chosen(functions,unit,o.frame)){if(v::len<clean::Operand>(ref operands)>=16384){return .err(fail(1002,0));}o.frame=functions::mapped(functions,unit,o.frame);o.type=types::mapped(graph,unit,o.type);if(!v::push<clean::Operand>(ref mut operands,o).is_ok){return .err(fail(1003,0));}id=v::len<clean::Operand>(ref operands);}if(!v::push<usize>(ref mut maps,id).is_ok){return .err(fail(1003,0));}i+=1;}
        i=0;while(i<clean::drop_count(source)){if(!limits::project_charge(work,1)){return .err(fail(1002,0));}var d=clean::drop_at(source,i);d.operand=*v::at<usize>(ref maps,d.operand-1);if(d.operand!=0){if(v::len<clean::Drop>(ref drops)>=16384){return .err(fail(1002,0));}if(!v::push<clean::Drop>(ref mut drops,d).is_ok){return .err(fail(1003,0));}}i+=1;}unit+=1;
    }return clean::assemble_project(descriptors,frames,glues,fields,operands,drops,*work);
}
