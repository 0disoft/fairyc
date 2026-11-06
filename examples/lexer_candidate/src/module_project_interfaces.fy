module module_project_interfaces;
import module_project_identified_ir as checked;
import module_project_artifact_admission as admitted;
import module_project_public_view as view;
import module_public_interface_extract as extract;
import module_public_interface_model as model;
import module_inline_packages as inline;
import module_public_interface as ordinary;
import module_generic_interface as generic;
import module_public_templates as public;
import module_public_type_text as coordinates;
import module_signatures as signatures;
import module_source_provenance as provenance;
import module_source_specialization_set as specializations;
import generic_source_plan as plans;
import generic_source_plan_input as input;
import generic_source_catalog as catalog;
import project_path as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import module_state_graph_source as authored;

// Detached package interfaces derived only after complete body/Flow checking.
// No parser/planner owner, provisional Interfaces or caller-supplied hash survives.
pub resource Record{usize owner;u32 inline_0;u32 inline_1;u32 inline_2;u32 inline_3;u32 inline_4;u32 inline_5;u32 inline_6;u32 inline_7;extract::Projection ordinary;generic::Templates templates;bytes::Buffer ordinary_hash;bytes::Buffer source_hash;v::Vec<bytes::Buffer> specializations;cleanup{}}
pub resource Records{v::Vec<Record> packages;cleanup{}}
pub fn usize count(ref Records p){return v::len<Record>(ref p.packages);}
pub fn ref Record at(ref Records p,usize index){assert(index<count(p));return v::at<Record>(ref p.packages,index);}
pub fn usize owner(ref Record p){return p.owner;}
pub fn u32[8] inline_hash(ref Record p){return [p.inline_0,p.inline_1,p.inline_2,p.inline_3,p.inline_4,p.inline_5,p.inline_6,p.inline_7];}
pub fn ref model::Descriptor descriptor(ref Record p){return extract::descriptor(ref p.ordinary);}
pub fn ref generic::Templates templates(ref Record p){return ref p.templates;}
pub fn ref u8[] ordinary_hash(ref Record p){return bytes::view(ref p.ordinary_hash);}
pub fn ref u8[] source_hash(ref Record p){return bytes::view(ref p.source_hash);}
pub fn ref v::Vec<bytes::Buffer> source_specializations(ref Record p){return ref p.specializations;}
// Complete graph source has no public/generic/inline declarations. Derive its
// original module descriptor and both interfaces from its sealed source proof.
pub fn Result<Records,u32> build_state_graph(ref authored::Produced current,usize owner,ref mut usize work){
    if(!budget::charge(work,1)){return .err(10);}if(owner>=512){return .err(9);}var usize measured=0;
    let descriptor=switch(extract::extract_state_graph(current,ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(10);}return .err(code);}};if(!budget::charge(work,measured)){return .err(10);}
    measured=0;let ordinary_hash=switch(ordinary::hash_graph(authored::layouts(current),authored::owner(current).package,extract::descriptor(ref descriptor),ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(10);}return .err(code);}};if(!budget::charge(work,measured)){return .err(10);}
    let templates=generic::assemble(bytes::empty(),v::empty<generic::Template>());measured=0;
    let source_hash=switch(generic::hash_graph(authored::layouts(current),authored::owner(current).package,extract::descriptor(ref descriptor),ref templates,ref mut measured)){case .ok(p):{p}case .err(code):{if(!budget::charge(work,measured)){return .err(10);}return .err(code);}};if(!budget::charge(work,measured)){return .err(10);}
    let record=Record{.owner,.inline_0=0,.inline_1=0,.inline_2=0,.inline_3=0,.inline_4=0,.inline_5=0,.inline_6=0,.inline_7=0,.ordinary=descriptor,.templates,.ordinary_hash,.source_hash,.specializations=v::empty<bytes::Buffer>()};
    var packages=v::empty<Record>();if(!v::push<Record>(ref mut packages,record).is_ok){return .err(11);}return .ok(Records{.packages});
}

fn Result<Record,u32> package(ref view::View facts,ref plans::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,usize owner,ref mut usize work){
    let original=plans::input(plan);let templates=switch(public::collect_records(input::templates(original),input::dependencies(original),owner,work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    var usize measured=0;let descriptor=switch(extract::extract_records_prepared(sources,metadata,view::type_records(facts),view::shape_records(facts),view::field_records(facts),view::name_records(facts),table,view::type_names(facts),owner,true,ref mut measured)){case .ok(p):{p}case .err(e):{discard budget::charge(work,measured);return .err(e);}};if(!budget::charge(work,measured)){return .err(10);}if(extract::has_templates(ref descriptor)!=(generic::count(ref templates)!=0)){return .err(9);}
    measured=0;let ordinary_hash=switch(ordinary::hash(signatures::state_layouts(metadata),owner,extract::descriptor(ref descriptor),ref mut measured)){case .ok(p):{p}case .err(e):{discard budget::charge(work,measured);return .err(e);}};if(!budget::charge(work,measured)){return .err(10);}
    measured=0;let source_hash=switch(generic::hash(signatures::state_layouts(metadata),owner,extract::descriptor(ref descriptor),ref templates,ref mut measured)){case .ok(p):{p}case .err(e):{discard budget::charge(work,measured);return .err(e);}};if(!budget::charge(work,measured)){return .err(10);}
    let selected=switch(specializations::collect(plan,owner,work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    let inline_hash=switch(inline::derive(sources,metadata,table,owner,work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    return .ok(Record{.owner,.inline_0=inline_hash[0],.inline_1=inline_hash[1],.inline_2=inline_hash[2],.inline_3=inline_hash[3],.inline_4=inline_hash[4],.inline_5=inline_hash[5],.inline_6=inline_hash[6],.inline_7=inline_hash[7],.ordinary=descriptor,.templates,.ordinary_hash,.source_hash,.specializations=selected});
}
// Authenticate the complete checked view and original plan once for all owners.
// Independent descriptor/hash/closure histories fold into the 16MiB ledger.
// Return no package prefix if a later package or the shared budget fails.
pub fn Result<Records,u32> build(ref checked::Program program,ref plans::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize work){
    let facts=switch(view::build(program,sources,metadata,table,work)){case .ok(p):{p}case .err(e):{return .err(e);}};return collect(ref facts,plan,sources,metadata,table,work);
}
fn Result<Records,u32> collect(ref view::View facts,ref plans::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize work){
    let current=provenance::current(catalog::packages(input::templates(plans::input(plan))),sources,metadata,table,work);if(current!=0){return .err(current);}
    var bool[512] owners=[false;512];var usize i=0;while(i<signatures::module_count(metadata)){if(!budget::charge(work,1)){return .err(10);}let owner=signatures::module_owner(metadata,i);if(owner>=512){return .err(9);}owners[owner]=true;i+=1;}
    var v::Vec<Record> packages=v::empty<Record>();i=0;while(i<512){if(!budget::charge(work,1)){return .err(10);}if(owners[i]){let record=switch(package(facts,plan,sources,metadata,table,i,work)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!v::push<Record>(ref mut packages,record).is_ok){return .err(11);}}i+=1;}
    return .ok(Records{.packages});
}
pub fn Result<Records,u32> build_admitted(ref admitted::Admitted program,ref plans::Plan plan,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref coordinates::Coordinates table,ref mut usize work){let facts=switch(view::build_admitted(program,sources,metadata,table,work)){case .ok(p):{p}case .err(code):{return .err(code);}};return collect(ref facts,plan,sources,metadata,table,work);}
