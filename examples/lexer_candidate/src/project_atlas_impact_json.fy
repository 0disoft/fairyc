module project_atlas_impact_json;
import project_atlas_join as join;
import project_atlas_impact as impact;
import project_atlas_validate as validate;
import project_atlas_wire as wire;
import project_atlas_model as model;
import project_json_tree as json;
import project_json_string as strings;
import project_path as work;
import std::bytes as bytes;

fn Result<bool,u32> append(ref mut bytes::Buffer out,ref u8[] s,usize limit,ref mut usize meter){if(bytes::len(out)>limit || s.len>limit-bytes::len(out) || !work::charge(meter,s.len+1)){return .err(3);}if(!bytes::append(out,s).is_ok){return .err(2);}return .ok(true);}
fn Result<bool,u32> quoted(ref mut bytes::Buffer out,ref u8[] s,usize limit,ref mut usize meter){let q=strings::quote(s,meter)?;return append(out,bytes::view(ref q),limit,meter);}
fn Result<bool,u32> symbol(ref mut bytes::Buffer out,ref join::Graph g,join::Symbol s,usize limit,ref mut usize meter){discard append(out,b"{\"PackageInstance\":",limit,meter)?;discard quoted(out,join::instance(g,s.package),limit,meter)?;discard append(out,b",\"Symbol\":",limit,meter)?;discard quoted(out,join::visible(g,s),limit,meter)?;return append(out,b"}",limit,meter);}
fn Result<bool,u32> number(ref mut bytes::Buffer out,usize value,usize limit,ref mut usize meter){var u8[20] digits=[0;20];var usize used=0;var usize n=value;while(true){var u8 digit=48;var usize rem=n%10;while(rem>0){digit+=1;rem-=1;}digits[used]=digit;used+=1;n/=10;if(n==0){break;}}while(used>0){used-=1;let ref u8[] raw=ref digits;discard append(out,raw[used..used+1],limit,meter)?;}return .ok(true);}
fn Result<bool,u32> field(ref mut bytes::Buffer out,ref json::Tree t,usize h,ref u8[] output,ref u8[] input,usize limit,ref mut usize meter){discard quoted(out,output,limit,meter)?;discard append(out,b":",limit,meter)?;return quoted(out,wire::string(t,h,input),limit,meter);}
fn Result<bool,u32> completeness(ref mut bytes::Buffer out,ref json::Tree t,usize row,usize limit,ref mut usize meter){discard append(out,b"{\"Relation\":",limit,meter)?;discard quoted(out,wire::string(t,row,b"relation"),limit,meter)?;discard append(out,b",\"Status\":",limit,meter)?;discard quoted(out,wire::string(t,row,b"status"),limit,meter)?;discard append(out,b",\"Resolved\":",limit,meter)?;discard number(out,wire::integer(t,row,b"resolved"),limit,meter)?;discard append(out,b",\"External\":",limit,meter)?;discard number(out,wire::integer(t,row,b"external"),limit,meter)?;discard append(out,b",\"Unknown\":",limit,meter)?;discard number(out,wire::integer(t,row,b"unknown"),limit,meter)?;discard append(out,b",\"Reasons\":",limit,meter)?;let reasons=wire::field(t,row,b"reasons");discard append(out,if(wire::count(t,reasons)==0){b"null"}else{model::raw(t,reasons)},limit,meter)?;return append(out,b"}",limit,meter);}
fn Result<bool,u32> context(ref mut bytes::Buffer out,ref join::Graph g,usize p,usize limit,ref mut usize meter){let part=join::partition(g,p);let t=validate::tree(part);let h=validate::header(part);discard append(out,b"{",limit,meter)?;
 if(wire::string(t,h,b"generic_interface_hash").len!=0){discard field(out,t,h,b"generic_interface_hash",b"generic_interface_hash",limit,meter)?;discard append(out,b",",limit,meter)?;}
 let names=b"logical_schema|compiler_build|language|package_uid|package_instance|package_coordinate|package_source|interface_hash|generation";var usize start=0;
 while(start<names.len){var usize end=start;while(end<names.len && names[end]!=124){end+=1;}let name=names[start..end];if(bytes::equal(name,b"logical_schema")){discard quoted(out,name,limit,meter)?;discard append(out,b":",limit,meter)?;discard quoted(out,wire::string(t,h,b"logical_schema"),limit,meter)?;}else{discard field(out,t,h,name,if(bytes::equal(name,b"generation")){b"generation_status"}else{name},limit,meter)?;}discard append(out,b",",limit,meter)?;start=end+1;}
 discard append(out,b"\"map_omissions\":",limit,meter)?;let omissions=wire::field(t,h,b"omissions");discard append(out,if(wire::count(t,omissions)==0){b"null"}else{model::raw(t,omissions)},limit,meter)?;discard append(out,b",\"completeness\":[",limit,meter)?;var usize i=0;var bool first=true;while(i<validate::completeness_count(part)){let c=validate::completeness(part,i);let relation=wire::string(t,c,b"relation");if(bytes::equal(relation,b"calls") || validate::version(part)==3 && (bytes::equal(relation,b"generic-specializations") || bytes::equal(relation,b"generic-template-bodies"))){if(!first){discard append(out,b",",limit,meter)?;}discard completeness(out,t,c,limit,meter)?;first=false;}i+=1;}return append(out,b"]}",limit,meter);
}
// Entire output is buffered before the caller can publish it; failure returns none.
pub fn Result<bytes::Buffer,u32> render(ref join::Graph g,ref impact::Impact r,usize limit,ref mut usize meter){if(limit==0 || limit>4194304){return .err(3);}var bytes::Buffer out=bytes::empty();discard append(ref mut out,b"{\"schema\":\"fairyc.atlas-impact/v1\",\"scope\":\"resolved-direct-call-reachability\",\"state\":",limit,meter)?;discard quoted(ref mut out,if(impact::partial(r)){b"partial"}else{b"bounded"},limit,meter)?;discard append(ref mut out,b",\"subject\":",limit,meter)?;discard symbol(ref mut out,g,impact::subject(r),limit,meter)?;
 let implementation=impact::subject_implementation(r);
 if(implementation!=0){
  discard append(ref mut out,b",\"subjectSpecialization\":",limit,meter)?;
  discard quoted(ref mut out,join::specialization(g,impact::subject(r).package,implementation),limit,meter)?;
 }
 discard append(ref mut out,b",\"nodes\":[",limit,meter)?;
 var usize i=0;while(i<impact::count(r)){if(i>0){discard append(ref mut out,b",",limit,meter)?;}let node=impact::node(r,i);discard append(ref mut out,b"{\"subject\":",limit,meter)?;discard symbol(ref mut out,g,node.subject,limit,meter)?;discard append(ref mut out,b",\"depth\":",limit,meter)?;discard number(ref mut out,node.depth,limit,meter)?;discard append(ref mut out,b"}",limit,meter)?;i+=1;}
 discard append(ref mut out,b"],\"contexts\":[",limit,meter)?;i=0;while(i<join::package_count(g)){if(i>0){discard append(ref mut out,b",",limit,meter)?;}discard context(ref mut out,g,join::package_at(g,i),limit,meter)?;i+=1;}
 discard append(ref mut out,b"],\"traversalComplete\":",limit,meter)?;discard append(ref mut out,if(impact::complete(r)){b"true"}else{b"false"},limit,meter)?;discard append(ref mut out,b",\"omissions\":[\"non-call-dependencies-not-analyzed\",\"external-callback-and-dynamic-targets-not-inferred\",\"outside-workspace-callers-not-analyzed\"",limit,meter)?;if(impact::depth_cut(r)){discard append(ref mut out,b",\"depth-limit\"",limit,meter)?;}if(impact::symbol_cut(r)){discard append(ref mut out,b",\"symbol-limit\"",limit,meter)?;}discard append(ref mut out,b"]}",limit,meter)?;return .ok(out);
}
