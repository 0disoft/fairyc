module module_unit_value_field;
import module_unit_value_model as m;
import module_unit_interfaces as interfaces;
import module_types as types;
import scalar_views as views;
import std::bytes as bytes;
import std::vec as v;

pub struct Selected {usize type;m::Field field;}
pub fn Result<Selected,m::Problem> resolve(ref interfaces::Interfaces imported,ref v::Vec<types::Type> graph,ref m::Atoms atoms,m::Frame frame,m::Expression expression,usize receiver,ref u8[] name,ref mut usize work){
    if(receiver==0 || receiver>types::count(graph)){return .err(m::fail(frame,12,expression.start,expression.end,0,receiver));}if(!m::charge(work,name.len+1)){return .err(m::fail(frame,10,expression.start,expression.end,0,0));}
    let t=types::at(graph,receiver);var m::Field field=m::Field{.id=0,.owner=receiver,.module_index=0,.declaration=0,.source_index=0};var usize builtin=0;
    if((t.kind==9 || t.kind==10 || t.kind==11 || t.kind==15 && views::extended(t.scalar) || t.kind==17 || t.kind==18) && (bytes::equal(name,b"len") || bytes::equal(name,b"is_empty"))){builtin=m::atom(atoms,if(bytes::equal(name,b"len")){5}else{6});}
    else if(t.kind==16 && bytes::equal(name,b"len")){builtin=m::atom(atoms,5);}else if(t.kind==6 && bytes::equal(name,b"is_ok")){builtin=m::atom(atoms,6);}
    if(builtin!=0){return .ok(Selected{.type=builtin,.field});}
    if(t.kind==7 || t.kind==8){field.owner=t.first;}var usize probes=0;var u32 code=0;let usize id=switch(interfaces::field_measured(imported,field.owner,frame.module_index,name,ref mut probes)){case .ok(id):{id}case .err(c):{code=c;0}};if(!m::charge(work,probes)){return .err(m::fail(frame,10,expression.start,expression.end,0,0));}if(code!=0){return .err(m::fail(frame,if(code==6201){103}else{2},expression.start,expression.end,0,receiver));}
    let f=interfaces::field_at(imported,id-1);if(f.type==0 || f.type>types::count(graph)){return .err(m::fail(frame,12,expression.start,expression.end,0,f.type));}field.id=id;field.module_index=f.module_index;field.declaration=f.declaration;field.source_index=f.source_index;return .ok(Selected{.type=f.type,.field});
}
