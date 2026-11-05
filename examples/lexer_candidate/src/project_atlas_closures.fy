module project_atlas_closures;
import project_compiler_input as input;
import project_atlas_declarations as declarations;
import project_atlas_instances as instances;
import project_atlas_schema as schema;
import project_atlas_model as model;
import project_atlas_write as w;
import project_path as work;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import copied_closure_tables as private;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// A checked helper body can serve several concrete owners. Atlas gives each
// owner its Go-compatible private declaration, without changing executable IR.
pub struct Row{declarations::Row declaration;usize frame;usize owner;usize start;usize end;}
pub resource Projection{bytes::Buffer names;v::Vec<Row> rows;cleanup{}}
pub fn Row plain(declarations::Row declaration){return Row{.declaration,.frame=0,.owner=0,.start=0,.end=0};}
pub fn usize count(ref Projection p){return v::len<Row>(ref p.rows);}
pub fn Row at(ref Projection p,usize index){return *v::at<Row>(ref p.rows,index);}
pub fn ref u8[] name(ref Projection p,Row row){return bytes::view(ref p.names)[row.start..row.end];}
pub fn Result<bool,u32> key(ref mut bytes::Buffer out,ref input::Input source,ref Projection p,Row row,ref mut usize meter){return schema::key_named(out,source,row.declaration,name(p,row),meter);}
fn bool helper(ref input::Input source,declarations::Row row){return private::name(syntax::private_names(input::source(source,row.module_index)),row.declaration+1).len!=0;}
fn Result<usize,u32> owner(ref syntax::Parsed p,usize helper,ref mut usize meter){
 let h=syntax::declaration_at(p,helper);var usize result=0;var usize i=0;
 while(i<syntax::declaration_count(p)){if(!work::charge(meter,1)){return .err(3);}let d=syntax::declaration_at(p,i);
  if(d.kind==3 && private::name(syntax::private_names(p),i+1).len==0 && d.start<=h.start && h.end<=d.end){if(result!=0){return .err(7);}result=i+1;}i+=1;
 }return if(result==0){.err(7)}else{.ok(result)};
}
fn Result<bool,u32> push(ref mut v::Vec<Row> rows,Row row,ref mut usize meter){if(!work::charge(meter,1) || v::len<Row>(rows)>=4096){return .err(3);}return if(v::push<Row>(rows,row).is_ok){.ok(true)}else{.err(2)};}
fn Result<bool,u32> retain(ref input::Input source,ref ir::Program program,ref v::Vec<declarations::Row> originals,usize module_index,usize declaration,usize selected,ref u8[] identity,ref mut usize ordinal,ref mut bytes::Buffer names,ref mut v::Vec<Row> rows,ref mut usize meter){
 let p=input::source(source,module_index);var usize i=0;
 while(i<v::len<declarations::Row>(originals)){if(!work::charge(meter,1)){return .err(3);}let row=*v::at<declarations::Row>(originals,i);i+=1;
  if(row.module_index!=module_index || !helper(source,row) || owner(p,row.declaration,meter)?!=declaration){continue;}
  let d=syntax::declaration_at(p,row.declaration);if(d.public || d.generic_parameters!=0 || d.kind!=3){return .err(7);}var usize frame=0;var usize j=1;
  while(j<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}let f=clean::frame_at(ir::cleanup_ir(program),j);if((f.kind==1 || f.kind==3) && f.module_index==module_index && f.declaration==row.declaration+1){if(frame!=0){return .err(7);}frame=j;}j+=1;}
  if(frame==0 || ir::call_count(ir::at(program,frame))!=0){return .err(7);};
  *ordinal=*ordinal+1;let start=bytes::len(names);discard w::raw(names,b"_fy_closure_",meter)?;
  if(selected!=0){if(identity.len!=67 || !bytes::equal(identity[..3],b"g0_")){return .err(7);}discard w::raw(names,b"_fy_generic_",meter)?;discard w::raw(names,identity,meter)?;}
  else{discard w::raw(names,syntax::declaration_name(p,declaration),meter)?;}
  discard w::raw(names,b"_",meter)?;discard w::number(names,*ordinal,meter)?;
  discard push(rows,Row{.declaration=row,.frame,.owner=selected,.start,.end=bytes::len(names)},meter)?;
 }return .ok(true);
}
pub fn Result<Projection,u32> build(ref input::Input source,ref ir::Program program,ref instances::Instances context,ref v::Vec<declarations::Row> originals,ref mut usize meter){
 var names=bytes::empty();var rows=v::empty<Row>();var bool active=false;var usize i=0;
 while(i<v::len<declarations::Row>(originals)){if(!work::charge(meter,1)){return .err(3);}let row=*v::at<declarations::Row>(originals,i);i+=1;if(helper(source,row)){active=true;}else{discard push(ref mut rows,plain(row),meter)?;}}
 if(!active){return .ok(Projection{.names,.rows});}var usize m=0;
 while(m<input::count(source)){if(!work::charge(meter,1)){return .err(3);}var bool local=false;i=0;while(i<v::len<declarations::Row>(originals)){if(!work::charge(meter,1)){return .err(3);}if(v::at<declarations::Row>(originals,i).module_index==m){local=true;break;}i+=1;}if(!local){m+=1;continue;}
  let p=input::source(source,m);var usize ordinal=0;i=0;
  // Go first emits ordinary owners in written order, then concrete owners in
  // canonical g0 order. Unused generic bodies produce no helper declarations.
  while(i<syntax::declaration_count(p)){if(!work::charge(meter,1)){return .err(3);}let d=syntax::declaration_at(p,i);i+=1;if(d.kind==3 && d.generic_parameters==0 && private::name(syntax::private_names(p),i).len==0){discard retain(source,program,originals,m,i,0,b"",ref mut ordinal,ref mut names,ref mut rows,meter)?;}}
  i=0;while(i<instances::count(context)){if(!work::charge(meter,1)){return .err(3);}let instance=instances::at(context,i);i+=1;if(instance.type==0 && instance.frame!=0 && instance.declaration.module_index==m){discard retain(source,program,originals,m,instance.declaration.declaration+1,instance.frame,instances::identity(context,instance),ref mut ordinal,ref mut names,ref mut rows,meter)?;}}
  m+=1;
 }
 i=1;
 while(i<v::len<Row>(ref rows)){let row=*v::at<Row>(ref rows,i);var left=bytes::empty();discard schema::key_named(ref mut left,source,row.declaration,bytes::view(ref names)[row.start..row.end],meter)?;var usize j=i;
  while(j>0){let prior=*v::at<Row>(ref rows,j-1);var right=bytes::empty();discard schema::key_named(ref mut right,source,prior.declaration,bytes::view(ref names)[prior.start..prior.end],meter)?;if(bytes::equal(bytes::view(ref left),bytes::view(ref right))){return .err(7);}if(!model::before(bytes::view(ref left),bytes::view(ref right))){break;}{let slot=v::at_mut<Row>(ref mut rows,j);*slot=prior;}{let slot=v::at_mut<Row>(ref mut rows,j-1);*slot=row;}j-=1;}i+=1;
 }return .ok(Projection{.names,.rows});
}
pub fn Result<bool,u32> target(ref mut bytes::Buffer out,ref input::Input source,ref Projection p,declarations::Row declaration,usize caller,ref mut usize meter){
 if(!helper(source,declaration)){return schema::key(out,source,declaration,meter);}var usize i=0;var usize found=count(p);
 while(i<count(p)){if(!work::charge(meter,1)){return .err(3);}let row=at(p,i);if(row.declaration.module_index==declaration.module_index && row.declaration.declaration==declaration.declaration && (row.owner==0 || row.owner==caller)){if(found!=count(p)){return .err(7);}found=i;}i+=1;}
 if(found==count(p)){return .err(7);}return key(out,source,p,at(p,found),meter);
}
