module module_project_artifact_sdk;
import module_unit_sdk_sources as sdk;
import sdk_authority as authority;
import module_catalog as catalog;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

pub struct Stats {usize modules;usize units;usize bytes;}
// This result records exact current SDK selection. Complete/cache publication
// still requires source/body/state admission under the same program owner.
pub resource Matched {Stats stats;cleanup{}}
pub fn Stats stats(ref Matched p){return p.stats;}
pub fn Result<Matched,u32> build(ref sdk::Sources saved,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog current,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}var usize measured=0;let valid=sdk::check(saved,ref mut measured);if(!budget::charge(work,measured)){return .err(1002);}if(valid.code!=0){return .err(valid.code);}
    let count=v::len<syntax::Parsed>(sources);if(sdk::module_count(saved)!=count || catalog::module_count(current)!=count){return .err(1001);}var usize index=0;var usize selected=0;var usize size=0;var u8[8] seen=[0;8];
    while(index<count){let p=v::at<syntax::Parsed>(sources,index);let raw=syntax::source_view(p);if(!budget::charge(work,raw.len+1)){return .err(1002);}if(!catalog::source_matches(current,index,p) || syntax::problem_count(p)!=0 || syntax::module_count(p)!=1 || syntax::declaration_count(p)==0){return .err(1001);}let d=syntax::declaration_at(p,0);
        if(d.kind!=1 || d.name_start>d.name_end || d.name_end>raw.len){return .err(1001);}let name=raw[d.name_start..d.name_end];let id=authority::module_id(name);
        if(authority::reserved(name)){if(!budget::charge(work,raw.len+name.len+1)){return .err(1002);}if(id==0 || seen[id-1]!=0 || !authority::authentic(name,raw) || selected>=sdk::count(saved)){return .err(1001);}seen[id-1]=1;let row=sdk::at(saved,selected);
            if(row.module_index!=index || row.sdk_id!=id){return .err(1001);}if(!budget::charge(work,raw.len+1)){return .err(1002);}if(!bytes::equal(raw,sdk::source_at(saved,selected))){return .err(1001);}selected+=1;size+=raw.len;
        }else if(selected<sdk::count(saved) && sdk::at(saved,selected).module_index==index){return .err(1001);}index+=1;
    }
    if(selected!=sdk::count(saved) || size!=sdk::data(saved).len){return .err(1001);}return .ok(Matched{.stats=Stats{.modules=count,.units=selected,.bytes=size}});
}
