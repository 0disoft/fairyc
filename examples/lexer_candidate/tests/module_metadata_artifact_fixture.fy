module tests::module_metadata_artifact_fixture;
import module_metadata_artifact_model as model;
import module_unit_body_ir_model as ir;
import module_unit_interfaces as interfaces;
import module_unit_cleanup_mir_model as clean;
import module_types as types;
import std::bytes as bytes;
import std::vec as v;
pub fn clean::Program cleaned(bool changed){var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();
assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=1,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);
assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=7,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);
assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
assert(v::push<types::Type>(ref mut graph,types::Type{.kind=1,.scalar=6,.module_index=0,.declaration=0,.first=0,.second=0}).is_ok);
assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
assert(v::push<types::Type>(ref mut graph,types::Type{.kind=2,.scalar=0,.module_index=1,.declaration=if(changed){9}else{3},.first=0,.second=0}).is_ok);
assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
assert(v::push<types::Type>(ref mut graph,types::Type{.kind=3,.scalar=0,.module_index=0,.declaration=4,.first=0,.second=0}).is_ok);
assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=1,.frame=0,.nonempty=false,.effects=0,.abilities=1,.field_start=0,.field_count=1}).is_ok);
assert(v::push<types::Type>(ref mut graph,types::Type{.kind=8,.scalar=0,.module_index=0,.declaration=0,.first=4,.second=0}).is_ok);
assert(v::push<clean::Glue>(ref mut glues,clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0}).is_ok);
var v::Vec<clean::Field> fields=v::empty<clean::Field>();assert(v::push<clean::Field>(ref mut fields,clean::Field{.owner=5,.ordinal=0,.type=1}).is_ok);
return switch(clean::assemble(graph,v::empty<clean::Frame>(),glues,fields,v::empty<clean::Operand>(),v::empty<clean::Drop>(),0)){case .ok(p):{p}case .err(_):{assert(false);return cleaned(changed);}};}
pub fn model::Metadata metadata(ref clean::Program cleaned){
var v::Vec<ir::Layout> layouts=v::empty<ir::Layout>();
assert(v::push<ir::Layout>(ref mut layouts,ir::Layout{.value=interfaces::Shape{.type=4,.module_index=1,.declaration=3,.field_start=0,.field_count=2,.abilities=0,.repr_c=true,.start=10,.end=50,.cleanup_nonempty=false,.cleanup_effects=0},.name_start=0,.name_end=1}).is_ok);
assert(v::push<ir::Layout>(ref mut layouts,ir::Layout{.value=interfaces::Shape{.type=5,.module_index=0,.declaration=4,.field_start=2,.field_count=1,.abilities=1,.repr_c=false,.start=60,.end=100,.cleanup_nonempty=false,.cleanup_effects=0},.name_start=3,.name_end=4}).is_ok);
var v::Vec<interfaces::Field> fields=v::empty<interfaces::Field>();
assert(v::push<interfaces::Field>(ref mut fields,interfaces::Field{.kind=1,.type=1,.module_index=1,.declaration=3,.source_index=1,.start=20,.end=25,.name_start=1,.name_end=2,.payload_start=0,.payload_count=0}).is_ok);
assert(v::push<interfaces::Field>(ref mut fields,interfaces::Field{.kind=1,.type=3,.module_index=1,.declaration=3,.source_index=2,.start=30,.end=35,.name_start=2,.name_end=3,.payload_start=0,.payload_count=0}).is_ok);
assert(v::push<interfaces::Field>(ref mut fields,interfaces::Field{.kind=1,.type=1,.module_index=0,.declaration=4,.source_index=1,.start=70,.end=75,.name_start=4,.name_end=5,.payload_start=0,.payload_count=0}).is_ok);
var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();
assert(v::push<interfaces::Callable>(ref mut callables,interfaces::Callable{.module_index=0,.declaration=1,.template=false,.result=1,.parameter_start=0,.parameter_count=1,.effects=0,.must_use=true,.unsafe_call=false,.foreign=false,.link_start=0,.link_end=0}).is_ok);
assert(v::push<interfaces::Callable>(ref mut callables,interfaces::Callable{.module_index=1,.declaration=8,.template=false,.result=1,.parameter_start=1,.parameter_count=1,.effects=4,.must_use=false,.unsafe_call=true,.foreign=true,.link_start=5,.link_end=19}).is_ok);
var v::Vec<usize> parameters=v::empty<usize>();
assert(v::push<usize>(ref mut parameters,6).is_ok);
assert(v::push<usize>(ref mut parameters,1).is_ok);
let names=switch(bytes::copy(b"SxyRnforeign_number")){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};var usize work=0;return switch(model::assemble(layouts,fields,callables,parameters,names,cleaned,2,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return metadata(cleaned);}};}
