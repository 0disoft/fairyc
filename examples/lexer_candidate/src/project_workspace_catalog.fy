module project_workspace_catalog;
import project_package_input as packages;
import project_source_modules as modules;
import project_source_set as sources;
import project_manifest as manifest;
import project_directory as directory;
import project_path as work;
import project_key_index as keys;
import std::bytes as bytes;
import std::vec as v;

resource Entry{modules::Modules input;bytes::Buffer digest;cleanup{}}
pub resource Catalog{v::Vec<Entry> entries;keys::Index paths;keys::Index uids;cleanup{}}
// 1 limit/work, 2 overlapping members, 3 member case collision, 4 nested
// workspace, 5 duplicate UID, 6 language mismatch, 7 member directory,
// 8 package-input failure, 9 digest, 10 allocation.
pub struct Problem{u32 code;u32 reason;u32 os_code;usize package_index;usize related_index;usize start;usize end;}
pub resource Error{Problem problem;bytes::Buffer path;bytes::Buffer related;v::Vec<packages::Error> input;cleanup{}}
fn Problem problem(u32 code,u32 reason,usize package_index,usize related_index){return Problem{.code,.reason,.os_code=0,.package_index,.related_index,.start=0,.end=0};}
fn Error fail(Problem p,ref u8[] path,ref u8[] related){var bytes::Buffer a=bytes::empty();var bytes::Buffer b=bytes::empty();if(!bytes::append(ref mut a,path).is_ok || !bytes::append(ref mut b,related).is_ok){return Error{.problem=problem(10,0,p.package_index,0),.path=bytes::empty(),.related=bytes::empty(),.input=v::empty<packages::Error>()};}return Error{.problem=p,.path=a,.related=b,.input=v::empty<packages::Error>()};}
fn Error input_failure(packages::Error e,ref u8[] path,usize at){var bytes::Buffer owned=bytes::empty();var v::Vec<packages::Error> errors=v::empty<packages::Error>();if(!bytes::append(ref mut owned,path).is_ok){return fail(problem(10,0,at,0),path,b"");}if(!v::push<packages::Error>(ref mut errors,e).is_ok){return fail(problem(10,0,at,0),path,b"");}return Error{.problem=problem(8,0,at,0),.path=owned,.related=bytes::empty(),.input=errors};}
pub fn Problem error(ref Error e){return e.problem;}
pub fn ref u8[] error_path(ref Error e){return bytes::view(ref e.path);}
pub fn ref u8[] related_path(ref Error e){return bytes::view(ref e.related);}
pub fn bool has_input_error(ref Error e){return v::len<packages::Error>(ref e.input)==1;}
pub fn ref packages::Error input_error(ref Error e){assert(has_input_error(e));return v::at<packages::Error>(ref e.input,0);}
pub fn ref str reason(u32 code){if(code==2){return "overlapping-workspace-member";}if(code==3){return "noncanonical-path-case";}if(code==4){return "nested-workspace";}if(code==5){return "duplicate-package-uid";}if(code==6){return "language-revision-mismatch";}if(code==8){return "invalid-package-input";}if(code==10){return "allocation-failed";}return "invalid-workspace-path";}
pub fn usize count(ref Catalog catalog){return v::len<Entry>(ref catalog.entries);}
pub fn ref u8[] path(ref Catalog catalog,usize at){return keys::key(ref catalog.paths,at);}
pub fn ref modules::Modules input(ref Catalog catalog,usize at){return ref v::at<Entry>(ref catalog.entries,at).input;}
pub fn ref manifest::Manifest package(ref Catalog catalog,usize at){return packages::package(input(catalog,at));}
pub fn ref u8[] digest(ref Catalog catalog,usize at){return bytes::view(ref v::at<Entry>(ref catalog.entries,at).digest);}
pub fn usize find(ref Catalog catalog,ref u8[] path){return keys::find(ref catalog.paths,path);}
pub fn Result<usize,u32> find_metered(ref Catalog catalog,ref u8[] path,ref mut usize meter){return keys::find_metered(ref catalog.paths,path,meter);}
pub fn usize find_uid(ref Catalog catalog,ref u8[] uid){return keys::find(ref catalog.uids,uid);}
fn Result<keys::Index,Error> members(ref manifest::Manifest root,ref mut usize meter){
    var bytes::Buffer names=bytes::empty();var v::Vec<keys::Span> spans=v::empty<keys::Span>();var bytes::Buffer lowered=bytes::empty();var v::Vec<keys::Span> lower_spans=v::empty<keys::Span>();if(!keys::append(ref mut names,ref mut spans,b".")){return .err(fail(problem(10,0,0,0),b".",b""));}var usize id=manifest::member_first(root);
    while(id!=0){let ref u8[] path=manifest::value(root,id);let usize at=v::len<keys::Span>(ref spans);if(at>=1024 || path.len>4095 || !work::charge(meter,path.len+1)){return .err(fail(problem(1,0,at,0),path,b""));}
        var bytes::Buffer lower=bytes::empty();switch(work::append_lower(ref mut lower,path,meter)){case .ok(_):{}case .err(code):{return .err(fail(problem(if(code==3){10}else{1},code,at,0),path,b""));}};
        if(!keys::append(ref mut names,ref mut spans,path) || !keys::append(ref mut lowered,ref mut lower_spans,bytes::view(ref lower))){return .err(fail(problem(10,0,at,0),path,b""));}id=manifest::next_value(root,id);
    }
    let keys::Index paths=switch(keys::build(names,spans,meter)){case .ok(value):{value}case .err(code):{return .err(fail(problem(if(code==1){10}else{1},code,0,0),b".",b""));}};
    let keys::Index folded=switch(keys::build(lowered,lower_spans,meter)){case .ok(value):{value}case .err(code):{return .err(fail(problem(if(code==1){10}else{1},code,0,0),b".",b""));}};
    var usize at=0;while(at<keys::count(ref folded)){let usize original=keys::ordered(ref folded,at);let ref u8[] value=keys::key(ref folded,original);
        if(at>0){let usize previous=keys::ordered(ref folded,at-1);if(bytes::equal(keys::key(ref folded,previous),value)){return .err(fail(problem(3,0,original+1,previous+1),keys::key(ref paths,original+1),keys::key(ref paths,previous+1)));}}
        var usize split=0;while(split<value.len){if(!work::charge(meter,1)){return .err(fail(problem(1,0,original+1,0),keys::key(ref paths,original+1),b""));}if(value[split]==47){let usize ancestor=switch(keys::find_metered(ref folded,value[..split],meter)){case .ok(value):{value}case .err(code):{return .err(fail(problem(1,code,original+1,0),keys::key(ref paths,original+1),b""));}};if(ancestor!=0){return .err(fail(problem(2,0,original+1,ancestor),keys::key(ref paths,original+1),keys::key(ref paths,ancestor)));}}split+=1;}at+=1;
    }
    return .ok(paths);
}
fn Result<Entry,Error> entry(modules::Modules loaded,ref u8[] path,usize at){let bytes::Buffer digest=switch(packages::source_digest(ref loaded)){case .ok(value):{value}case .err(code):{return .err(fail(problem(9,code,at,0),path,b""));}};return .ok(Entry{.input=loaded,.digest});}
@effects(filesystem)
pub fn Result<Catalog,Error> load(ref u8[] root,ref mut sources::Budget budget,ref mut usize meter){let selected=v::empty<bytes::Buffer>();return load_tests(root,ref selected,budget,meter);}
fn bool selected_test(ref v::Vec<bytes::Buffer> selected,ref u8[] path){var usize i=0;while(i<v::len<bytes::Buffer>(selected)){if(bytes::equal(bytes::view(v::at<bytes::Buffer>(selected,i)),path)){return true;}i+=1;}return false;}
@effects(filesystem)
pub fn Result<Catalog,Error> load_tests(ref u8[] root,ref v::Vec<bytes::Buffer> selected,ref mut sources::Budget budget,ref mut usize meter){
    var sources::Budget staged=*budget;let modules::Modules first=switch(packages::load_manifest_tests(root,selected_test(selected,b"."),ref mut staged,meter)){case .ok(value):{value}case .err(e):{return .err(input_failure(e,b".",0));}};
    let keys::Index paths=switch(members(packages::package(ref first),meter)){case .ok(value):{value}case .err(e):{return .err(e);}};var bytes::Buffer uid_names=bytes::empty();var v::Vec<keys::Span> uid_spans=v::empty<keys::Span>();if(!keys::append(ref mut uid_names,ref mut uid_spans,manifest::package_value(packages::package(ref first),b"uid"))){return .err(fail(problem(10,0,0,0),b".",b""));}
    var v::Vec<Entry> entries=v::empty<Entry>();let Entry first_entry=switch(entry(first,b".",0)){case .ok(value):{value}case .err(e):{return .err(e);}};if(!v::push<Entry>(ref mut entries,first_entry).is_ok){return .err(fail(problem(10,0,0,0),b".",b""));}
    var usize at=1;while(at<keys::count(ref paths)){let ref u8[] relative=keys::key(ref paths,at);let bytes::Buffer absolute=switch(directory::resolve(root,relative,meter)){case .ok(value):{value}case .err(e):{var Problem p=problem(7,e.code,at,0);p.os_code=e.os_code;p.start=e.start;p.end=e.end;return .err(fail(p,relative,b""));}};
        let modules::Modules loaded=switch(packages::load_manifest_tests(bytes::view(ref absolute),selected_test(selected,relative),ref mut staged,meter)){case .ok(value):{value}case .err(e):{return .err(input_failure(e,relative,at));}};let ref manifest::Manifest m=packages::package(ref loaded);
        if(manifest::workspace_present(m)){return .err(fail(problem(4,0,at,0),relative,b""));}if(!keys::append(ref mut uid_names,ref mut uid_spans,manifest::package_value(m,b"uid"))){return .err(fail(problem(10,0,at,0),relative,b""));}
        let Entry retained=switch(entry(loaded,relative,at)){case .ok(value):{value}case .err(e):{return .err(e);}};if(!v::push<Entry>(ref mut entries,retained).is_ok){return .err(fail(problem(10,0,at,0),relative,b""));}at+=1;
    }
    let keys::Index uids=switch(keys::build(uid_names,uid_spans,meter)){case .ok(value):{value}case .err(code):{return .err(fail(problem(if(code==2){1}else{10},code,0,0),b".",b""));}};
    var Catalog output=Catalog{.entries,.paths,.uids};at=1;while(at<count(ref output)){let usize previous=keys::ordered(ref output.uids,at-1);let usize current=keys::ordered(ref output.uids,at);if(bytes::equal(keys::key(ref output.uids,previous),keys::key(ref output.uids,current))){return .err(fail(problem(5,0,current,previous),path(ref output,current),path(ref output,previous)));}at+=1;}
    let ref u8[] language=manifest::package_value(package(ref output,0),b"language");if(!bytes::equal(language,b"fairyc-language/20")){return .err(fail(problem(6,0,0,0),b".",b""));}at=1;while(at<count(ref output)){if(!work::charge(meter,language.len+1)){return .err(fail(problem(1,0,at,0),path(ref output,at),b""));}if(!bytes::equal(language,manifest::package_value(package(ref output,at),b"language"))){return .err(fail(problem(6,0,at,0),path(ref output,at),b"."));}at+=1;}
    *budget=staged;return .ok(output);
}
