module project_source_set;
import project_source_root as roots;
import project_source_catalog as catalog;
import project_source_file as files;
import std::bytes as bytes;
import std::vec as v;
import project_manifest as manifest;

pub struct Budget{usize files;usize bytes;}
struct Span{usize start;usize end;}
pub resource Sources{roots::SourceRoot root;catalog::Catalog paths;bytes::Buffer contents;v::Vec<Span> spans;cleanup{}}
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
pub fn ref u8[] path(ref Sources sources,usize index){return catalog::path(ref sources.paths,index);}
pub fn ref u8[] source(ref Sources sources,usize index){if(index>=count(sources)){return bytes::view(ref sources.contents)[..0];}let Span span=*v::at<Span>(ref sources.spans,index);return bytes::view(ref sources.contents)[span.start..span.end];}
pub fn ref u8[] package_value(ref Sources sources,ref u8[] key){return roots::package_value(ref sources.root,key);}
pub fn ref u8[] source_root_path(ref Sources sources){return roots::path(ref sources.root);}
pub fn ref manifest::Manifest typed_manifest(ref Sources sources){return roots::typed_manifest(ref sources.root);}

@effects(filesystem)
pub fn Result<Sources,Error> load(ref u8[] root,ref u8[] manifest,ref mut Budget budget,ref mut usize work){
    let Budget initial=*budget;if(initial.files>4096 || initial.bytes>67108864){return .err(fail(problem(5,1,0,0,0,0),root));}
    let roots::SourceRoot bound=switch(roots::load(root,manifest,work)){case .ok(value):{value}case .err(e):{return .err(fail(problem(e.phase,e.reason,e.field,e.os_code,e.start,e.end),if(e.phase==1){b"fairy.toml"}else{root}));}};
    var usize file_count=initial.files;let catalog::Catalog discovered=switch(catalog::collect(roots::path(ref bound),ref mut file_count,work)){case .ok(value):{value}case .err(e):{return .err(fail(problem(3,catalog::error_code(ref e),0,catalog::error_os_code(ref e),0,0),catalog::error_path(ref e)));}};
    var bytes::Buffer contents=bytes::empty();var v::Vec<Span> spans=v::empty<Span>();var usize index=0;
    while(index<catalog::count(ref discovered)){
        let ref u8[] path=catalog::path(ref discovered,index);let bytes::Buffer input=switch(files::read(roots::path(ref bound),path,67108864-initial.bytes-bytes::len(ref contents),work)){case .ok(value):{value}case .err(e):{return .err(fail(problem(4,e.code,0,e.os_code,e.start,e.end),path));}};
        let usize start=bytes::len(ref contents);if(!bytes::append(ref mut contents,bytes::view(ref input)).is_ok || !v::push<Span>(ref mut spans,Span{.start,.end=bytes::len(ref contents)}).is_ok){return .err(fail(problem(5,2,0,0,0,0),path));}index+=1;
    }
    let usize byte_count=initial.bytes+bytes::len(ref contents);*budget=Budget{.files=file_count,.bytes=byte_count};return .ok(Sources{.root=bound,.paths=discovered,.contents,.spans});
}

// The manifest file uses the same exact-spelling/reparse/identity read boundary.
// Its 1 MiB parsing input cap is separate from the aggregate source byte budget.
@effects(filesystem)
pub fn Result<Sources,Error> load_manifest(ref u8[] root,ref mut Budget budget,ref mut usize work){
    let bytes::Buffer manifest=switch(files::read(root,b"fairy.toml",1048576,work)){case .ok(value):{value}case .err(e):{return .err(fail(problem(0,e.code,0,e.os_code,e.start,e.end),b"fairy.toml"));}};return load(root,bytes::view(ref manifest),budget,work);
}
