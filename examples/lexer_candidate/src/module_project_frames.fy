module module_project_frames;
import module_project_types as types;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import std::vec as v;

pub struct Origin {usize unit;usize frame;}
pub struct Range {usize start;usize count;}
pub resource Plan {v::Vec<clean::Frame> frames;v::Vec<Origin> origins;v::Vec<Range> units;v::Vec<usize> maps;usize work;cleanup{}}
fn clean::Problem fail(u32 code,usize module_index){return clean::Problem{.code,.module_index,.start=0,.end=0};}
fn bool ordinary(ref ir::Program p,clean::Frame f){if(f.kind==3){return true;}var usize i=0;while(i<ir::callable_count(p)){let c=ir::callable_at(p,i);if(c.module_index==f.module_index && c.declaration==f.declaration){return !c.template && !c.foreign;}i+=1;}return false;}
pub fn Result<Plan,clean::Problem> build(ref v::Vec<ir::Program> inputs,ref types::Plan graph){
    if(v::len<ir::Program>(inputs)!=types::unit_count(graph)){return .err(fail(1001,0));}var v::Vec<clean::Frame> frames=v::empty<clean::Frame>();var v::Vec<Origin> origins=v::empty<Origin>();var v::Vec<Range> units=v::empty<Range>();var v::Vec<usize> maps=v::empty<usize>();var usize work=0;var usize unit=0;
    while(unit<v::len<ir::Program>(inputs)){let p=v::at<ir::Program>(inputs,unit);let usize start=v::len<usize>(ref maps);var usize i=1;while(i<=ir::count(p)){if(!budget::charge(ref mut work,1)){return .err(fail(1002,0));}var f=clean::frame_at(ir::cleanup_ir(p),i);f.result=types::mapped(graph,unit,f.result);f.receiver_type=types::mapped(graph,unit,f.receiver_type);if(f.result==0){return .err(fail(1001,f.module_index));}var usize id=0;var usize j=0;
        while(j<v::len<clean::Frame>(ref frames)){if(!budget::charge(ref mut work,1)){return .err(fail(1002,f.module_index));}let old=*v::at<clean::Frame>(ref frames,j);let selected=*v::at<Origin>(ref origins,j);let peer=v::at<ir::Program>(inputs,selected.unit);
            if(f.kind==2 && old.kind==2 && f.receiver_type==old.receiver_type){if(f.module_index!=old.module_index || f.declaration!=old.declaration || f.result!=old.result || f.effects!=old.effects || f.receiver!=old.receiver){return .err(fail(1001,f.module_index));}id=j+1;break;}
            if(f.kind!=2 && old.kind!=2 && f.module_index==old.module_index && f.declaration==old.declaration && ordinary(p,f) && ordinary(peer,old)){return .err(fail(1001,f.module_index));}j+=1;
        }
        if(id==0){if(v::len<clean::Frame>(ref frames)>=1024){return .err(fail(1002,f.module_index));}id=v::len<clean::Frame>(ref frames)+1;f.context=id;if(!v::push<clean::Frame>(ref mut frames,f).is_ok || !v::push<Origin>(ref mut origins,Origin{.unit,.frame=i}).is_ok){return .err(fail(1003,f.module_index));}}
        if(!v::push<usize>(ref mut maps,id).is_ok){return .err(fail(1003,f.module_index));}i+=1;
    }if(!v::push<Range>(ref mut units,Range{.start,.count=ir::count(p)}).is_ok){return .err(fail(1003,0));}unit+=1;
    }return .ok(Plan{.frames,.origins,.units,.maps,.work});
}
pub fn usize count(ref Plan p){return v::len<clean::Frame>(ref p.frames);}
pub fn clean::Frame at(ref Plan p,usize id){return *v::at<clean::Frame>(ref p.frames,id-1);}
pub fn Origin origin(ref Plan p,usize id){return *v::at<Origin>(ref p.origins,id-1);}
pub fn usize mapped(ref Plan p,usize unit,usize frame){if(frame==0 || unit>=v::len<Range>(ref p.units)){return 0;}let r=*v::at<Range>(ref p.units,unit);return if(frame>r.count){0}else{*v::at<usize>(ref p.maps,r.start+frame-1)};}
pub fn usize specialized(ref Plan p,ref v::Vec<ir::Program> inputs,usize unit,usize module_index,usize declaration,usize instance){if(unit>=v::len<ir::Program>(inputs)){return 0;}let source=v::at<ir::Program>(inputs,unit);var usize i=1;var usize found=0;while(i<=ir::count(source)){let f=clean::frame_at(ir::cleanup_ir(source),i);if(f.kind==1 && f.context==instance && f.module_index==module_index && f.declaration==declaration){if(found!=0){return 0;}found=mapped(p,unit,i);}i+=1;}return found;}
pub fn usize work(ref Plan p){return p.work;}
