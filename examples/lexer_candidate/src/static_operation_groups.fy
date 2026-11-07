module static_operation_groups;
import static_operation_declarations as operations;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

struct Group{usize declaration;usize module_index;usize members;usize[13] masks;usize[13] first;}
struct Member{usize declaration;usize group;usize position;}
fn operations::Problem fail(u32 code,usize declaration){return operations::Problem{.code,.declaration};}
fn bool charge(ref mut usize work){if(*work>=1048576){return false;};*work+=1;return true;}
// Only the new grouped family gains declaration-specific errors. Existing
// ungrouped failures retain the old catalog fallback coordinate.
pub fn bool related(ref syntax::Parsed p,usize declaration){
    if(operations::grouped(p,declaration)){return true;}let d=syntax::declaration_at(p,declaration-1);let raw=syntax::source_view(p);var usize i=0;
    while(i<d.attribute_count){let a=syntax::attribute_at(p,d.attribute_start+i);if(a.argument_count!=0 && bytes::equal(raw[a.name_start..a.name_end],b"implements")){let argument=syntax::attribute_argument_at(p,a.argument_start);let name=raw[argument.start..argument.end];var usize j=1;while(j<syntax::declaration_count(p)){let target=syntax::declaration_at(p,j);if(operations::grouped(p,j+1) && (bytes::equal(name,raw[target.name_start..target.name_end]) || bytes::equal(name,operations::group(p,j+1)))){return true;}j+=1;}}i+=1;}return false;
}
fn usize scalar_slot(u32 code){return if(code<=6){(code-1) as usize}else{(code-32+6) as usize};}
pub fn operations::Problem check(ref syntax::Parsed p,ref mut usize work){
    var v::Vec<Group> groups=v::empty<Group>();var v::Vec<Member> members=v::empty<Member>();var usize i=1;
    while(i<syntax::declaration_count(p)){if(!charge(work)){return fail(4503,0);}let name=operations::group(p,i+1);if(name.len!=0){let module_index=syntax::module_index(p,syntax::declaration_at(p,i).start);var usize index=0;while(index<v::len<Group>(ref groups)){if(!charge(work)){return fail(4503,0);}let old=*v::at<Group>(ref groups,index);if(old.module_index==module_index && bytes::equal(name,operations::group(p,old.declaration))){break;}index+=1;}if(index==v::len<Group>(ref groups) && !v::push<Group>(ref mut groups,Group{.declaration=i+1,.module_index,.members=0,.masks=[0;13],.first=[0;13]}).is_ok){return fail(11,0);}let group=*v::at<Group>(ref groups,index);if(group.members<16 && !v::push<Member>(ref mut members,Member{.declaration=i+1,.group=index,.position=group.members}).is_ok){return fail(11,0);}{let target=v::at_mut<Group>(ref mut groups,index);if(group.members<17){target.members+=1;}}}i+=1;}
    i=0;while(i<v::len<Group>(ref groups)){if(!charge(work)){return fail(4503,0);}let group=*v::at<Group>(ref groups,i);if(group.members<2 || group.members>16){return fail(4501,group.declaration);}i+=1;}
    i=1;let usize one=1;while(i<syntax::declaration_count(p)){if(!charge(work)){return fail(4503,0);}let provider=operations::provider(p,i+1);if(provider.code!=0){return fail(provider.code,i+1);}if(provider.kind==2){var usize low=0;var usize high=v::len<Member>(ref members);while(low<high){if(!charge(work)){return fail(4503,0);}let mid=low+(high-low)/2;if(v::at<Member>(ref members,mid).declaration<provider.operation){low=mid+1;}else{high=mid;}}if(low<v::len<Member>(ref members) && v::at<Member>(ref members,low).declaration==provider.operation){let member=*v::at<Member>(ref members,low);let slot=scalar_slot(provider.scalar);let old=*v::at<Group>(ref groups,member.group);let mask=old.masks[slot]|(one<<member.position);let first=old.first[slot];{let group=v::at_mut<Group>(ref mut groups,member.group);group.masks[slot]=mask;if(first==0){group.first[slot]=i+1;}}}}i+=1;}
    var usize first=0;i=0;while(i<v::len<Group>(ref groups)){if(!charge(work)){return fail(4503,0);}let group=*v::at<Group>(ref groups,i);let expected=(one<<group.members)-1;var usize scalar=0;while(scalar<13){if(!charge(work)){return fail(4503,0);}if(group.masks[scalar]!=0 && group.masks[scalar]!=expected && (first==0 || group.first[scalar]<first)){first=group.first[scalar];}scalar+=1;}i+=1;}return if(first==0){fail(0,0)}else{fail(4501,first)};
}
