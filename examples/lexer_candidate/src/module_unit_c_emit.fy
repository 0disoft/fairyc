module module_unit_c_emit;
import module_unit_c_function as function_emit;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cfg_model as cfg;
import module_unit_cfg_lower as flow;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import module_unit_c_scalar as scalar;
import module_unit_c_aggregate as aggregate;
import module_unit_c_literals as literals;
import module_unit_c_sdk as sdk;
import module_unit_c_callable_index as contracts;
import module_unit_c_frame_index as targets;
import module_unit_c_reachability as reachable;
import module_unit_c_region as lifecycle;
import module_unit_c_names as names;
import module_unit_c_calls as calls;
import module_identified_ir as identified;
import module_project_identified_ir as project;
import module_project_c_owner as owner;
import candidate_limits as limits;
import c_text as w;
import std::bytes as bytes;
import std::vec as v;

fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
// Keep the legacy API while keeping per-source borrow analysis bounded.
pub fn clean::Problem render(ref ir::Program p,usize entry,ref mut bytes::Buffer out){return function_emit::render_project(p,entry,out);}
pub fn clean::Problem render_identified(ref identified::Program program,usize entry,ref mut bytes::Buffer out){return function_emit::render_identified(program,entry,out);}

resource Prepared {names::Table named;contracts::Index indexed;targets::Index body_index;reachable::Plan selected;bytes::Buffer header;bytes::Buffer literal_source;usize entry;bool test_entry;usize work;cleanup{}}
resource Unit {usize first;usize stop;bytes::Buffer text;cleanup{}}
pub resource Emission {Prepared prepared;v::Vec<Unit> units;usize frames;usize work;usize retries;cleanup{}}
pub resource Bundle {owner::Program program;Emission emission;cleanup{}}
fn u32 error_code(w::State s){return if(s.code<1000){if(s.code==3){1003}else{1002}}else{s.code};}
fn clean::Problem project_entry(ref ir::Program raw,usize entry,bool test_entry){if(!clean::project(ir::cleanup_ir(raw)) || entry==0 || entry>ir::count(raw)){return fail(1001,0,0,0);}let f=clean::frame_at(ir::cleanup_ir(raw),entry);return fail(if(scalar::scalar(raw,f.result)!=if(test_entry){7}else{1} || calls::parameters(ir::at(raw,entry))!=0){1001}else{0},f.module_index,0,0);}
fn Result<Prepared,clean::Problem> prepare_project(ref ir::Program raw,ref v::Vec<cfg::Graph> graphs,ref sdk::Plan sdk_plan,usize preparation_work,names::Table named,usize entry,bool entry_only,bool test_entry){
    let checked=project_entry(raw,entry,test_entry);if(checked.code!=0){return .err(checked);}
    // Replay the same graph/SDK preparation cost on every image, including hits.
    var usize work=preparation_work;var usize i=0;let indexed=switch(contracts::build(raw,ref mut work)){case .ok(p):{p}case .err(e):{return .err(fail(e,0,0,0));}};
    let body_index=switch(targets::build(raw,ref indexed,ref mut work)){case .ok(p):{p}case .err(e):{return .err(fail(e,0,0,0));}};
    // Index/SDK preparation and the reachability fixed point each retain their
    // own work bound. Every completed history enters the whole-project ledger.
    var usize total=0;var usize name_work=0;if(!names::matches(ref named,raw,ref mut name_work)){return .err(fail(if(name_work>=1048576){1002}else{1001},0,0,0));}if(!limits::project_charge(ref mut total,names::work(ref named)+name_work)){return .err(fail(1002,0,0,0));}if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}work=0;
    let selected=switch(if(entry_only){reachable::build(raw,graphs,ref indexed,ref body_index,entry,ref mut work)}else{reachable::all(raw,ref mut work)}){case .ok(p):{p}case .err(e):{return .err(fail(e,0,0,0));}};
    if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}work=0;
    var bytes::Buffer header=bytes::empty();var w::State s=w::State{.code=0};if(!w::text(ref mut header,ref mut s,b"#ifndef FY_CANDIDATE_PROJECT_HEADER_V1\n#define FY_CANDIDATE_PROJECT_HEADER_V1\n") || !function_emit::project_prologue(raw,ref mut header,ref mut s,ref mut work)){return .err(fail(error_code(s),0,0,0));}
    let marks=switch(aggregate::definitions(raw,ref mut header,ref mut s,ref mut work)){case .ok(m):{m}case .err(e):{return .err(fail(if(e<1000){if(e==3){1003}else{1002}}else{e},0,0,0));}};
    if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}work=0;
    var bytes::Buffer literal_source=bytes::empty();if(!literals::shared_storage(raw,ref mut header,ref mut literal_source,ref mut s,ref mut work)){return .err(fail(error_code(s),0,0,0));}
    if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}work=0;
    if(!w::text(ref mut header,ref mut s,b"#include <stddef.h>\n") || !sdk::definitions(raw,sdk_plan,ref mut header,ref mut s,ref mut work) || !lifecycle::definitions(raw,sdk_plan,ref mut header,ref mut s,ref mut work) || !calls::foreign_signatures(raw,sdk_plan,ref indexed,ref mut header,ref mut s,ref mut work)){return .err(fail(error_code(s),0,0,0));}
    i=1;while(i<=ir::count(raw)){if(reachable::frame(ref selected,i) && (!calls::signature(raw,ref named,i,ref mut header,ref mut s,true) || !w::text(ref mut header,ref mut s,b";\n"))){return .err(fail(error_code(s),clean::frame_at(ir::cleanup_ir(raw),i).module_index,ir::start(ir::at(raw,i)),ir::end(ir::at(raw,i))));}i+=1;}
    if(!aggregate::drops_named(raw,ref named,if(entry_only){reachable::drops(ref selected)}else{ref marks},ref mut header,ref mut s,ref mut work) || !w::text(ref mut header,ref mut s,b"#endif\n")){return .err(fail(error_code(s),0,0,0));}
    if(!limits::project_charge(ref mut total,work)){return .err(fail(1002,0,0,0));}
    return .ok(Prepared{.named,.indexed,.body_index,.selected,.header,.literal_source,.entry,.test_entry,.work=total});
}
fn Result<bytes::Buffer,clean::Problem> project_unit(ref ir::Program raw,ref v::Vec<cfg::Graph> graphs,ref sdk::Plan sdk_plan,ref Prepared p,usize first,usize stop,ref mut usize work){
    var bytes::Buffer text=bytes::empty();var w::State s=w::State{.code=0};if(!w::text(ref mut text,ref mut s,b"/* fairyc.candidate-project-unit-c/v1 */\n#include \"fy_project.h\"\n")){return .err(fail(error_code(s),0,0,0));}var usize i=first;
    while(i<stop){if(reachable::frame(ref p.selected,i)){var usize start=ir::start(ir::at(raw,i));var usize end=ir::end(ir::at(raw,i));if(!function_emit::render(raw,ref p.named,sdk_plan,ref p.indexed,ref p.body_index,v::at<cfg::Graph>(graphs,i-1),ref mut text,ref mut s,work,ref mut start,ref mut end,true)){return .err(fail(error_code(s),clean::frame_at(ir::cleanup_ir(raw),i).module_index,start,end));}}i+=1;}
    if(first<=p.entry && p.entry<stop && (!w::text(ref mut text,ref mut s,if(p.test_entry){b"int32_t fy_candidate_entry(void){"}else{b"int32_t fy_candidate_entry(void){return "}) || !names::function(ref p.named,p.entry,ref mut text,ref mut s) || !w::text(ref mut text,ref mut s,if(p.test_entry){b"();return 0;}\n"}else{b"();}\n"}) || !w::text(ref mut text,ref mut s,b"#ifndef FY_CANDIDATE_NO_MAIN\nint main(void){return (int)fy_candidate_entry();}\n#endif\n"))){return .err(fail(error_code(s),0,0,0));}return .ok(text);
}
fn clean::Problem append_project_units(ref ir::Program raw,ref v::Vec<cfg::Graph> graphs,ref sdk::Plan sdk_plan,ref Prepared p,usize first,usize stop,usize depth,ref mut v::Vec<Unit> units,ref mut usize total,ref mut usize retries){
    var usize work=0;switch(project_unit(raw,graphs,sdk_plan,p,first,stop,ref mut work)){
        case .ok(text):{if(!limits::project_charge(total,work)){return fail(1002,0,0,0);}if(v::len<Unit>(units)>=limits::project_frames()){return fail(1002,0,0,0);}if(!v::push<Unit>(units,Unit{.first,.stop,.text}).is_ok){return fail(1003,0,0,0);}return fail(0,0,0,0);}
        case .err(e):{if(!limits::project_charge(total,work)){return fail(1002,0,0,0);}if(e.code!=1002 || stop-first<=1 || depth>=12){return e;};*retries=*retries+1;let middle=first+(stop-first)/2;let left=append_project_units(raw,graphs,sdk_plan,p,first,middle,depth+1,units,total,retries);if(left.code!=0){return left;}return append_project_units(raw,graphs,sdk_plan,p,middle,stop,depth+1,units,total,retries);}
    };return fail(1001,0,0,0);
}
// A complete project owns its validated IR and never exposes a successful prefix.
// Analysis, common declarations and each body attempt retain 1 MiB work;
// output buffers retain 2 MiB. All histories share limits::project_work().
pub fn Result<Bundle,clean::Problem> partition_scoped(owner::Program program,names::Table named,usize entry,usize max_frames,bool entry_only){return partition_mode(program,named,entry,max_frames,entry_only,false);}
pub fn Result<Bundle,clean::Problem> partition_test(owner::Program program,names::Table named,usize entry,usize max_frames){return partition_mode(program,named,entry,max_frames,true,true);}
fn Result<Bundle,clean::Problem> partition_mode(owner::Program program,names::Table named,usize entry,usize max_frames,bool entry_only,bool test_entry){
    let emission=switch(emit_mode(owner::ir_program(ref program),named,entry,max_frames,entry_only,test_entry)){case .ok(p):{p}case .err(e):{return .err(e);}};return .ok(Bundle{.program,.emission});
}
// The caller retains the checked IR. Every returned field owns its storage;
// no IR reference or owner is retained across test images.
pub fn Result<Emission,clean::Problem> emit_test(ref ir::Program raw,names::Table named,usize entry,usize max_frames){return emit_mode(raw,named,entry,max_frames,true,true);}
pub fn Result<Emission,clean::Problem> emit_lowered_test(ref owner::Lowered program,names::Table named,usize entry,usize max_frames){return emit_graphs(owner::ir_program(owner::lowered_program(program)),owner::lowered_graphs(program),owner::lowered_sdk(program),owner::lowered_work(program),named,entry,max_frames,true,true);}
fn Result<Emission,clean::Problem> emit_mode(ref ir::Program raw,names::Table named,usize entry,usize max_frames,bool entry_only,bool test_entry){
    if(max_frames==0 || max_frames>limits::project_frames()){return .err(fail(1001,0,0,0));}let checked=proof::check(raw);if(checked.code!=0){return .err(checked);}let valid=project_entry(raw,entry,test_entry);if(valid.code!=0){return .err(valid);}let graphs=switch(flow::lower_all(raw)){case .ok(p):{p}case .err(e):{return .err(e);}};var usize work=0;var usize i=0;while(i<v::len<cfg::Graph>(ref graphs)){if(!budget::charge(ref mut work,cfg::count(v::at<cfg::Graph>(ref graphs,i)))){return .err(fail(1002,0,0,0));}i+=1;}let sdk_plan=switch(sdk::build(raw,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};return emit_graphs(raw,ref graphs,ref sdk_plan,work,named,entry,max_frames,entry_only,test_entry);
}
fn Result<Emission,clean::Problem> emit_graphs(ref ir::Program raw,ref v::Vec<cfg::Graph> graphs,ref sdk::Plan sdk_plan,usize preparation_work,names::Table named,usize entry,usize max_frames,bool entry_only,bool test_entry){
    if(max_frames==0 || max_frames>limits::project_frames()){return .err(fail(1001,0,0,0));}let p=switch(prepare_project(raw,graphs,sdk_plan,preparation_work,named,entry,entry_only,test_entry)){case .ok(p):{p}case .err(e):{return .err(e);}};let frames=ir::count(raw);var usize total=p.work;var usize retries=0;var v::Vec<Unit> units=v::empty<Unit>();var usize first=1;
    while(first<=frames){let stop=if(max_frames<frames+1-first){first+max_frames}else{frames+1};let e=append_project_units(raw,graphs,sdk_plan,ref p,first,stop,0,ref mut units,ref mut total,ref mut retries);if(e.code!=0){return .err(e);}first=stop;}
    let usize count=v::len<Unit>(ref units);var usize index=0;var usize next=1;while(index<count){let unit=v::at<Unit>(ref units,index);if(unit.first!=next || unit.first>=unit.stop){return .err(fail(1001,0,0,0));}next=unit.stop;index+=1;}if(next!=frames+1){return .err(fail(1001,0,0,0));}
    return .ok(Emission{.prepared=p,.units,.frames,.work=total,.retries});
}
fn Result<Bundle,clean::Problem> partition_plain(ir::Program program,usize entry,usize max_frames,bool entry_only){if(max_frames==0 || max_frames>limits::project_frames()){return .err(fail(1001,0,0,0));}let owned=switch(owner::plain(program)){case .ok(p):{p}case .err(e):{return .err(e);}};return partition_scoped(owned,names::empty(),entry,max_frames,entry_only);}
pub fn Result<Bundle,clean::Problem> partition(ir::Program program,usize entry,usize max_frames){return partition_plain(program,entry,max_frames,false);}
pub fn Result<Bundle,clean::Problem> partition_entry(ir::Program program,usize entry,usize max_frames){return partition_plain(program,entry,max_frames,true);}
fn Result<Bundle,clean::Problem> partition_named(project::Program program,usize entry,usize max_frames,bool entry_only){if(max_frames==0 || max_frames>limits::project_frames()){return .err(fail(1001,0,0,0));}var usize work=0;let named=switch(names::build_project(ref program,ref mut work)){case .ok(p):{p}case .err(e):{return .err(e);}};let owned=switch(owner::named(program)){case .ok(p):{p}case .err(e):{return .err(e);}};return partition_scoped(owned,named,entry,max_frames,entry_only);}
pub fn Result<Bundle,clean::Problem> partition_project(project::Program program,usize entry,usize max_frames){return partition_named(program,entry,max_frames,false);}
pub fn Result<Bundle,clean::Problem> partition_project_entry(project::Program program,usize entry,usize max_frames){return partition_named(program,entry,max_frames,true);}
pub fn ref owner::Program project_owner(ref Bundle p){return ref p.program;}
pub fn ref Emission project_emission(ref Bundle p){return ref p.emission;}
pub fn ref u8[] project_header(ref Bundle p){return bytes::view(ref p.emission.prepared.header);}
pub fn ref u8[] project_literals(ref Bundle p){return bytes::view(ref p.emission.prepared.literal_source);}
pub fn usize project_unit_count(ref Bundle p){return v::len<Unit>(ref p.emission.units);}
pub fn usize project_frames(ref Bundle p){return p.emission.frames;}
pub fn usize project_selected_frames(ref Bundle p){return reachable::count(ref p.emission.prepared.selected);}
pub fn usize project_work(ref Bundle p){return p.emission.work;}
pub fn usize project_retries(ref Bundle p){return p.emission.retries;}
pub fn usize project_first(ref Bundle p,usize index){return v::at<Unit>(ref p.emission.units,index).first;}
pub fn usize project_stop(ref Bundle p,usize index){return v::at<Unit>(ref p.emission.units,index).stop;}
pub fn ref u8[] project_source(ref Bundle p,usize index){return bytes::view(ref v::at<Unit>(ref p.emission.units,index).text);}
pub fn ref u8[] emission_header(ref Emission p){return bytes::view(ref p.prepared.header);}
pub fn ref u8[] emission_literals(ref Emission p){return bytes::view(ref p.prepared.literal_source);}
pub fn usize emission_unit_count(ref Emission p){return v::len<Unit>(ref p.units);}
pub fn ref u8[] emission_source(ref Emission p,usize index){return bytes::view(ref v::at<Unit>(ref p.units,index).text);}
pub struct EmissionStats {usize frames;usize selected;usize work;usize retries;}
pub struct UnitRange {usize first;usize stop;}
pub fn EmissionStats emission_stats(ref Emission p){return EmissionStats{.frames=p.frames,.selected=reachable::count(ref p.prepared.selected),.work=p.work,.retries=p.retries};}
pub fn UnitRange emission_range(ref Emission p,usize index){let unit=v::at<Unit>(ref p.units,index);return UnitRange{.first=unit.first,.stop=unit.stop};}
