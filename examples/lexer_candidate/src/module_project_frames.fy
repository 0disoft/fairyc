module module_project_frames;
import module_project_types as types;
import candidate_limits as limits;
import module_project_inputs as inputs_view;
import module_project_identified_compile as named_units;
import module_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import std::vec as v;
import std::bytes as bytes;

pub struct Origin {usize unit;usize frame;}
pub struct Range {usize start;usize count;}
pub resource Plan {v::Vec<clean::Frame> frames;v::Vec<Origin> origins;v::Vec<Range> units;v::Vec<usize> maps;usize work;cleanup{}}
fn clean::Problem fail(u32 code,usize module_index){return clean::Problem{.code,.module_index,.start=0,.end=0};}
fn Result<bool,clean::Problem> ordinary(ref ir::Program p,clean::Frame f,ref mut usize work){if(f.kind==3){return .ok(true);}var usize i=0;while(i<ir::callable_count(p)){if(!budget::charge(work,1)){return .err(fail(1002,f.module_index));}let c=ir::callable_at(p,i);if(c.module_index==f.module_index && c.declaration==f.declaration){return .ok(!c.template && !c.foreign);}i+=1;}return .ok(false);}
fn ref u8[] identity(ref named_units::Plan inputs,usize unit,usize frame){let p=named_units::at(inputs,unit);return identified::identity(p,identified::frame_at(p,frame));}
fn Result<usize,clean::Problem> specialization_slot(ref named_units::Plan inputs,usize unit,usize frame,ref v::Vec<Origin> origins,ref v::Vec<usize> table,ref mut usize work){
    let id=identity(inputs,unit,frame);if(id.len!=67 || !bytes::equal(id[..3],b"g0_")){return .err(fail(1001,0));}if(!budget::charge(work,id.len+1)){return .err(fail(1002,0));}var usize slot=0;var usize i=3;
    while(i<id.len){let digit=id[i];if(!(digit>=48 && digit<=57 || digit>=97 && digit<=102)){return .err(fail(1001,0));}slot=(slot*33+(digit as usize))&8191;i+=1;}
    while(true){if(!budget::charge(work,1)){return .err(fail(1002,0));}let old=*v::at<usize>(table,slot);if(old==0){return .ok(slot);}let origin=*v::at<Origin>(origins,old-1);if(!budget::charge(work,id.len)){return .err(fail(1002,0));}if(bytes::equal(id,identity(inputs,origin.unit,origin.frame))){return .ok(slot);}slot=(slot+1)&8191;}
    return .err(fail(1001,0));
}
pub fn Result<Plan,clean::Problem> build(ref v::Vec<ir::Program> inputs,ref types::Plan graph){let empty=named_units::empty();return build_inputs(inputs,ref empty,graph);}
pub fn Result<Plan,clean::Problem> build_inputs(ref v::Vec<ir::Program> inputs,ref named_units::Plan identified_inputs,ref types::Plan graph){
    if(inputs_view::count(inputs,identified_inputs)!=types::unit_count(graph)){return .err(fail(1001,0));}var v::Vec<clean::Frame> frames=v::empty<clean::Frame>();var v::Vec<Origin> origins=v::empty<Origin>();var v::Vec<Range> units=v::empty<Range>();var v::Vec<usize> maps=v::empty<usize>();var usize work=0;var usize unit=0;
    // Ordinary bodies are unique by source coordinate. Cleanup aliases are
    // unique by rebound receiver type. Identified generic frames share source IDs;
    // ordinary inputs preserve their existing per-unit generic namespaces.
    var v::Vec<usize> specializations=v::empty<usize>();var v::Vec<usize> declarations=v::empty<usize>();var v::Vec<usize> receivers=v::empty<usize>();var usize slot=0;
    while(slot<8192){if(!budget::charge(ref mut work,1)){return .err(fail(1002,0));}if(!v::push<usize>(ref mut declarations,0).is_ok){return .err(fail(1003,0));}slot+=1;}
    if(named_units::count(identified_inputs)!=0){slot=0;while(slot<8192){if(!budget::charge(ref mut work,1)){return .err(fail(1002,0));}if(!v::push<usize>(ref mut specializations,0).is_ok){return .err(fail(1003,0));}slot+=1;}}
    slot=0;while(slot<=types::type_count(graph)){if(!budget::charge(ref mut work,1)){return .err(fail(1002,0));}if(!v::push<usize>(ref mut receivers,0).is_ok){return .err(fail(1003,0));}slot+=1;}
    while(unit<inputs_view::count(inputs,identified_inputs)){let p=if(named_units::count(identified_inputs)==0){v::at<ir::Program>(inputs,unit)}else{inputs_view::source_at(identified_inputs,unit)};let usize start=v::len<usize>(ref maps);var usize i=1;while(i<=ir::count(p)){if(!budget::charge(ref mut work,1)){return .err(fail(1002,0));}var f=clean::frame_at(ir::cleanup_ir(p),i);f.result=types::mapped(graph,unit,f.result);f.receiver_type=types::mapped(graph,unit,f.receiver_type);if(f.result==0){return .err(fail(1001,f.module_index));}var usize id=0;var bool unique=false;var usize bucket=0;
        if(f.kind==2){if(f.receiver_type==0 || f.receiver_type>types::type_count(graph)){return .err(fail(1001,f.module_index));}id=*v::at<usize>(ref receivers,f.receiver_type);if(id!=0){let old=*v::at<clean::Frame>(ref frames,id-1);if(f.module_index!=old.module_index || f.declaration!=old.declaration || f.result!=old.result || f.effects!=old.effects || f.receiver!=old.receiver){return .err(fail(1001,f.module_index));}}}
        else{unique=switch(ordinary(p,f,ref mut work)){case .ok(b):{b}case .err(e):{return .err(e);}};if(unique){bucket=(f.module_index*104729 ^ f.declaration*3266489917)&8191;while(*v::at<usize>(ref declarations,bucket)!=0){if(!budget::charge(ref mut work,1)){return .err(fail(1002,f.module_index));}let old=*v::at<clean::Frame>(ref frames,*v::at<usize>(ref declarations,bucket)-1);if(f.module_index==old.module_index && f.declaration==old.declaration){return .err(fail(1001,f.module_index));}bucket=(bucket+1)&8191;}}}
        if(f.kind==1 && !unique && named_units::count(identified_inputs)!=0){bucket=switch(specialization_slot(identified_inputs,unit,i,ref origins,ref specializations,ref mut work)){case .ok(n):{n}case .err(e):{return .err(e);}};id=*v::at<usize>(ref specializations,bucket);if(id!=0){let old=*v::at<clean::Frame>(ref frames,id-1);if(f.kind!=old.kind || f.module_index!=old.module_index || f.declaration!=old.declaration || f.result!=old.result || f.effects!=old.effects || f.receiver!=old.receiver || f.receiver_type!=old.receiver_type){return .err(fail(1001,f.module_index));}}}
        if(id==0){if(v::len<clean::Frame>(ref frames)>=limits::project_frames()){return .err(fail(1002,f.module_index));}id=v::len<clean::Frame>(ref frames)+1;f.context=id;if(!v::push<clean::Frame>(ref mut frames,f).is_ok || !v::push<Origin>(ref mut origins,Origin{.unit,.frame=i}).is_ok){return .err(fail(1003,f.module_index));}if(f.kind==2){let target=v::at_mut<usize>(ref mut receivers,f.receiver_type);*target=id;}else if(unique){let target=v::at_mut<usize>(ref mut declarations,bucket);*target=id;}else if(named_units::count(identified_inputs)!=0){let target=v::at_mut<usize>(ref mut specializations,bucket);*target=id;}}
        if(!v::push<usize>(ref mut maps,id).is_ok){return .err(fail(1003,f.module_index));}i+=1;
    }if(!v::push<Range>(ref mut units,Range{.start,.count=ir::count(p)}).is_ok){return .err(fail(1003,0));}unit+=1;
    }return .ok(Plan{.frames,.origins,.units,.maps,.work});
}
pub fn usize count(ref Plan p){return v::len<clean::Frame>(ref p.frames);}
pub fn clean::Frame at(ref Plan p,usize id){return *v::at<clean::Frame>(ref p.frames,id-1);}
pub fn Origin origin(ref Plan p,usize id){return *v::at<Origin>(ref p.origins,id-1);}
pub fn usize mapped(ref Plan p,usize unit,usize frame){if(frame==0 || unit>=v::len<Range>(ref p.units)){return 0;}let r=*v::at<Range>(ref p.units,unit);return if(frame>r.count){0}else{*v::at<usize>(ref p.maps,r.start+frame-1)};}
pub fn usize specialized(ref Plan p,ref v::Vec<ir::Program> inputs,usize unit,usize module_index,usize declaration,usize instance){let empty=named_units::empty();return specialized_inputs(p,inputs,ref empty,unit,module_index,declaration,instance);}
pub fn usize specialized_inputs(ref Plan p,ref v::Vec<ir::Program> inputs,ref named_units::Plan identified_inputs,usize unit,usize module_index,usize declaration,usize instance){if(unit>=inputs_view::count(inputs,identified_inputs)){return 0;}let source=if(named_units::count(identified_inputs)==0){v::at<ir::Program>(inputs,unit)}else{inputs_view::source_at(identified_inputs,unit)};var usize i=1;var usize found=0;while(i<=ir::count(source)){let f=clean::frame_at(ir::cleanup_ir(source),i);if(f.kind==1 && f.context==instance && f.module_index==module_index && f.declaration==declaration){if(found!=0){return 0;}found=mapped(p,unit,i);}i+=1;}return found;}
pub fn usize work(ref Plan p){return p.work;}
