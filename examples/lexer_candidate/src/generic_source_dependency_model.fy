module generic_source_dependency_model;
import generic_source_catalog as catalog;
import project_path as budget;
import std::bytes as bytes;
import std::vec as v;

pub struct Text {usize start;usize end;}
pub struct Argument {Text name;bool parameter;}
pub struct Edge {usize target;usize first;usize count;Text key;}
pub struct Range {usize first;usize count;}
pub struct Problem {u32 code;usize module_index;usize declaration;usize start;usize end;}
pub resource Data {bytes::Buffer names;v::Vec<Argument> arguments;v::Vec<Edge> edges;v::Vec<Range> ranges;cleanup{}}
pub resource Graph {catalog::Catalog templates;Data data;cleanup{}}
pub fn Data empty(){return Data{.names=bytes::empty(),.arguments=v::empty<Argument>(),.edges=v::empty<Edge>(),.ranges=v::empty<Range>()};}
pub fn Graph assemble(catalog::Catalog templates,Data data){return Graph{.templates,.data};}
pub fn ref catalog::Catalog templates(ref Graph p){return ref p.templates;}
pub fn ref Data data(ref Graph p){return ref p.data;}
pub fn usize edge_count(ref Data p){return v::len<Edge>(ref p.edges);}
pub fn Edge edge_at(ref Data p,usize at){assert(at<edge_count(p));return *v::at<Edge>(ref p.edges,at);}
pub fn Argument argument_at(ref Data p,usize at){assert(at<v::len<Argument>(ref p.arguments));return *v::at<Argument>(ref p.arguments,at);}
pub fn Range range_at(ref Data p,usize at){assert(at<v::len<Range>(ref p.ranges));return *v::at<Range>(ref p.ranges,at);}
pub fn ref u8[] text(ref Data p,Text t){assert(t.start<=t.end && t.end<=bytes::len(ref p.names));return bytes::view(ref p.names)[t.start..t.end];}
pub fn usize argument_count(ref Data p){return v::len<Argument>(ref p.arguments);}
pub fn usize range_count(ref Data p){return v::len<Range>(ref p.ranges);}
pub fn bool valid_text(ref Data p,Text t){return t.start<t.end && t.end<=bytes::len(ref p.names);}
pub fn Data assemble_data(bytes::Buffer names,v::Vec<Argument> arguments,v::Vec<Edge> edges,v::Vec<Range> ranges){return Data{.names,.arguments,.edges,.ranges};}
fn Result<Text,u32> retain(ref mut bytes::Buffer names,ref u8[] value,ref mut usize work){if(value.len>1048576 || bytes::len(names)>1048576-value.len || !budget::charge(work,value.len+1)){return .err(10);}let start=bytes::len(names);if(!bytes::append(names,value).is_ok){return .err(11);}return .ok(Text{.start,.end=bytes::len(names)});}
pub fn u32 argument(ref mut bytes::Buffer names,ref mut v::Vec<Argument> arguments,ref u8[] name,bool parameter,ref mut usize work){if(v::len<Argument>(arguments)>=1048576){return 10;}let saved=switch(retain(names,name,work)){case .ok(value):{value}case .err(code):{return code;}};return if(v::push<Argument>(arguments,Argument{.name=saved,.parameter}).is_ok){0}else{11};}
// Calls append arguments temporarily. Duplicate requests roll them and text back.
pub fn u32 edge_bounded(ref mut bytes::Buffer names,ref mut v::Vec<Argument> arguments,ref mut v::Vec<Edge> edges,usize target,ref u8[] id,usize edge_first,usize argument_first,usize names_first,usize limit,ref mut usize work){
    let count=v::len<Argument>(arguments)-argument_first;var bytes::Buffer key=bytes::empty();if(!budget::charge(work,id.len+2)){return 10;}if(!bytes::append(ref mut key,id).is_ok || !bytes::append_byte(ref mut key,60).is_ok){return 11;}var usize i=0;
    while(i<count){let a=*v::at<Argument>(arguments,argument_first+i);let name=bytes::view(names)[a.name.start..a.name.end];if(!budget::charge(work,name.len+2)){return 10;}if(i!=0 && !bytes::append_byte(ref mut key,44).is_ok || a.parameter && !bytes::append_byte(ref mut key,36).is_ok || !bytes::append(ref mut key,name).is_ok){return 11;}i+=1;}if(!bytes::append_byte(ref mut key,62).is_ok){return 11;}
    i=edge_first;while(i<v::len<Edge>(edges)){let old=*v::at<Edge>(edges,i);if(!budget::charge(work,bytes::len(ref key)+1)){return 10;}if(bytes::equal(bytes::view(names)[old.key.start..old.key.end],bytes::view(ref key))){while(v::len<Argument>(arguments)>argument_first){discard v::pop<Argument>(arguments);}discard bytes::truncate(names,names_first);return 0;}i+=1;}
    if(v::len<Edge>(edges)-edge_first>=limit || v::len<Edge>(edges)>=131072){return 10;}let saved=switch(retain(names,bytes::view(ref key),work)){case .ok(value):{value}case .err(code):{return code;}};return if(v::push<Edge>(edges,Edge{.target,.first=argument_first,.count,.key=saved}).is_ok){0}else{11};
}
fn Result<bool,u32> less(ref bytes::Buffer names,Text a,Text b,ref mut usize work){let left=bytes::view(names)[a.start..a.end];let right=bytes::view(names)[b.start..b.end];var usize i=0;while(i<left.len && i<right.len){if(!budget::charge(work,1)){return .err(10);}if(left[i]!=right[i]){return .ok(left[i]<right[i]);}i+=1;}return .ok(left.len<right.len);}
pub fn u32 finish(ref mut v::Vec<Edge> edges,ref bytes::Buffer names,ref mut v::Vec<Range> ranges,usize first,ref mut usize work){var usize i=first+1;while(i<v::len<Edge>(edges)){let item=*v::at<Edge>(edges,i);var usize at=i;while(at>first){if(!budget::charge(work,1)){return 10;}let old=*v::at<Edge>(edges,at-1);let before=switch(less(names,item.key,old.key,work)){case .ok(value):{value}case .err(code):{return code;}};if(!before){break;}let slot=v::at_mut<Edge>(edges,at);*slot=old;at-=1;}let slot=v::at_mut<Edge>(edges,at);*slot=item;i+=1;}return if(v::push<Range>(ranges,Range{.first,.count=v::len<Edge>(edges)-first}).is_ok){0}else{11};}

pub fn u32 edge(ref mut bytes::Buffer names,ref mut v::Vec<Argument> arguments,ref mut v::Vec<Edge> edges,usize target,ref u8[] id,usize edge_first,usize argument_first,usize names_first,ref mut usize work){return edge_bounded(names,arguments,edges,target,id,edge_first,argument_first,names_first,32,work);}
