module module_unit_cleanup_mir_model;
import module_types as types;
import candidate_limits as limits;
import module_unit_cleanup_mir_verify as proof;
import std::vec as v;

// Source-independent cleanup MIR. Frame IDs qualify all binding and expression
// operands; canonical type IDs never become temporary joined declaration IDs.
pub struct Frame {u32 kind;usize context;usize module_index;usize declaration;usize result;u32 effects;usize receiver;usize receiver_type;}
// Glue 0 has no resource body, 1 is a resource, 2 Option and 3 Result.
// Resource fields retain declaration order; execution visits them in reverse.
pub struct Glue {u32 kind;usize frame;bool nonempty;u32 effects;u32 abilities;usize field_start;usize field_count;}
pub struct Field {usize owner;usize ordinal;usize type;}
// Operand kind 1 is a lexical binding, kind 2 a materialized expression value.
pub struct Operand {usize frame;u32 kind;usize id;usize type;usize start;usize end;}
pub struct Drop {usize operand;u32 exit;usize start;usize end;}
pub struct Problem {u32 code;usize module_index;usize start;usize end;}
pub resource Program {v::Vec<types::Type> types;v::Vec<Frame> frames;v::Vec<Glue> glues;v::Vec<Field> fields;v::Vec<Operand> operands;v::Vec<Drop> drops;usize work;bool project;bool aggregates;cleanup{}}
pub fn usize type_count(ref Program p){return v::len<types::Type>(ref p.types);}
pub fn ref v::Vec<types::Type> type_records(ref Program p){return ref p.types;}
pub fn usize glue_count(ref Program p){return v::len<Glue>(ref p.glues);}
pub fn types::Type type_at(ref Program p,usize id){assert(id!=0 && id<=type_count(p));return *v::at<types::Type>(ref p.types,id-1);}
pub fn Glue glue_at(ref Program p,usize type){assert(type!=0 && type<=type_count(p));return *v::at<Glue>(ref p.glues,type-1);}
pub fn usize frame_count(ref Program p){return v::len<Frame>(ref p.frames);}
pub fn Frame frame_at(ref Program p,usize id){assert(id!=0 && id<=frame_count(p));return *v::at<Frame>(ref p.frames,id-1);}
pub fn usize field_count(ref Program p){return v::len<Field>(ref p.fields);}
pub fn Field field_at(ref Program p,usize index){return *v::at<Field>(ref p.fields,index);}
pub fn usize operand_count(ref Program p){return v::len<Operand>(ref p.operands);}
pub fn Operand operand_at(ref Program p,usize id){assert(id!=0 && id<=operand_count(p));return *v::at<Operand>(ref p.operands,id-1);}
pub fn usize drop_count(ref Program p){return v::len<Drop>(ref p.drops);}
// Linked projects combine selected unit sites; operand identity stays unit-sized.
pub fn usize drop_limit(bool project){return if(project){32768}else{16384};}
pub fn Drop drop_at(ref Program p,usize index){return *v::at<Drop>(ref p.drops,index);}
pub fn usize work(ref Program p){return p.work;}
pub fn bool project(ref Program p){return p.project;}
pub fn bool aggregates(ref Program p){return p.aggregates;}
pub fn usize work_limit(ref Program p){return if(p.project){limits::project_work()}else{1048576};}
fn Result<Program,Problem> assemble_scoped(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops,usize work,bool project,bool aggregates){
    let p=Program{.types,.frames,.glues,.fields,.operands,.drops,.work,.project,.aggregates};let checked=proof::check(ref p);if(checked.code!=0){return .err(checked);}return .ok(p);
}
pub fn Result<Program,Problem> assemble(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops,usize work){
    return assemble_scoped(types,frames,glues,fields,operands,drops,work,false,false);
}
pub fn Result<Program,Problem> assemble_project(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops,usize work){
    return assemble_scoped(types,frames,glues,fields,operands,drops,work,true,false);
}

// Compiler-private construction. Ordinary and artifact entry points stay closed
// until aggregate restoration has an explicit source and transport contract.
pub fn Result<Program,Problem> assemble_aggregates(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops,usize work){return assemble_scoped(types,frames,glues,fields,operands,drops,work,false,true);}

// Linked aggregate units preserve both admission modes. Ordinary project and
// artifact constructors still require their existing non-aggregate contracts.
pub fn Result<Program,Problem> assemble_project_aggregates(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops,usize work){return assemble_scoped(types,frames,glues,fields,operands,drops,work,true,true);}

// Artifact reconstruction resets producer accounting and re-enters the same
// verifier. Fresh verification work is returned even on rejection; this owner
// alone grants neither source checking nor whole-package cache-hit authority.
pub fn Result<Program,Problem> assemble_artifact(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops,bool project,ref mut usize verification_work){
    return artifact_mode(types,frames,glues,fields,operands,drops,project,false,verification_work);
}
// Explicit detached aggregate reconstruction. Fresh verification grants no
// current-source, Complete, cache-hit or executable publication authority.
pub fn Result<Program,Problem> assemble_aggregate_artifact(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops,bool project,ref mut usize verification_work){
    return artifact_mode(types,frames,glues,fields,operands,drops,project,true,verification_work);
}
fn Result<Program,Problem> artifact_mode(v::Vec<types::Type> types,v::Vec<Frame> frames,v::Vec<Glue> glues,v::Vec<Field> fields,v::Vec<Operand> operands,v::Vec<Drop> drops,bool project,bool aggregates,ref mut usize verification_work){
    if(v::len<Glue>(ref glues)!=v::len<types::Type>(ref types)){return .err(Problem{.code=1001,.module_index=0,.start=0,.end=0});}
    let p=Program{.types,.frames,.glues,.fields,.operands,.drops,.work=0,.project,.aggregates};let checked=proof::check_metered(ref p,verification_work);if(checked.code!=0){return .err(checked);}return .ok(p);
}
