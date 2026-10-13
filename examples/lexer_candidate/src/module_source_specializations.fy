module module_source_specializations;
import generic_source_plan as source;
import generic_source_plan_input as input;
import generic_source_catalog as catalog;
import generic_concrete_types as concrete;
import generic_package_bindings as bindings;
import module_unit_interfaces as typed;
import module_unit_generics as generics;
import module_signatures as signatures;
import module_public_interface_model as records;
import module_types as types;
import generic_scalar_arguments as source_arguments;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Bridge compiler-local one-based handles to original-source g0_ identities.
// Source template declaration indices are zero-based; compiler handles are one-based.
// Zero rows denote ordinary functions/types/cleanup, never a specialization.
// This detached selection does not authorize executable or cache publication.
pub struct Problem{u32 code;u32 kind;usize id;}
pub struct Row{usize node;records::Text id;}
pub resource Selection{bytes::Buffer names;v::Vec<Row> functions;v::Vec<Row> types;v::Vec<Row> cleanups;cleanup{}}
pub fn usize function_count(ref Selection p){return v::len<Row>(ref p.functions);}
pub fn usize type_count(ref Selection p){return v::len<Row>(ref p.types);}
pub fn usize cleanup_count(ref Selection p){return v::len<Row>(ref p.cleanups);}
pub fn Row function_at(ref Selection p,usize id){assert(id>0 && id<=function_count(p));return *v::at<Row>(ref p.functions,id-1);}
pub fn Row type_at(ref Selection p,usize id){assert(id>0 && id<=type_count(p));return *v::at<Row>(ref p.types,id-1);}
pub fn Row cleanup_at(ref Selection p,usize id){assert(id>0 && id<=cleanup_count(p));return *v::at<Row>(ref p.cleanups,id-1);}
pub fn ref u8[] identity(ref Selection p,Row row){assert(row.id.start<=row.id.end && row.id.end<=bytes::len(ref p.names));return bytes::view(ref p.names)[row.id.start..row.id.end];}
fn Row empty(){return Row{.node=0,.id=records::Text{.start=0,.end=0}};}
fn u32 current(ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref mut usize work){
    let templates=input::templates(source::input(plan));let packages=catalog::packages(templates);
    if(bindings::count(packages)!=v::len<syntax::Parsed>(sources) || signatures::module_count(metadata)!=v::len<syntax::Parsed>(sources)){return 9;}
    var usize i=0;while(i<v::len<syntax::Parsed>(sources)){if(!budget::charge(work,1)){return 10;}let p=v::at<syntax::Parsed>(sources,i);let code=catalog::current(packages,i,p,work);if(code!=0){return code;}if(bindings::module_at(packages,i).owner!=signatures::module_owner(metadata,i)){return 9;}if(!budget::charge(work,syntax::source_view(p).len+1)){return 10;}if(!signatures::source_matches(metadata,i,p)){return 12;}i+=1;}
    i=0;while(i<bindings::import_count(packages)){let row=bindings::import_at(packages,i);var usize measured=0;let target=signatures::import_target_measured(metadata,row.module_index,bindings::text(packages,row.alias),ref mut measured);if(!budget::charge(work,measured+1)){return 10;}if(target.code!=0 || target.item!=row.target+1){return 9;}i+=1;}
    let count=generics::count(typed::specializations(graph));if(count>generics::instance_limit() || typed::type_count(graph)>4096 || generics::cleanup_count(typed::specializations(graph))>4096){return 9;}
    if(!budget::charge(work,typed::type_count(graph)+v::len<syntax::Parsed>(sources)+1)){return 10;}
    return typed::validate_specializations(graph,sources,metadata).code;
}
fn Result<Row,u32> select(ref source::Plan plan,usize module_index,usize declaration,usize count,usize[8] arguments,ref mut bytes::Buffer names,ref mut usize work){
    let templates=input::templates(source::input(plan));var usize selected=0;var usize i=0;
    while(i<source::count(plan)){if(!budget::charge(work,count+1)){return .err(10);}let node=source::at(plan,i);let row=catalog::template_at(templates,node.template_index);if(row.module_index==module_index && row.declaration+1==declaration && node.count==count){var bool same=true;var usize j=0;while(j<count){if(arguments[j]!=node.arguments[j]){same=false;}j+=1;}if(same){if(selected!=0 || node.state!=2){return .err(9);}selected=i+1;}}i+=1;}
    if(selected==0){return .err(9);}let id=source::text(plan,source::at(plan,selected-1).id);if(id.len!=67 || !bytes::equal(id[..3],b"g0_")){return .err(9);}if(bytes::len(names)>1048576-id.len || !budget::charge(work,id.len+1)){return .err(10);}let start=bytes::len(names);if(!bytes::append(names,id).is_ok){return .err(11);}return .ok(Row{.node=selected,.id=records::Text{.start,.end=bytes::len(names)}});
}
fn Result<Row,u32> callable(ref source::Plan plan,ref typed::Interfaces graph,usize module_index,usize declaration,usize first,usize count,ref mut bytes::Buffer names,ref mut usize work){
    if(count==0){return .ok(empty());}if(count>8){return .err(9);}var usize[8] arguments=[0;8];var usize i=0;
    while(i<count){if(!budget::charge(work,1)){return .err(10);}let id=generics::argument_at(typed::specializations(graph),first+i);if(!source_arguments::descriptor(id) && (id==0 || id>typed::type_count(graph))){return .err(9);}arguments[i]=switch(if(source_arguments::descriptor(id)){concrete::lookup_id(input::types(source::input(plan)),source_arguments::name(id),work)}else{concrete::lookup_typed(input::types(source::input(plan)),typed::type_at(graph,id),work)}){case .ok(value):{value}case .err(code):{return .err(code);}};i+=1;}
    return select(plan,module_index,declaration,count,arguments,names,work);
}
pub fn Result<Selection,Problem> build(ref source::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref mut usize work){
    let code=current(plan,sources,metadata,graph,work);if(code!=0){return .err(Problem{.code,.kind=0,.id=0});}var bytes::Buffer names=bytes::empty();var v::Vec<Row> functions=v::empty<Row>();var v::Vec<Row> selected_types=v::empty<Row>();var v::Vec<Row> cleanups=v::empty<Row>();let specialized=typed::specializations(graph);var usize i=1;
    while(i<=generics::count(specialized)){if(!budget::charge(work,1)){return .err(Problem{.code=10,.kind=1,.id=i});}let instance=generics::at(specialized,i);let row=switch(callable(plan,graph,instance.module_index,instance.declaration,instance.argument_start,instance.argument_count,ref mut names,work)){case .ok(value):{value}case .err(code):{return .err(Problem{.code,.kind=1,.id=i});}};if(!v::push<Row>(ref mut functions,row).is_ok){return .err(Problem{.code=11,.kind=1,.id=i});}i+=1;}
    i=1;while(i<=typed::type_count(graph)){if(!budget::charge(work,1)){return .err(Problem{.code=10,.kind=2,.id=i});}let t=typed::type_at(graph,i);var Row row=empty();if(t.kind==13){if(t.scalar!=0 || t.second!=0 || t.first==0 || t.first>=i){return .err(Problem{.code=9,.kind=2,.id=i});}var usize[8] args=[0;8];args[0]=switch(concrete::lookup_typed(input::types(source::input(plan)),typed::type_at(graph,t.first),work)){case .ok(value):{value}case .err(code):{return .err(Problem{.code,.kind=2,.id=i});}};row=switch(select(plan,t.module_index,t.declaration,1,args,ref mut names,work)){case .ok(value):{value}case .err(code):{return .err(Problem{.code,.kind=2,.id=i});}};}if(!v::push<Row>(ref mut selected_types,row).is_ok){return .err(Problem{.code=11,.kind=2,.id=i});}i+=1;}
    i=1;while(i<=generics::cleanup_count(specialized)){if(!budget::charge(work,1)){return .err(Problem{.code=10,.kind=3,.id=i});}let item=generics::cleanup_at(specialized,i);let row=switch(callable(plan,graph,item.module_index,item.declaration,item.argument_start,item.argument_count,ref mut names,work)){case .ok(value):{value}case .err(code):{return .err(Problem{.code,.kind=3,.id=i});}};if(!v::push<Row>(ref mut cleanups,row).is_ok){return .err(Problem{.code=11,.kind=3,.id=i});}i+=1;}
    return .ok(Selection{.names,.functions,.types=selected_types,.cleanups});
}
