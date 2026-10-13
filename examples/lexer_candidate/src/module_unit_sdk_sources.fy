module module_unit_sdk_sources;
import sdk_authority as sdk;
import sdk_scopes as scopes;
import candidate_limits as limits;
import syntax as syntax;
import module_unit_cleanup_mir_model as clean;
import module_unit_value_model as budget;
import std::bytes as bytes;
import std::vec as v;

pub struct Unit {usize module_index;usize sdk_id;usize start;usize end;}
// Retain exact bundled bytes, not a caller-supplied digest or authority flag.
// Empty is the legacy authority-free constructor; lowered programs retain
// their exact source count, including user modules whose bytes are not copied.
pub resource Sources {usize modules;v::Vec<Unit> units;bytes::Buffer source;usize work;cleanup{}}
pub fn usize count(ref Sources p){return v::len<Unit>(ref p.units);}
pub fn usize module_count(ref Sources p){return p.modules;}
pub fn Unit at(ref Sources p,usize index){return *v::at<Unit>(ref p.units,index);}
pub fn ref u8[] data(ref Sources p){return bytes::view(ref p.source);}
pub fn ref u8[] source_at(ref Sources p,usize index){let u=at(p,index);return data(p)[u.start..u.end];}
pub fn usize work(ref Sources p){return p.work;}
pub fn usize find(ref Sources p,usize module_index){var usize i=0;while(i<count(p)){if(at(p,i).module_index==module_index){return i+1;}i+=1;}return 0;}
fn clean::Problem fail(u32 code,usize module_index,usize start,usize end){return clean::Problem{.code,.module_index,.start,.end};}
pub fn Sources empty(){return Sources{.modules=0,.units=v::empty<Unit>(),.source=bytes::empty(),.work=0};}
pub fn clean::Problem check(ref Sources p,ref mut usize used){
    if(module_count(p)>limits::source_files() || count(p)>8 || data(p).len>8*limits::source_bytes() || work(p)>1048576 || *used>1048576){return fail(1002,0,0,0);}
    var usize end=0;var usize previous=0;var u8[8] seen=[0;8];var usize i=0;
    while(i<count(p)){if(!budget::charge(used,1)){return fail(1002,0,0,0);}let u=at(p,i);
        if(u.module_index>=module_count(p) || i!=0 && u.module_index<=previous || u.sdk_id==0 || u.sdk_id>8 || u.start!=end || u.start>=u.end || u.end>data(p).len || u.end-u.start>limits::source_bytes()){return fail(1001,u.module_index,u.start,u.end);}
        let slot=u.sdk_id-1;if(seen[slot]!=0){return fail(1001,u.module_index,u.start,u.end);}seen[slot]=1;
        if(!budget::charge(used,u.end-u.start)){return fail(1002,u.module_index,u.start,u.end);}
        if(!sdk::authentic(scopes::name(u.sdk_id),source_at(p,i))){return fail(1001,u.module_index,u.start,u.end);}end=u.end;previous=u.module_index;i+=1;
    }
    if(end!=data(p).len){return fail(1001,0,end,data(p).len);}return fail(0,0,0,0);
}
pub fn Result<Sources,clean::Problem> assemble(usize modules,v::Vec<Unit> units,bytes::Buffer source,usize work){let p=Sources{.modules,.units,.source,.work};var usize used=work;let checked=check(ref p,ref mut used);if(checked.code!=0){return .err(checked);}return .ok(p);}
// Stored bytes never carry producer history or an authentication flag. Restore
// a fresh owner only after the ordinary checker authenticates its exact bytes.
pub fn Result<Sources,clean::Problem> assemble_artifact(usize modules,v::Vec<Unit> units,bytes::Buffer source,ref mut usize verification_work){let p=Sources{.modules,.units,.source,.work=0};let checked=check(ref p,verification_work);if(checked.code!=0){return .err(checked);}return .ok(p);}
pub fn Result<Sources,clean::Problem> collect(ref v::Vec<syntax::Parsed> sources,ref mut usize work){
    let base=*work;let modules=v::len<syntax::Parsed>(sources);if(modules>limits::source_files()){return .err(fail(1002,0,0,0));}var v::Vec<Unit> units=v::empty<Unit>();var bytes::Buffer owned=bytes::empty();var usize i=0;
    while(i<modules){if(!budget::charge(work,1)){return .err(fail(1002,i,0,0));}let p=v::at<syntax::Parsed>(sources,i);let raw=syntax::source_view(p);
        if(raw.len>limits::source_bytes()){return .err(fail(1002,i,0,raw.len));}if(syntax::module_count(p)!=1 || syntax::declaration_count(p)==0){return .err(fail(1001,i,0,raw.len));}let d=syntax::declaration_at(p,0);
        if(d.kind!=1 || d.name_start>d.name_end || d.name_end>raw.len){return .err(fail(1001,i,0,raw.len));}let name=raw[d.name_start..d.name_end];let id=sdk::module_id(name);
        if(sdk::reserved(name)){
            if(!budget::charge(work,raw.len)){return .err(fail(1002,i,0,raw.len));}if(id==0 || !sdk::authentic(name,raw)){return .err(fail(6801,i,0,raw.len));}
            var usize j=0;while(j<v::len<Unit>(ref units)){if(!budget::charge(work,1)){return .err(fail(1002,i,0,raw.len));}if(v::at<Unit>(ref units,j).sdk_id==id){return .err(fail(1001,i,0,raw.len));}j+=1;}
            if(v::len<Unit>(ref units)>=8){return .err(fail(1002,i,0,raw.len));}
            let start=bytes::len(ref owned);if(!bytes::append(ref mut owned,raw).is_ok || !v::push<Unit>(ref mut units,Unit{.module_index=i,.sdk_id=id,.start,.end=bytes::len(ref owned)}).is_ok){return .err(fail(1003,i,0,raw.len));}
        }i+=1;
    }return assemble(modules,units,owned,*work-base);
}
