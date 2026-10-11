module tests::project_source_set;
import project_source_set as sources;
import file_source as files;
import std::bytes as bytes;
import std::process as process;
fn bytes::Buffer manifest(ref u8[] source){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,b"schema='fairyc.manifest/v1'\n[package]\nuid='example.catalog'\nname='catalog'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='").is_ok && bytes::append(ref mut out,source).is_ok && bytes::append(ref mut out,b"'\n").is_ok);return out;}
@effects(filesystem)
fn void rejected(ref u8[] root,ref u8[] source,u32 phase,u32 reason,sources::Budget initial){let document=manifest(source);var sources::Budget budget=initial;var usize work=0;switch(sources::load(root,bytes::view(ref document),ref mut budget,ref mut work)){case .ok(_):{assert(false);}case .err(e):{let problem=sources::error(ref e);assert(problem.phase==phase && problem.reason==reason && sources::error_path(ref e).len>0 && budget.files==initial.files && budget.bytes==initial.bytes);}};}
@test
@effects(filesystem,console)
fn void retains_source_set_and_commits_cumulative_budgets_only_on_success(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] argument_path=ref argument;
    let fixture=switch(files::manifest(argument_path[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref u8[] root=bytes::view(ref fixture);var bytes::Buffer document=manifest(b"Good");var sources::Budget budget=sources::Budget{.files=4093,.bytes=67108826};var usize work=0;
    let loaded=switch(sources::load(root,bytes::view(ref document),ref mut budget,ref mut work)){case .ok(value):{value}case .err(_):{assert(false);return;}};assert(bytes::resize_zeroed(ref mut document,0).is_ok);
    assert(budget.files==4096 && budget.bytes==67108864 && sources::count(ref loaded)==3);
    assert(bytes::equal(sources::path(ref loaded,0),b"a.fy") && bytes::equal(sources::path(ref loaded,1),b"nested/z.fy") && bytes::equal(sources::path(ref loaded,2),b"z.fy"));
    assert(bytes::equal(sources::source(ref loaded,0),b"module a;\n") && bytes::equal(sources::source(ref loaded,1),b"module nested::z;\n") && bytes::equal(sources::source(ref loaded,2),b"module z;\n"));
    assert(bytes::equal(sources::package_value(ref loaded,b"uid"),b"example.catalog") && bytes::equal(sources::package_value(ref loaded,b"source"),b"Good") && sources::source_root_path(ref loaded).len>root.len && bytes::equal(sources::source(ref loaded,3),b"") && bytes::equal(sources::path(ref loaded,3),b""));
    rejected(root,b"Good",4,4,sources::Budget{.files=0,.bytes=67108827});rejected(root,b"Good",3,6,sources::Budget{.files=4095,.bytes=0});rejected(root,b"Collision",3,4,sources::Budget{.files=1,.bytes=3});rejected(root,b"Reparse",3,2,sources::Budget{.files=1,.bytes=3});rejected(root,b"Empty",3,5,sources::Budget{.files=1,.bytes=3});rejected(root,b"Good",5,1,sources::Budget{.files=4097,.bytes=0});
    budget=sources::Budget{.files=2,.bytes=3};work=0;switch(sources::load(root,b"",ref mut budget,ref mut work)){case .ok(_):{assert(false);}case .err(e):{let problem=sources::error(ref e);assert(problem.phase==1 && problem.reason==3 && problem.field==1 && work==0 && budget.files==2 && budget.bytes==3);}};
    var bytes::Buffer package_root=bytes::empty();assert(bytes::append(ref mut package_root,root).is_ok && bytes::append(ref mut package_root,b"/Package").is_ok);budget=sources::Budget{.files=0,.bytes=0};work=0;
    switch(sources::load_manifest(bytes::view(ref package_root),ref mut budget,ref mut work)){case .ok(value):{assert(sources::count(ref value)==3 && budget.files==3 && budget.bytes==38 && bytes::equal(sources::package_value(ref value,b"source"),b"src") && bytes::equal(sources::source(ref value,1),b"module nested::z;\n"));}case .err(_):{assert(false);}};
}
