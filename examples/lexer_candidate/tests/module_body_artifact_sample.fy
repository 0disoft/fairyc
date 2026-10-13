module tests::module_body_artifact_sample;
import tests::module_body_artifact_cleanup as fixture_clean;
import module_unit_body_ir_model as ir;
import module_unit_value_model as values;
import module_unit_interfaces as interfaces;
import module_unit_region_model as regions;
import syntax as syntax;
import std::vec as v;
import std::bytes as bytes;
fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
pub fn ir::Program build(){let cleaned=fixture_clean::build();
var v::Vec<ir::Node> nodes=v::empty<ir::Node>();assert(v::push<ir::Node>(ref mut nodes,ir::Node{.value=values::Expression{.constant=0,.constant_bits=0,.source_index=1,.binding=0,.kind=1,.start=3,.end=5,.left=0,.right=0,.target=0,.field=values::Field{.id=0,.owner=0,.module_index=0,.declaration=0,.source_index=0},.place=values::no_place(),.builtin=values::Builtin{.kind=0,.element=0,.slice=false,.unsafe_context=false},.expected=1,.type=1,.state=1,.flexible=true,.minimum=false},.syntax=syntax::Expression{.kind=1,.start=3,.end=5,.token=1,.left=0,.right=0},.literal_kind=3,.text_start=0,.text_end=2,.cooked_start=2,.cooked_end=2}).is_ok);
var v::Vec<ir::Statement> statements=v::empty<ir::Statement>();assert(v::push<ir::Statement>(ref mut statements,ir::Statement{.source_index=1,.value=syntax::Statement{.kind=5,.start=2,.end=6,.expression=1,.target=0,.first=0,.second=0,.entry=0,.operator=0},.operator_start=2,.operator_end=2}).is_ok && v::push<ir::Statement>(ref mut statements,ir::Statement{.source_index=2,.value=syntax::Statement{.kind=7,.start=2,.end=6,.expression=0,.target=0,.first=1,.second=0,.entry=0,.operator=0},.operator_start=2,.operator_end=2}).is_ok && v::push<ir::Statement>(ref mut statements,ir::Statement{.source_index=3,.value=syntax::Statement{.kind=1,.start=0,.end=10,.expression=0,.target=0,.first=2,.second=1,.entry=0,.operator=0},.operator_start=2,.operator_end=2}).is_ok);let text=copied(b"42");let body=ir::assemble_body_regions(3,10,0,10,nodes,statements,v::empty<values::Local>(),v::empty<values::Invocation>(),v::empty<values::Argument>(),v::empty<regions::Scope>(),v::empty<regions::Placement>(),v::empty<regions::Exit>(),text);var v::Vec<ir::Body> bodies=v::empty<ir::Body>();assert(v::push<ir::Body>(ref mut bodies,body).is_ok);return switch(ir::assemble(cleaned,bodies,v::empty<ir::Layout>(),v::empty<interfaces::Field>(),v::empty<interfaces::Callable>(),v::empty<usize>(),bytes::empty(),0)){case .ok(p):{p}case .err(_):{assert(false);return build();}};}
