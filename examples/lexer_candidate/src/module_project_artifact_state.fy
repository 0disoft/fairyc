module module_project_artifact_state;
import module_state_layout as state;
import module_state_declarations as declarations;
import module_catalog as catalog;
import module_program_artifact_wire as budget;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Match canonical saved state facts against actual current source declarations.
// Row order and storage offsets may differ after rebasing; field order may not.
// This detached index alone confers no Complete-owner or cache-hit authority.
pub resource Matched {v::Vec<usize> layouts;cleanup{}}
pub fn usize count(ref Matched p){return v::len<usize>(ref p.layouts);}
pub fn usize at(ref Matched p,usize index){return *v::at<usize>(ref p.layouts,index);}
fn u32 translated(u32 code){return if(code==10){1002}else if(code==11){1003}else{1001};}
fn u32 checked(ref state::Plan p,ref mut usize work){var usize measured=0;let code=state::validate(p,ref mut measured);if(!budget::charge(work,measured)){return 1002;}return if(code==0){0}else{translated(code)};}
fn u32 layout(ref state::Plan saved,usize target,ref state::Plan fresh,usize index,ref mut usize work){
    let a=state::at(fresh,index);let b=state::at(saved,target);if(!budget::charge(work,1)){return 1002;}
    if(a.owner!=b.owner || a.module_index!=b.module_index || a.declaration!=b.declaration || a.start!=b.start || a.end!=b.end || a.field_count!=b.field_count || !state::equal(a.hash,b.hash)){return 1001;}
    let uid=state::uid(fresh,index);if(!budget::charge(work,uid.len+1)){return 1002;}if(!bytes::equal(uid,state::uid(saved,target))){return 1001;}var usize i=0;
    while(i<a.field_count){let x=state::field_at(fresh,a.field_start+i);let y=state::field_at(saved,b.field_start+i);let text=state::name(fresh,a.field_start+i);if(!budget::charge(work,text.len+1)){return 1002;}
        if(x.kind!=y.kind || x.key!=y.key || x.value!=y.value || x.start!=y.start || x.end!=y.end || !state::equal(x.prefix,y.prefix) || !bytes::equal(text,state::name(saved,b.field_start+i))){return 1001;}i+=1;
    }return 0;
}
pub fn Result<Matched,u32> build(ref state::Plan saved,ref v::Vec<syntax::Parsed> sources,ref catalog::Catalog current,ref mut usize work){
    if(!budget::charge(work,0)){return .err(1002);}let valid=checked(saved,work);if(valid!=0){return .err(valid);}if(catalog::module_count(current)!=v::len<syntax::Parsed>(sources)){return .err(1001);}var usize i=0;
    while(i<v::len<syntax::Parsed>(sources)){let p=v::at<syntax::Parsed>(sources,i);if(!budget::charge(work,syntax::source_view(p).len+1)){return .err(1002);}if(!catalog::source_matches(current,i,p)){return .err(1001);}i+=1;}
    var usize measured=0;let fresh=switch(declarations::build_metered(sources,current,ref mut measured)){case .ok(p):{p}case .err(e):{if(!budget::charge(work,measured)){return .err(1002);}return .err(translated(e.code));}};
    if(!budget::charge(work,measured)){return .err(1002);}let fresh_valid=checked(ref fresh,work);if(fresh_valid!=0){return .err(fresh_valid);}if(state::count(saved)!=state::count(ref fresh) || state::field_count(saved)!=state::field_count(ref fresh)){return .err(1001);}
    var v::Vec<usize> rows=v::empty<usize>();var v::Vec<u8> used=v::empty<u8>();i=0;while(i<state::count(saved)){if(!budget::charge(work,1)){return .err(1002);}if(!v::push<u8>(ref mut used,0).is_ok){return .err(1003);}i+=1;}i=0;
    while(i<state::count(ref fresh)){let a=state::at(ref fresh,i);var usize target=0;var usize j=0;while(j<state::count(saved)){if(!budget::charge(work,1)){return .err(1002);}let b=state::at(saved,j);if(a.owner==b.owner && a.module_index==b.module_index && a.declaration==b.declaration){if(target!=0){return .err(1001);}target=j+1;}j+=1;}
        if(target==0 || *v::at<u8>(ref used,target-1)!=0){return .err(1001);}let code=layout(saved,target-1,ref fresh,i,work);if(code!=0){return .err(code);}if(!v::push<usize>(ref mut rows,target).is_ok){return .err(1003);}let cell=v::at_mut<u8>(ref mut used,target-1);*cell=1;i+=1;
    }
    i=0;while(i<v::len<u8>(ref used)){if(!budget::charge(work,1)){return .err(1002);}if(*v::at<u8>(ref used,i)!=1){return .err(1001);}i+=1;}return .ok(Matched{.layouts=rows});
}
