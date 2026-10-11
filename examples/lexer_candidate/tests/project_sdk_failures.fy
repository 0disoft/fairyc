module tests::project_sdk_failures;
import project_instance_graph as local;
import project_workspace_graph as graph;
import project_module_visibility as visibility;
import project_source_set as sources;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer join(ref u8[] root,ref u8[] leaf){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,root).is_ok && bytes::append_byte(ref mut result,47).is_ok && bytes::append(ref mut result,leaf).is_ok);return result;}
@effects(filesystem)
fn bytes::Buffer root(){var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] input=ref argument;return switch(files::manifest(input[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};}
// Meter placement measures the same source/graph input, without timing or a
// hard-coded fixture work cost. It exhausts work at an exact owning phase.
@effects(filesystem)
fn usize used(ref u8[] path,bool selection){var sources::Budget budget=sources::Budget{.files=17,.bytes=31};var usize meter=0;let input=switch(local::load(path,ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return 0;}};if(selection){let selected=switch(local::select_sdk(ref input,false,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return 0;}};discard local::sdk_profile(ref selected);}return meter;}
@effects(filesystem)
fn void exhausted(ref u8[] path,bool selection){let usize cost=used(path,selection);assert(cost>0 && cost<16777216);var usize meter=16777216-cost;var sources::Budget budget=sources::Budget{.files=17,.bytes=31};let failure=switch(visibility::load(path,ref mut budget,ref mut meter)){case .ok(_):{assert(false);return;}case .err(value):{value}};assert(budget.files==17 && budget.bytes==31 && meter==16777216 && visibility::load_phase(ref failure)==4);let e=visibility::sdk_error(ref failure);let p=graph::error(e);assert(local::node_count(graph::error_local_graph(e))==1 && p.node==0);if(selection){assert(p.code==8 && p.reason==5);}else{assert(p.code==4 && p.reason==0);}}
@test
@effects(filesystem)
fn void selection_and_bundle_work_failures_keep_distinct_errors_and_graph(){let owned=root();let core=join(bytes::view(ref owned),b"Core");exhausted(bytes::view(ref core),false);exhausted(bytes::view(ref core),true);var sources::Budget budget=sources::Budget{.files=4096,.bytes=31};var usize meter=0;let rejected=switch(visibility::load(bytes::view(ref core),ref mut budget,ref mut meter)){case .ok(_):{assert(false);return;}case .err(value):{value}};assert(visibility::load_phase(ref rejected)==1 && budget.files==4096 && budget.bytes==31);}
@test
@effects(filesystem)
fn void unneeded_sdk_stays_lazy_at_the_last_resolution_work_unit(){let owned=root();let path=join(bytes::view(ref owned),b"None");let usize cost=used(bytes::view(ref path),true);assert(cost>0 && cost<16777215);var usize meter=16777216-cost-1;var sources::Budget budget=sources::Budget{.files=17,.bytes=31};let loaded=switch(visibility::load(bytes::view(ref path),ref mut budget,ref mut meter)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(meter==16777216 && !graph::has_sdk(visibility::workspace_graph(ref loaded)) && visibility::lookup_count(ref loaded)==0 && budget.files==18 && budget.bytes>31);}
