module project_atlas_aggregate;
import project_compiler_input as input;
import module_public_type_text as coordinates;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import project_atlas_instances as instances;
import project_atlas_schema as schema;
import project_atlas_types as types;
import project_atlas_declarations as declarations;
import project_atlas_write as w;
import project_path as work;
import syntax as syntax;
import std::bytes as bytes;

// Publish the verified concrete layout, never a second substitution of source
// spelling. Original template fields and cleanup remain separate declarations.
pub fn Result<bool,u32> declaration(ref mut bytes::Buffer out,ref input::Input source,ref coordinates::Coordinates table,ref ir::Program program,instances::Row row,ref mut usize meter){
 let context=ir::cleanup_ir(program);
 if(!clean::aggregates(context) || row.type==0 || row.type>clean::type_count(context) || row.declaration.module_index>=input::count(source)){return .err(7);}
 let t=clean::type_at(context,row.type);let parsed=input::source(source,row.declaration.module_index);
 if(row.declaration.declaration>=syntax::declaration_count(parsed) || t.module_index!=row.declaration.module_index || t.declaration!=row.declaration.declaration+1 || t.kind!=20 && t.kind!=21){return .err(7);}
 let d=syntax::declaration_at(parsed,row.declaration.declaration);
 if(d.generic_parameters==0 || d.kind!=if(t.kind==20){4}else{6}){return .err(7);}
 var usize selected=ir::layout_count(program);var usize i=0;
 while(i<ir::layout_count(program)){if(!work::charge(meter,1)){return .err(3);}if(ir::layout_at(program,i).value.type==row.type){if(selected!=ir::layout_count(program)){return .err(7);}selected=i;}i+=1;}
 if(selected==ir::layout_count(program)){return .err(7);}let shape=ir::layout_at(program,selected).value;
 if(shape.module_index!=t.module_index || shape.declaration!=t.declaration || shape.field_count!=d.field_count || shape.field_start>ir::field_count(program) || shape.field_count>ir::field_count(program)-shape.field_start){return .err(7);}
 let glue=clean::glue_at(context,row.type);
 if(t.kind==21 && (glue.kind!=1 || glue.frame!=row.frame || glue.nonempty!=(row.frame!=0)) || t.kind==20 && row.frame!=0){return .err(7);}
 discard w::raw(out,b"{\"key\":",meter)?;discard schema::key(out,source,row.declaration,meter)?;
 discard w::raw(out,b",\"span\":",meter)?;discard declarations::span(out,source,t.module_index,d.start,d.end,meter)?;
 discard w::raw(out,b",\"provenance\":\"compiler-observed\",\"public\":",meter)?;discard w::raw(out,if(d.public){b"true"}else{b"false"},meter)?;
 discard w::raw(out,b",\"unsafe\":false,\"foreign\":false,\"abi\":\"\",\"link_name\":\"\",\"type\":",meter)?;
 let self=types::render(source,table,context,row.type,meter)?;discard w::raw(out,bytes::view(ref self),meter)?;
 discard w::raw(out,b",\"owner\":null,\"receiver\":\"\",\"parameters\":[],\"effects\":[],\"fields\":[",meter)?;i=0;
 while(i<shape.field_count){
  if(!work::charge(meter,1)){return .err(3);}let f=ir::field_at(program,shape.field_start+i);let written=syntax::field_at(parsed,d.field_start+i);
  let name=ir::field_name(program,shape.field_start+i);
  if(f.kind!=0 || f.module_index!=t.module_index || f.declaration!=t.declaration || f.source_index!=d.field_start+i || f.start!=written.start || f.end!=written.end || !bytes::equal(name,syntax::source_view(parsed)[written.name_start..written.name_end])){return .err(7);}
  if(i>0){discard w::raw(out,b",",meter)?;}discard w::raw(out,b"{",meter)?;discard w::pair(out,b"name",name,meter)?;
  discard w::raw(out,b",\"type\":",meter)?;let value=types::render(source,table,context,f.type,meter)?;discard w::raw(out,bytes::view(ref value),meter)?;
  discard w::raw(out,b",\"span\":",meter)?;discard declarations::span(out,source,t.module_index,f.start,f.end,meter)?;discard w::raw(out,b"}",meter)?;i+=1;
 }
 discard w::raw(out,b"],\"variants\":[],\"repr_c\":",meter)?;discard w::raw(out,if(shape.repr_c){b"true"}else{b"false"},meter)?;
 discard w::raw(out,b",",meter)?;discard w::pair(out,b"drop_kind",if(t.kind==20){b""}else if(glue.nonempty){b"non-empty"}else{b"empty"},meter)?;
 discard w::raw(out,b",\"drop_effects\":",meter)?;discard schema::effects(out,if(t.kind==21){glue.effects}else{0},meter)?;return w::raw(out,b"}",meter);
}
