module module_public_type_ids;
import module_types as types;
import module_public_interface_model as model;
import module_unit_interfaces as typed;
import module_signatures as signatures;
import module_state_layout as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Detached type spelling facts, not executable or cache publication authority.
pub struct Row{types::Type type;usize owner;model::Text name;}
pub struct Stamp{usize owner;u32[8] digest;}
pub resource Table{bytes::Buffer names;v::Vec<Row> rows;v::Vec<Stamp> stamps;cleanup{}}
pub fn Table assemble(bytes::Buffer names,v::Vec<Row> rows,v::Vec<Stamp> stamps){return Table{.names,.rows,.stamps};}
pub fn Table empty(){return Table{.names=bytes::empty(),.rows=v::empty<Row>(),.stamps=v::empty<Stamp>()};}
pub fn usize count(ref Table p){return v::len<Row>(ref p.rows);}
pub fn Row at(ref Table p,usize id){assert(id>0 && id<=count(p));return *v::at<Row>(ref p.rows,id-1);}
pub fn bool same(types::Type a,types::Type b){return a.kind==b.kind && a.scalar==b.scalar && a.module_index==b.module_index && a.declaration==b.declaration && a.first==b.first && a.second==b.second;}
pub fn bool valid(ref u8[] text){let prefix=b"std::vec::_fy_generic_g0_";if(text.len!=prefix.len+64 || !bytes::equal(text[..prefix.len],prefix)){return false;}var usize i=prefix.len;while(i<text.len){let n=text[i];if(!(n>=48 && n<=57 || n>=97 && n<=102)){return false;}i+=1;}return true;}
pub fn Result<usize,u32> owner(ref Table p,usize id,types::Type actual){if(count(p)==0){return .err(4503);}if(id==0 || id>count(p)){return .err(9);}let row=at(p,id);if(!same(row.type,actual) || actual.kind!=13 || row.owner>=512 || row.name.start>row.name.end || row.name.end>bytes::len(ref p.names) || !valid(bytes::view(ref p.names)[row.name.start..row.name.end])){return .err(9);}return .ok(row.owner);}
pub fn ref u8[] name(ref Table p,usize id){let row=at(p,id);assert(row.name.start<=row.name.end && row.name.end<=bytes::len(ref p.names));return bytes::view(ref p.names)[row.name.start..row.name.end];}
pub fn u32 current(ref Table p,ref v::Vec<syntax::Parsed> sources,ref signatures::Program metadata,ref typed::Interfaces graph,ref mut usize work){
    if(count(p)==0 || count(p)>4096 || count(p)!=typed::type_count(graph) || v::len<Stamp>(ref p.stamps)!=v::len<syntax::Parsed>(sources) || v::len<syntax::Parsed>(sources)!=signatures::module_count(metadata) || v::len<Stamp>(ref p.stamps)>512 || bytes::len(ref p.names)>1048576){return 9;}var usize i=0;
    while(i<v::len<Stamp>(ref p.stamps)){let stamp=v::at<Stamp>(ref p.stamps,i);let raw=syntax::source_view(v::at<syntax::Parsed>(sources,i));if(!budget::charge(work,raw.len+1)){return 10;}if(stamp.owner!=signatures::module_owner(metadata,i)){return 9;}if(!signatures::source_matches(metadata,i,v::at<syntax::Parsed>(sources,i))){return 12;}let digest=signatures::source_digest(metadata,i);var usize j=0;while(j<8){if(digest[j]!=stamp.digest[j]){return 12;}j+=1;}i+=1;}
    i=1;while(i<=count(p)){if(!budget::charge(work,1)){return 10;}let actual=typed::type_at(graph,i);let row=at(p,i);if(!same(row.type,actual)){return 9;}if(actual.kind==13){let checked=switch(owner(p,i,actual)){case .ok(n):{n}case .err(code):{return code;}};if(actual.module_index>=signatures::module_count(metadata) || checked!=signatures::module_owner(metadata,actual.module_index)){return 9;}}else if(row.name.start!=row.name.end){return 9;}i+=1;}return 0;
}
