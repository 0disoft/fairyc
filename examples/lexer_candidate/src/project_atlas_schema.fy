module project_atlas_schema;
import project_compiler_input as input;
import project_compiler_core as compiler;
import module_project_identified_ir as identified;
import module_signatures as signatures;
import module_types as graph;
import module_public_type_text as coordinates;
import project_atlas_types as types;
import project_atlas_template_types as templates;
import project_atlas_declarations as spans;
import project_atlas_write as w;
import project_path as work;
import effect_declarations as effect_policy;
import repr_c_declarations as representation;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import project_source_plan as planner;
import generic_aggregate_types as aggregates;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;

// Build from the exact source closure authenticated by Complete. Rebuilding the
// signature table preserves unused foreign declarations and source-only types;
// the executable callable table cannot stand in for that declaration inventory.
pub fn Result<signatures::Program,u32> prepare(ref input::Input source,ref coordinates::Coordinates table,ref compiler::Complete complete,ref mut usize meter){
    let metadata=prepare_declarations(source,meter)?;
    let current=identified::current(compiler::checked_program(complete),input::sources(source),ref metadata,table,meter);
    if(current!=0){return .err(if(current==10){3}else{7});}
    return .ok(metadata);
}
// Re-enter the source/type owner before projecting an aggregate Complete. The
// ordinary projection remains closed; detached type rows cannot select this mode.
pub fn Result<signatures::Program,u32> prepare_aggregates(ref input::Input source,ref coordinates::Coordinates table,ref compiler::Complete complete,ref mut usize meter){
    if(!clean::aggregates(ir::cleanup_ir(identified::ir_program(compiler::checked_program(complete))))){return .err(7);}
    let original=switch(planner::build_aggregate_sources(input::sources(source),input::owners(source),input::bindings(source),table,meter)){case .ok(p):{p}case .err(e):{return .err(if(e.code==10){3}else if(e.code==11){2}else{7});}};
    let prepared=switch(aggregates::build(original,input::sources(source),meter)){case .ok(p):{p}case .err(e):{return .err(if(e.code==10){3}else if(e.code==11){2}else{7});}};
    let metadata=switch(signatures::build_aggregates(input::sources(source),prepared,meter)){case .ok(p):{p}case .err(e):{return .err(if(e.code==10){3}else if(e.code==11){2}else{7});}};
    let current=identified::current(compiler::checked_program(complete),input::sources(source),ref metadata,table,meter);if(current!=0){return .err(if(current==10){3}else{7});}return .ok(metadata);
}
// Source-only original-header projection, matching GenericDeclarations. It is
// not Complete, current-source query authority, or proof of any template body.
pub fn Result<signatures::Program,u32> prepare_declarations(ref input::Input source,ref mut usize meter){
    if(!work::charge(meter,input::count(source)+1)){return .err(3);}
    let catalog=switch(input::declarations(source)){case .ok(p):{p}case .err(_):{return .err(7);}};
    let metadata=switch(signatures::build_catalog(input::sources(source),catalog)){case .ok(p):{p}case .err(_):{return .err(7);}};
    return .ok(metadata);
}
pub fn Result<v::Vec<graph::Type>,u32> type_rows(ref signatures::Program metadata,ref mut usize meter){
    if(signatures::type_count(metadata)>4096){return .err(7);}var rows=v::empty<graph::Type>();var usize i=1;
    while(i<=signatures::type_count(metadata)){
        if(!work::charge(meter,1)){return .err(3);}if(!v::push<graph::Type>(ref mut rows,signatures::type_at(metadata,i)).is_ok){return .err(2);}i+=1;
    }return .ok(rows);
}
pub fn ref u8[] kind(u32 kind){
    if(kind==4){return b"struct";}if(kind==6){return b"resource";}if(kind==11){return b"enum";}if(kind==9){return b"foreign-function";}return b"function";
}
pub fn Result<bool,u32> key(ref mut bytes::Buffer out,ref input::Input source,spans::Row row,ref mut usize meter){
    return key_named(out,source,row,b"",meter);
}
pub fn Result<bool,u32> key_named(ref mut bytes::Buffer out,ref input::Input source,spans::Row row,ref u8[] override,ref mut usize meter){
    let parsed=input::source(source,row.module_index);let d=syntax::declaration_at(parsed,row.declaration);
    let name=types::module_name(input::file(source,row.module_index),meter)?;
    discard w::raw(out,b"{",meter)?;discard w::pair(out,b"package_instance",input::instance_id(source,row.module_index),meter)?;
    discard w::raw(out,b",",meter)?;discard w::pair(out,b"module",bytes::view(ref name),meter)?;
    discard w::raw(out,b",",meter)?;discard w::pair(out,b"kind",kind(d.kind),meter)?;
    let written=if(override.len==0 && d.owner_end!=0){spans::name(parsed,row.declaration+1,meter)?}else{bytes::empty()};discard w::raw(out,b",",meter)?;discard w::pair(out,b"name",if(override.len!=0){override}else if(d.owner_end!=0){bytes::view(ref written)}else{syntax::declaration_name(parsed,row.declaration+1)},meter)?;return w::raw(out,b"}",meter);
}
pub fn Result<bool,u32> effects(ref mut bytes::Buffer out,u32 bits,ref mut usize meter){
    if(bits>4095){return .err(7);}let labels=b"block_context|clock|console|emit|environment|external_call|filesystem|network|process|random|storage_read|storage_write|";
    let u32[12] flags=[128,32,1,256,4,512,2,16,8,64,1024,2048];var usize index=0;var usize start=0;var usize end=0;var bool first=true;
    discard w::raw(out,b"[",meter)?;while(end<labels.len){
        if(!work::charge(meter,1)){return .err(3);}if(labels[end]==124){if((bits&flags[index])!=0){if(!first){discard w::raw(out,b",",meter)?;}discard w::string(out,labels[start..end],meter)?;first=false;}index+=1;start=end+1;}end+=1;
    }return w::raw(out,b"]",meter);
}
fn Result<usize,u32> resolve(ref input::Input source,ref signatures::Program metadata,usize module_index,usize root,ref mut v::Vec<graph::Type> rows,ref mut usize meter){
    if(!work::charge(meter,1) || graph::count(rows)>4096){return .err(3);}
    let p=input::source(source,module_index);if(root==0 || root>syntax::type_count(p)){return .err(7);}
    let t=signatures::resolve_type(metadata,p,module_index,root,rows);if(t.problem.code!=0 || t.type==0 || graph::count(rows)>4096){return .err(7);}return .ok(t.type);
}
fn Result<bool,u32> type(ref mut bytes::Buffer out,ref input::Input source,ref coordinates::Coordinates table,ref v::Vec<graph::Type> rows,usize id,ref mut usize meter){
    let value=types::render_records(source,table,rows,id,meter)?;return w::raw(out,bytes::view(ref value),meter);
}
fn Result<bool,u32> field(ref mut bytes::Buffer out,ref input::Input source,ref coordinates::Coordinates table,ref signatures::Program metadata,ref mut v::Vec<graph::Type> rows,usize module_index,usize id,usize template_declaration,ref mut usize meter){
    let p=input::source(source,module_index);if(id>=syntax::field_count(p)){return .err(7);}let f=syntax::field_at(p,id);
    let usize t=if(template_declaration==0){resolve(source,metadata,module_index,f.type_root,rows,meter)?}else{0};
    discard w::raw(out,b"{",meter)?;discard w::pair(out,b"name",syntax::source_view(p)[f.name_start..f.name_end],meter)?;
    discard w::raw(out,b",\"type\":",meter)?;if(template_declaration==0){discard type(out,source,table,rows,t,meter)?;}else{discard templates::render(out,source,table,metadata,module_index,template_declaration-1,f.type_root,meter)?;}
    discard w::raw(out,b",\"span\":",meter)?;discard spans::span(out,source,module_index,if(template_declaration==0){f.start}else{f.name_start},if(template_declaration==0){f.end}else{f.name_end},meter)?;return w::raw(out,b"}",meter);
}
fn Result<bool,u32> fields(ref mut bytes::Buffer out,ref input::Input source,ref coordinates::Coordinates table,ref signatures::Program metadata,ref mut v::Vec<graph::Type> rows,usize module_index,usize first,usize count,usize template_declaration,ref mut usize meter){
    discard w::raw(out,b"[",meter)?;var usize i=0;while(i<count){if(!work::charge(meter,1)){return .err(3);}if(i>0){discard w::raw(out,b",",meter)?;}discard field(out,source,table,metadata,rows,module_index,first+i,template_declaration,meter)?;i+=1;}return w::raw(out,b"]",meter);
}
pub fn Result<bytes::Buffer,u32> declaration(ref input::Input source,ref coordinates::Coordinates table,ref signatures::Program metadata,ref mut v::Vec<graph::Type> rows,spans::Row row,ref mut usize meter){
    return declaration_named(source,table,metadata,rows,row,b"",meter);
}
pub fn Result<bytes::Buffer,u32> declaration_named(ref input::Input source,ref coordinates::Coordinates table,ref signatures::Program metadata,ref mut v::Vec<graph::Type> rows,spans::Row row,ref u8[] override,ref mut usize meter){
    if(row.module_index>=input::count(source)){return .err(7);}let p=input::source(source,row.module_index);
    if(row.declaration>=syntax::declaration_count(p)){return .err(7);}if(!work::charge(meter,1)){return .err(3);}let d=syntax::declaration_at(p,row.declaration);
    if(d.kind!=3 && d.kind!=4 && d.kind!=6 && d.kind!=9 && d.kind!=11 || d.owner_end!=0 && (d.kind!=3 || d.generic_parameters!=0)){return .err(101);}
    let template=d.generic_parameters!=0;if(template && d.kind!=3 && d.kind!=4 && d.kind!=6){return .err(101);}
    if(template && d.kind==4 && representation::read_template(p,row.declaration).code!=0){return .err(101);}
    if(template && d.kind==6){var usize i=0;while(i<d.attribute_count){if(!work::charge(meter,1)){return .err(3);}let a=syntax::attribute_at(p,d.attribute_start+i);if(!bytes::equal(syntax::source_view(p)[a.name_start..a.name_end],b"effects")){return .err(101);}i+=1;}if(effect_policy::read(p,row.declaration).code!=0){return .err(7);}}
    var usize result=0;var u32 declared_effects=0;var bool unsafe_call=false;var usize signature=0;
    if(template){if(d.kind==3){let effects=effect_policy::read(p,row.declaration);if(effects.code!=0){return .err(7);}declared_effects=effects.bits;unsafe_call=d.unsafe_end!=0;}}
    else if(d.kind==3 || d.kind==9){signature=signatures::identity(metadata,row.module_index,row.declaration+1);if(signature==0){return .err(7);}let s=signatures::at(metadata,signature);result=s.result;declared_effects=s.effects;unsafe_call=s.unsafe_call;}
    else{let t=signatures::declaration_type(metadata,row.module_index,row.declaration+1,rows);if(t.problem.code!=0){return .err(7);}result=t.type;}
    var out=bytes::empty();discard w::raw(ref mut out,b"{",meter)?;if(template){discard w::raw(ref mut out,b"\"type_parameters\":",meter)?;discard templates::parameters(ref mut out,p,row.declaration,meter)?;discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"\"key\":",meter)?;discard key_named(ref mut out,source,row,override,meter)?;
    discard w::raw(ref mut out,b",\"span\":",meter)?;discard spans::span(ref mut out,source,row.module_index,d.start,d.end,meter)?;
    discard w::raw(ref mut out,b",\"provenance\":\"compiler-declared\",\"public\":",meter)?;discard w::raw(ref mut out,if(d.public){b"true"}else{b"false"},meter)?;
    discard w::raw(ref mut out,if(unsafe_call){b",\"unsafe\":true"}else{b",\"unsafe\":false"},meter)?;
    discard w::raw(ref mut out,if(d.kind==9){b",\"foreign\":true,"}else{b",\"foreign\":false,"},meter)?;
    discard w::pair(ref mut out,b"abi",if(d.kind==9){b"C"}else{b""},meter)?;discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"link_name",if(d.kind==9){signatures::linker_name(metadata,signature)}else{b""},meter)?;
    discard w::raw(ref mut out,b",\"type\":",meter)?;if(template){if(d.kind==3){discard templates::render(ref mut out,source,table,metadata,row.module_index,row.declaration,d.result_type,meter)?;}else{discard templates::self(ref mut out,source,table,row.module_index,row.declaration,meter)?;}}else{discard type(ref mut out,source,table,rows,result,meter)?;}
    discard w::raw(ref mut out,b",\"owner\":",meter)?;
    if(d.owner_end==0){discard w::raw(ref mut out,b"null",meter)?;}else{if(!work::charge(meter,signatures::type_count(metadata)+1)){return .err(3);}let id=signatures::owner_type(metadata,p,row.module_index,row.declaration+1);if(id==0){return .err(7);}discard type(ref mut out,source,table,rows,id,meter)?;}
    let receiver=signatures::receiver_kind(metadata,p,row.module_index,row.declaration+1);
    discard w::raw(ref mut out,b",",meter)?;discard w::pair(ref mut out,b"receiver",if(receiver==1){b"shared"}else if(receiver==2){b"exclusive"}else if(receiver==3){b"value"}else{b""},meter)?;discard w::raw(ref mut out,b",\"parameters\":[",meter)?;
    var usize i=0;var usize parameters=0;
    while(i<d.entry_count){if(!work::charge(meter,1)){return .err(3);}let e=syntax::entry_at(p,d.entry_start+i);i+=1;if(e.kind!=1){continue;}
        if(!template && (signature==0 || parameters>=signatures::at(metadata,signature).parameter_count)){return .err(7);}
        let usize id=if(template){0}else{signatures::parameter_at(metadata,signatures::at(metadata,signature).parameter_start+parameters)};
        if(parameters>0){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"{",meter)?;
        discard w::pair(ref mut out,b"name",syntax::source_view(p)[e.name_start..e.name_end],meter)?;discard w::raw(ref mut out,b",\"type\":",meter)?;
        if(template){discard templates::render(ref mut out,source,table,metadata,row.module_index,row.declaration,e.type_root,meter)?;}else{discard type(ref mut out,source,table,rows,id,meter)?;}discard w::raw(ref mut out,b",\"mutable\":false,\"span\":",meter)?;
        discard spans::span(ref mut out,source,row.module_index,e.name_start,e.name_end,meter)?;discard w::raw(ref mut out,b"}",meter)?;parameters+=1;
    }
    if(signature!=0 && parameters!=signatures::at(metadata,signature).parameter_count){return .err(7);}
    discard w::raw(ref mut out,b"],\"effects\":",meter)?;discard effects(ref mut out,declared_effects,meter)?;
    discard w::raw(ref mut out,b",\"fields\":",meter)?;
    if(d.kind==4 || d.kind==6){discard fields(ref mut out,source,table,metadata,rows,row.module_index,d.field_start,d.field_count,if(template){row.declaration+1}else{0},meter)?;}else{discard w::raw(ref mut out,b"[]",meter)?;}
    discard w::raw(ref mut out,b",\"variants\":[",meter)?;i=0;var usize variants=0;
    while(d.kind==11 && i<d.field_count){if(!work::charge(meter,1)){return .err(3);}let f=syntax::field_at(p,d.field_start+i);i+=1;if(f.kind!=3){continue;}
        if(variants>0){discard w::raw(ref mut out,b",",meter)?;}discard w::raw(ref mut out,b"{",meter)?;
        discard w::pair(ref mut out,b"name",syntax::source_view(p)[f.name_start..f.name_end],meter)?;discard w::raw(ref mut out,b",\"span\":",meter)?;
        discard spans::span(ref mut out,source,row.module_index,f.start,f.end,meter)?;discard w::raw(ref mut out,b",\"fields\":",meter)?;
        discard fields(ref mut out,source,table,metadata,rows,row.module_index,f.payload_start,f.payload_count,0,meter)?;
        discard w::raw(ref mut out,b"}",meter)?;variants+=1;
    }
    discard w::raw(ref mut out,b"],\"repr_c\":",meter)?;
    discard w::raw(ref mut out,if(d.kind==4 && if(template){representation::read_template(p,row.declaration).repr_c}else{representation::read(p,row.declaration).repr_c}){b"true"}else{b"false"},meter)?;
    discard w::raw(ref mut out,b",",meter)?;
    discard w::pair(ref mut out,b"drop_kind",if(d.kind!=6){b""}else if(template){b"source-declared"}else if(syntax::block_statement_count(p,d.cleanup_body)==0 && syntax::statement_at(p,d.cleanup_body).expression==0){b"empty"}else{b"non-empty"},meter)?;
    discard w::raw(ref mut out,b",\"drop_effects\":",meter)?;
    discard effects(ref mut out,if(d.kind==6){effect_policy::read(p,row.declaration).bits}else{0},meter)?;
    discard w::raw(ref mut out,b"}",meter)?;return .ok(out);
}
