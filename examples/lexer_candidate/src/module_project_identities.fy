module module_project_identities;
import candidate_limits as limits;
import module_project_identified_compile as units;
import module_identified_ir as identified;
import module_source_specializations as selection;
import module_public_interface_model as records;
import module_project_types as types;
import module_project_frames as frames;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import std::bytes as bytes;
import std::vec as v;

pub resource Table{bytes::Buffer names;v::Vec<selection::Row> types;v::Vec<selection::Row> frames;cleanup{}}
fn selection::Row empty(){return selection::Row{.node=0,.id=records::Text{.start=0,.end=0}};}
fn clean::Problem fail(u32 code){return clean::Problem{.code,.module_index=0,.start=0,.end=0};}
pub fn usize type_count(ref Table p){return v::len<selection::Row>(ref p.types);}
pub fn ref u8[] names_view(ref Table p){return bytes::view(ref p.names);}
pub fn usize frame_count(ref Table p){return v::len<selection::Row>(ref p.frames);}
pub fn selection::Row type_at(ref Table p,usize id){assert(id>0 && id<=type_count(p));return *v::at<selection::Row>(ref p.types,id-1);}
pub fn selection::Row frame_at(ref Table p,usize id){assert(id>0 && id<=frame_count(p));return *v::at<selection::Row>(ref p.frames,id-1);}
pub fn ref u8[] identity(ref Table p,selection::Row row){assert(row.id.start<=row.id.end && row.id.end<=bytes::len(ref p.names));return bytes::view(ref p.names)[row.id.start..row.id.end];}
fn u32 store(ref mut v::Vec<selection::Row> rows,usize index,selection::Row row,ref u8[] id,ref mut bytes::Buffer names,ref mut usize work){
    if(index==0 || index>v::len<selection::Row>(rows)){return 1001;}if(!limits::project_charge(work,id.len+1)){return 1002;}let old=*v::at<selection::Row>(rows,index-1);
    if(row.node==0){return if(id.len==0 && old.node==0){0}else{1001};}if(id.len!=67 || !bytes::equal(id[..3],b"g0_")){return 1001;}var usize i=3;while(i<id.len){let c=id[i];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return 1001;}i+=1;}
    if(old.node!=0){return if(bytes::equal(bytes::view(names)[old.id.start..old.id.end],id)){0}else{1001};}let start=bytes::len(names);if(start>1048576-id.len){return 1002;}if(!bytes::append(names,id).is_ok){return 1003;}let target=v::at_mut<selection::Row>(rows,index-1);*target=selection::Row{.node=row.node,.id=records::Text{.start,.end=bytes::len(names)}};return 0;
}
pub fn Result<Table,clean::Problem> build(ref units::Plan input,ref types::Plan graph,ref frames::Plan functions,ref mut usize work){
    if(units::count(input)==0 || units::count(input)!=types::unit_count(graph) || types::type_count(graph)>4096 || frames::count(functions)>limits::project_frames()){return .err(fail(1001));}var bytes::Buffer names=bytes::empty();var v::Vec<selection::Row> selected_types=v::empty<selection::Row>();var v::Vec<selection::Row> selected_frames=v::empty<selection::Row>();var usize i=0;
    while(i<types::type_count(graph)){if(!limits::project_charge(work,1)){return .err(fail(1002));}if(!v::push<selection::Row>(ref mut selected_types,empty()).is_ok){return .err(fail(1003));}i+=1;}i=0;while(i<frames::count(functions)){if(!limits::project_charge(work,1)){return .err(fail(1002));}if(!v::push<selection::Row>(ref mut selected_frames,empty()).is_ok){return .err(fail(1003));}i+=1;}
    var usize unit=0;while(unit<units::count(input)){let p=units::at(input,unit);let source=identified::ir_program(p);if(identified::frame_count(p)!=ir::count(source)){return .err(fail(1001));}i=1;
        while(i<=clean::type_count(ir::cleanup_ir(source))){let row=identified::type_at(p,i);if((row.node!=0)!=(clean::type_at(ir::cleanup_ir(source),i).kind==13)){return .err(fail(1001));}let code=store(ref mut selected_types,types::mapped(graph,unit,i),row,identified::identity(p,row),ref mut names,work);if(code!=0){return .err(fail(code));}i+=1;}i=1;
        while(i<=ir::count(source)){let row=identified::frame_at(p,i);let code=store(ref mut selected_frames,frames::mapped(functions,unit,i),row,identified::identity(p,row),ref mut names,work);if(code!=0){return .err(fail(code));}i+=1;}unit+=1;
    }
    let result=Table{.names,.types=selected_types,.frames=selected_frames};i=1;while(i<=frames::count(functions)){if(!limits::project_charge(work,1)){return .err(fail(1002));}let f=frames::at(functions,i);let row=frame_at(ref result,i);if(row.node!=0 && f.kind!=1 && f.kind!=2){return .err(fail(1001));}if(f.kind==2){let receiver=type_at(ref result,f.receiver_type);let id=identity(ref result,row);if(!limits::project_charge(work,id.len+1)){return .err(fail(1002));}if(!bytes::equal(id,identity(ref result,receiver))){return .err(fail(1001));}}i+=1;}
    return .ok(result);
}
