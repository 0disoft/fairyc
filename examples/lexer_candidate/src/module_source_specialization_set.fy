module module_source_specialization_set;
import generic_source_plan as plans;
import generic_source_plan_input as input;
import generic_source_catalog as templates;
import generic_package_bindings as bindings;
import project_cache_encoding as budget;
import std::bytes as bytes;
import std::vec as v;

struct Text{usize start;usize end;}
fn bool valid(ref u8[] id){if(id.len!=67 || !bytes::equal(id[..3],b"g0_")){return false;}var usize i=3;while(i<67){let c=id[i];if(!(c>=48 && c<=57 || c>=97 && c<=102)){return false;}i+=1;}return true;}
fn u32 compare(ref u8[] a,ref u8[] b,ref mut usize work){var usize i=0;while(i<a.len && i<b.len){if(!budget::charge(work,1)){return 10;}if(a[i]!=b[i]){return if(a[i]<b[i]){1}else{2};}i+=1;}return if(a.len==b.len){0}else if(a.len<b.len){1}else{2};}
fn u32 insert(ref mut bytes::Buffer names,ref mut v::Vec<Text> rows,ref u8[] id,ref mut usize work){
    if(!budget::charge(work,id.len+1)){return 10;}if(!valid(id)){return 9;}var usize low=0;var usize high=v::len<Text>(rows);
    while(low<high){let mid=low+(high-low)/2;let row=*v::at<Text>(rows,mid);let order=compare(id,bytes::view(names)[row.start..row.end],work);if(order==10){return 10;}if(order==0){return 0;}if(order==1){high=mid;}else{low=mid+1;}}
    let length=v::len<Text>(rows);if(length>=4096){return 10;}if(!budget::charge(work,length-low+id.len+1)){return 10;}let start=bytes::len(names);if(!bytes::append(names,id).is_ok){return 11;}let added=Text{.start,.end=bytes::len(names)};if(!v::push<Text>(rows,added).is_ok){return 11;}var usize at=length;while(at>low){let previous=*v::at<Text>(rows,at-1);let slot=v::at_mut<Text>(rows,at);*slot=previous;at-=1;}let slot=v::at_mut<Text>(rows,low);*slot=added;return 0;
}
// Original package membership includes concrete declarations lowered into SDK
// intrinsics. A completed owning compiler must authenticate this plan against
// its checked IR before retaining these facts; this helper grants no reuse.
pub fn Result<v::Vec<bytes::Buffer>,u32> collect(ref plans::Plan plan,usize owner,ref mut usize work){
    if(owner>=512 || plans::count(plan)>4096){return .err(9);}if(!budget::charge(work,0)){return .err(10);}let catalog=input::templates(plans::input(plan));let packages=templates::packages(catalog);var bytes::Buffer names=bytes::empty();var v::Vec<Text> rows=v::empty<Text>();var usize i=0;
    while(i<plans::count(plan)){if(!budget::charge(work,1)){return .err(10);}let node=plans::at(plan,i);if(node.state!=2 || node.template_index>=templates::count(catalog)){return .err(9);}let template=templates::template_at(catalog,node.template_index);if(template.module_index>=bindings::count(packages)){return .err(9);}if(bindings::module_at(packages,template.module_index).owner==owner){let code=insert(ref mut names,ref mut rows,plans::text(plan,node.id),work);if(code!=0){return .err(code);}}i+=1;}
    var v::Vec<bytes::Buffer> output=v::empty<bytes::Buffer>();i=0;while(i<v::len<Text>(ref rows)){let row=*v::at<Text>(ref rows,i);let id=bytes::view(ref names)[row.start..row.end];if(!budget::charge(work,id.len+1)){return .err(10);}let b=switch(bytes::copy(id)){case .ok(b):{b}case .err(_):{return .err(11);}};if(!v::push<bytes::Buffer>(ref mut output,b).is_ok){return .err(11);}i+=1;}return .ok(output);
}
pub fn Result<bool,u32> contains(ref v::Vec<bytes::Buffer> ids,ref u8[] id,ref mut usize work){if(!budget::charge(work,id.len+1)){return .err(10);}if(!valid(id)){return .err(9);}var usize low=0;var usize high=v::len<bytes::Buffer>(ids);while(low<high){let mid=low+(high-low)/2;let order=compare(id,bytes::view(v::at<bytes::Buffer>(ids,mid)),work);if(order==10){return .err(10);}if(order==0){return .ok(true);}if(order==1){high=mid;}else{low=mid+1;}}return .ok(false);}
