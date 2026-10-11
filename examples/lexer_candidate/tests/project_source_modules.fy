module tests::project_source_modules;
import project_source_modules as modules;
import project_module_path as paths;
import project_source_set as sources;
import file_source as files;
import syntax as syntax;
import std::bytes as bytes;
import std::process as process;
import std::vec as v;
import std::io as io;

@effects(console)
fn void checked(bool condition,ref str label){if(!condition){discard io::write(label);discard io::write("\n");}assert(condition);}
fn void expected(ref u8[] path,ref u8[] name){var usize work=0;switch(paths::expected(path,ref mut work)){case .ok(value):{assert(bytes::equal(bytes::view(ref value),name) && work>0);}case .err(_):{assert(false);}};}
fn void invalid(ref u8[] path){var usize work=0;switch(paths::expected(path,ref mut work)){case .ok(_):{assert(false);}case .err(code):{assert(code==1);}};}
@test
fn void derives_stage0_names_only_from_canonical_relative_paths(){
    expected(b"main.fy",b"main");expected(b"net/http.fy",b"net::http");expected(b"_private/A_0.fy",b"_private::A_0");expected(b"if.fy",b"if");
    invalid(b"../a.fy");invalid(b"a//b.fy");invalid(b"/a.fy");invalid(b"bad-name/a.fy");invalid(b"\xc3\x9c.fy");invalid(b"1a.fy");invalid(b"a.txt");invalid(b".fy");
    var usize work=16777216;switch(paths::expected(b"a.fy",ref mut work)){case .ok(_):{assert(false);}case .err(code):{assert(code==3);}};
}
fn bytes::Buffer manifest(ref u8[] source){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"schema='fairyc.manifest/v1'\n[package]\nuid='example.catalog'\nname='catalog'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='").is_ok && bytes::append(ref mut out,source).is_ok && bytes::append(ref mut out,b"'\n").is_ok);return out;}
@effects(filesystem,console)
fn void rejected(ref u8[] root,ref u8[] directory,u32 code,usize file){let document=manifest(directory);var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;let loaded=switch(sources::load(root,bytes::view(ref document),ref mut budget,ref mut work)){case .ok(value):{value}case .err(_):{checked(false,"module binding test line 22");return;}};
    switch(modules::bind(loaded,ref mut work)){case .ok(_):{checked(false,"module binding test line 23");}case .err(e):{let problem=modules::error(ref e);checked(problem.code==code && problem.file==file,"module binding test line 23");let original=modules::error_sources(ref e);checked(sources::count(original)>file && sources::path(original,file).len>0 && sources::source(original,file).len>0 && problem.start<=problem.end && problem.end<=sources::source(original,file).len,"module binding test line 23");if(code==1 || code==5){checked(problem.diagnostic==2 && problem.end>problem.start,"module binding test line 23");}}};
}
@test
@effects(filesystem,console)
fn void binds_retained_files_and_preserves_original_inputs_on_error(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{checked(false,"module binding test line 28");0}};let ref u8[] argument_path=ref argument;let fixture=switch(files::manifest(argument_path[..length])){case .ok(value):{value}case .err(_):{checked(false,"module binding test line 28");bytes::empty()}};let ref u8[] root=bytes::view(ref fixture);
    let document=manifest(b"Good");var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize work=0;let loaded=switch(sources::load(root,bytes::view(ref document),ref mut budget,ref mut work)){case .ok(value):{value}case .err(_):{checked(false,"module binding test line 29");return;}};
    let bound=switch(modules::bind(loaded,ref mut work)){case .ok(value):{value}case .err(_):{checked(false,"module binding test line 30");return;}};checked(modules::count(ref bound)==3 && modules::find(ref bound,b"a")==1 && modules::find(ref bound,b"nested::z")==2 && modules::find(ref bound,b"z")==3 && modules::find(ref bound,b"missing")==0,"module binding test line 30");
    checked(bytes::equal(modules::name(ref bound,1),b"nested::z") && bytes::equal(modules::path(ref bound,1),b"nested/z.fy") && bytes::equal(syntax::source_view(modules::parsed(ref bound,1)),b"module nested::z;\n") && v::len<syntax::Parsed>(modules::parser_inputs(ref bound))==3 && bytes::equal(modules::name(ref bound,3),b""),"module binding test line 31");
    let ordered_document=manifest(b"ModuleOrder");budget=sources::Budget{.files=0,.bytes=0};work=0;let ordered=switch(sources::load(root,bytes::view(ref ordered_document),ref mut budget,ref mut work)){case .ok(value):{value}case .err(_):{checked(false,"module binding test line 32");return;}};
    switch(modules::bind(ordered,ref mut work)){case .ok(value):{checked(modules::find(ref value,b"a::x")==1 && modules::find(ref value,b"a0")==2,"module binding test line 33");}case .err(_):{checked(false,"module binding test line 33");}};
    rejected(root,b"BadModule",5,0);rejected(root,b"BadCase",5,0);rejected(root,b"SecondModule",3,0);rejected(root,b"MissingModule",3,0);rejected(root,b"BrokenSyntax",3,0);rejected(root,b"InvalidUtf8",2,0);rejected(root,b"UnicodeFile",1,0);rejected(root,b"UnicodeDir",1,0);rejected(root,b"HyphenDir",1,0);rejected(root,b"DuplicateDeclared",5,1);
    let large_document=manifest(b"LargeModules");budget=sources::Budget{.files=0,.bytes=0};work=0;let large=switch(sources::load(root,bytes::view(ref large_document),ref mut budget,ref mut work)){case .ok(value):{value}case .err(_):{checked(false,"module binding test line 35");return;}};
    switch(modules::bind(large,ref mut work)){case .ok(value):{checked(modules::count(ref value)==1 && modules::find(ref value,b"large")==1 && syntax::source_view(modules::parsed(ref value,0)).len==65537,"module binding test line 36");}case .err(_):{checked(false,"module binding test line 36");}};
    let trivia_document=manifest(b"LongModuleTrivia");budget=sources::Budget{.files=0,.bytes=0};work=0;let trivia=switch(sources::load(root,bytes::view(ref trivia_document),ref mut budget,ref mut work)){case .ok(value):{value}case .err(_):{checked(false,"long module trivia source load");return;}};
    switch(modules::bind(trivia,ref mut work)){case .ok(value):{checked(modules::count(ref value)==1 && modules::find(ref value,b"net::http")==1 && syntax::source_view(modules::parsed(ref value,0)).len==70025,"long module trivia binding");}case .err(_):{checked(false,"long module trivia binding");}};
}
