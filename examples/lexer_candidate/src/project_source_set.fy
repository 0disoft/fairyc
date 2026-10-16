module project_source_set;
import project_source_root as roots;
import project_source_catalog as catalog;
import project_source_file as files;
import std::bytes as bytes;
import std::vec as v;
import project_manifest as manifest;
import project_directory as directory;
import project_manifest_values as rules;
import project_path as paths;
import std::fs as fs;
import std::text as text;

pub struct Budget{usize files;usize bytes;}
struct Span{usize start;usize end;}
pub resource Sources{roots::SourceRoot root;catalog::Catalog paths;bytes::Buffer contents;v::Vec<Span> spans;bytes::Buffer test_names;v::Vec<Span> test_paths;v::Vec<usize> order;cleanup{}}
// 0 manifest-file read, 1 manifest validation, 2 source-root resolution,
// 3 path catalog, 4 source-file read retain their leaf's reason/offset codes.
// 5 aggregate: reason 1 invalid input budget, 2 allocation.
pub resource Error{u32 phase;u32 reason;u32 field;u32 os_code;usize start;usize end;bytes::Buffer path;cleanup{}}
pub struct Problem{u32 phase;u32 reason;u32 field;u32 os_code;usize start;usize end;}
fn Error fail(Problem problem,ref u8[] path){var bytes::Buffer copy=bytes::empty();if(!bytes::append(ref mut copy,path).is_ok){return Error{.phase=5,.reason=2,.field=0,.os_code=0,.start=0,.end=0,.path=copy};}return Error{.phase=problem.phase,.reason=problem.reason,.field=problem.field,.os_code=problem.os_code,.start=problem.start,.end=problem.end,.path=copy};}
fn Problem problem(u32 phase,u32 reason,u32 field,u32 os_code,usize start,usize end){return Problem{.phase,.reason,.field,.os_code,.start,.end};}
pub fn Problem error(ref Error error){return problem(error.phase,error.reason,error.field,error.os_code,error.start,error.end);}
pub fn ref u8[] error_path(ref Error error){return bytes::view(ref error.path);}
pub fn usize count(ref Sources sources){return v::len<Span>(ref sources.spans);}
pub fn ref u8[] path(ref Sources sources,usize index){let at=if(v::len<usize>(ref sources.order)==0){index}else{*v::at<usize>(ref sources.order,index)};return raw_path(sources,at);}
pub fn ref u8[] source(ref Sources sources,usize index){if(index>=count(sources)){return bytes::view(ref sources.contents)[..0];}let at=if(v::len<usize>(ref sources.order)==0){index}else{*v::at<usize>(ref sources.order,index)};let Span span=*v::at<Span>(ref sources.spans,at);return bytes::view(ref sources.contents)[span.start..span.end];}
pub fn ref u8[] package_value(ref Sources sources,ref u8[] key){return roots::package_value(ref sources.root,key);}
pub fn ref u8[] source_root_path(ref Sources sources){return roots::path(ref sources.root);}
pub fn ref manifest::Manifest typed_manifest(ref Sources sources){return roots::typed_manifest(ref sources.root);}
pub fn ref u8[] manifest_input(ref Sources sources){return roots::manifest_input(ref sources.root);}

@effects(filesystem)
pub fn Result<Sources,Error> load(ref u8[] root,ref u8[] manifest,ref mut Budget budget,ref mut usize work){return load_tests(root,manifest,false,budget,work);}
@effects(filesystem)
pub fn Result<Sources,Error> load_tests(ref u8[] root,ref u8[] manifest,bool tests,ref mut Budget budget,ref mut usize work){
    let Budget initial=*budget;if(initial.files>4096 || initial.bytes>67108864){return .err(fail(problem(5,1,0,0,0,0),root));}
    let roots::SourceRoot bound=switch(roots::load(root,manifest,work)){case .ok(value):{value}case .err(e):{return .err(fail(problem(e.phase,e.reason,e.field,e.os_code,e.start,e.end),if(e.phase==1){b"fairy.toml"}else{root}));}};
    var usize file_count=initial.files;let catalog::Catalog discovered=switch(catalog::collect(roots::path(ref bound),ref mut file_count,work)){case .ok(value):{value}case .err(e):{return .err(fail(problem(3,catalog::error_code(ref e),0,catalog::error_os_code(ref e),0,0),catalog::error_path(ref e)));}};
    var bytes::Buffer contents=bytes::empty();var v::Vec<Span> spans=v::empty<Span>();var usize index=0;
    while(index<catalog::count(ref discovered)){
        let ref u8[] path=catalog::path(ref discovered,index);let bytes::Buffer input=switch(files::read(roots::path(ref bound),path,67108864-initial.bytes-bytes::len(ref contents),work)){case .ok(value):{value}case .err(e):{return .err(fail(problem(4,e.code,0,e.os_code,e.start,e.end),path));}};
        let usize start=bytes::len(ref contents);if(!bytes::append(ref mut contents,bytes::view(ref input)).is_ok || !v::push<Span>(ref mut spans,Span{.start,.end=bytes::len(ref contents)}).is_ok){return .err(fail(problem(5,2,0,0,0,0),path));}index+=1;
    }
    var bytes::Buffer test_names=bytes::empty();var v::Vec<Span> test_paths=v::empty<Span>();var v::Vec<usize> order=v::empty<usize>();
    if(tests){var Budget staged=Budget{.files=file_count,.bytes=initial.bytes+bytes::len(ref contents)};switch(collect_tests(ref bound,root,ref mut contents,ref mut spans,ref mut test_names,ref mut test_paths,ref mut staged,work)){case .ok(_):{}case .err(e):{return .err(e);}};file_count=staged.files;
        var usize i=0;while(i<v::len<Span>(ref spans)){if(!v::push<usize>(ref mut order,i).is_ok){return .err(fail(problem(5,2,0,0,0,0),b"tests"));}i+=1;}
        i=1;while(i<v::len<Span>(ref spans)){let value=*v::at<usize>(ref order,i);var usize j=i;while(j>0){let before=*v::at<usize>(ref order,j-1);if(!rules::before(raw_parts(ref discovered,ref test_names,ref test_paths,value),raw_parts(ref discovered,ref test_names,ref test_paths,before))){break;}let slot=v::at_mut<usize>(ref mut order,j);*slot=before;j-=1;}let slot=v::at_mut<usize>(ref mut order,j);*slot=value;i+=1;}
    }
    let usize byte_count=initial.bytes+bytes::len(ref contents);*budget=Budget{.files=file_count,.bytes=byte_count};return .ok(Sources{.root=bound,.paths=discovered,.contents,.spans,.test_names,.test_paths,.order});
}

// The manifest file uses the same exact-spelling/reparse/identity read boundary.
// Its 1 MiB parsing input cap is separate from the aggregate source byte budget.
@effects(filesystem)
pub fn Result<Sources,Error> load_manifest(ref u8[] root,ref mut Budget budget,ref mut usize work){
    let bytes::Buffer manifest=switch(files::read(root,b"fairy.toml",1048576,work)){case .ok(value):{value}case .err(e):{return .err(fail(problem(0,e.code,0,e.os_code,e.start,e.end),b"fairy.toml"));}};return load(root,bytes::view(ref manifest),budget,work);
}


fn ref u8[] raw_path(ref Sources s,usize at){let n=catalog::count(ref s.paths);if(at<n){return catalog::path(ref s.paths,at);}let p=*v::at<Span>(ref s.test_paths,at-n);return bytes::view(ref s.test_names)[p.start..p.end];}
pub fn bool is_test(ref Sources s,usize index){let at=if(v::len<usize>(ref s.order)==0){index}else{*v::at<usize>(ref s.order,index)};return at>=catalog::count(ref s.paths);}
fn bool overlaps(ref u8[] a,ref u8[] b){return bytes::equal(a,b) || a.len<b.len && b[a.len]==47 && bytes::equal(a,b[..a.len]) || b.len<a.len && a[b.len]==47 && bytes::equal(b,a[..b.len]);}

fn ref u8[] raw_parts(ref catalog::Catalog paths,ref bytes::Buffer names,ref v::Vec<Span> spans,usize at){let n=catalog::count(paths);if(at<n){return catalog::path(paths,at);}let p=*v::at<Span>(spans,at-n);return bytes::view(names)[p.start..p.end];}
@effects(filesystem)
fn Result<usize,Error> collect_tests(ref roots::SourceRoot bound,ref u8[] root,ref mut bytes::Buffer contents,ref mut v::Vec<Span> spans,ref mut bytes::Buffer test_names,ref mut v::Vec<Span> test_paths,ref mut Budget budget,ref mut usize meter){
    if(overlaps(roots::package_value(bound,b"source"),b"tests")){return .err(fail(problem(3,9,0,0,0,0),b"tests"));}
    let m=roots::typed_manifest(bound);var usize member=manifest::member_first(m);while(member!=0){if(overlaps(manifest::value(m,member),b"tests")){return .err(fail(problem(3,9,0,0,0,0),b"tests"));}member=manifest::next_value(m,member);}
    let directory_name=switch(text::validate_utf8(root)){case .ok(p):{p}case .err(_):{return .err(fail(problem(3,8,0,0,0,0),b"tests"));}};
    var fs::ReadDir stream=switch(fs::read_dir_utf8(directory_name)){case .ok(p):{p}case .err(e):{return .err(fail(problem(3,1,0,e.os_code,0,0),b"tests"));}};
    var u8[4096] name=[0;4096];var usize matches=0;var bool exact=false;var bool reparse=false;var bool folder=false;
    while(true){let e=switch(fs::next_dir(ref mut stream,ref mut name)){case .ok(p):{p}case .err(e):{return .err(fail(problem(3,1,0,e.os_code,0,0),b"tests"));}};if(!e.has_entry){break;}let ref u8[] view=ref name;let spelling=view[..e.name_length];let same=switch(paths::equal_fold(spelling,b"tests",meter)){case .ok(p):{p}case .err(_):{return .err(fail(problem(3,6,0,0,0,0),b"tests"));}};if(same){matches+=1;exact=bytes::equal(spelling,b"tests");reparse=e.is_reparse;folder=fs::path_kind_is_directory(e.kind);}}
    switch(fs::close_dir(stream)){case .ok(_):{}case .err(e):{return .err(fail(problem(3,1,0,e.os_code,0,0),b"tests"));}};
    if(matches==0){return .ok(0);}if(matches!=1 || !exact || reparse || !folder){return .err(fail(problem(3,if(reparse){2}else{9},0,0,0,0),b"tests"));}
    let test_root=switch(directory::resolve(root,b"tests",meter)){case .ok(p):{p}case .err(e):{return .err(fail(problem(3,9,0,e.os_code,0,0),b"tests"));}};
    var usize file_count=budget.files;let found=switch(catalog::collect_optional(bytes::view(ref test_root),true,ref mut file_count,meter)){case .ok(p):{p}case .err(e):{return .err(fail(problem(3,catalog::error_code(ref e),0,catalog::error_os_code(ref e),0,0),catalog::error_path(ref e)));}};
    var usize i=0;while(i<catalog::count(ref found)){let p=catalog::path(ref found,i);let body=switch(files::read(bytes::view(ref test_root),p,67108864-budget.bytes,meter)){case .ok(p):{p}case .err(e):{return .err(fail(problem(4,e.code,0,e.os_code,e.start,e.end),p));}};
        let start=bytes::len(test_names);let body_start=bytes::len(contents);
        if(!bytes::append(test_names,b"tests/").is_ok || !bytes::append(test_names,p).is_ok || !v::push<Span>(test_paths,Span{.start,.end=bytes::len(test_names)}).is_ok || !bytes::append(contents,bytes::view(ref body)).is_ok || !v::push<Span>(spans,Span{.start=body_start,.end=bytes::len(contents)}).is_ok){return .err(fail(problem(5,2,0,0,0,0),p));}budget.bytes+=bytes::len(ref body);i+=1;
    }budget.files=file_count;return .ok(i);
}
