module tests::project_package_input;
import project_package_input as packages;
import project_source_modules as modules;
import project_source_set as sources;
import project_manifest as manifest;
import file_source as files;
import std::bytes as bytes;
import std::process as process;

fn bytes::Buffer document(ref u8[] source){var bytes::Buffer result=bytes::empty();assert(bytes::append(ref mut result,b"schema='fairyc.manifest/v1'\n[package]\nuid='example.catalog'\nname='catalog'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='").is_ok && bytes::append(ref mut result,source).is_ok && bytes::append(ref mut result,b"'\n").is_ok);return result;}
@effects(filesystem)
fn void rejected(ref u8[] root,ref u8[] source,u32 phase,u32 reason,usize file){let input=document(source);var sources::Budget budget=sources::Budget{.files=7,.bytes=13};var usize work=0;
    switch(packages::load(root,bytes::view(ref input),ref mut budget,ref mut work)){case .ok(_):{assert(false);}case .err(e):{let p=packages::error(ref e);assert(p.phase==phase && p.reason==reason && p.file==file && budget.files==7 && budget.bytes==13 && packages::error_path(ref e).len>0);if(phase==6){assert(packages::error_input(ref e).len>0 && p.start<=p.end && p.end<=packages::error_input(ref e).len);if(reason==1 || reason==5){assert(p.diagnostic==2 && p.end>p.start);}}}};
}
@test
@effects(filesystem,console)
fn void commits_budgets_after_module_validation_and_retains_error_bytes(){
    var u8[4096] argument=[0;4096];let usize length=switch(process::arg(1,ref mut argument)){case .ok(value):{value}case .err(_):{assert(false);0}};let ref u8[] argument_path=ref argument;let fixture=switch(files::manifest(argument_path[..length])){case .ok(value):{value}case .err(_):{assert(false);bytes::empty()}};let ref u8[] root=bytes::view(ref fixture);
    let input=document(b"Good");var sources::Budget budget=sources::Budget{.files=4093,.bytes=67108826};var usize work=0;
    switch(packages::load(root,bytes::view(ref input),ref mut budget,ref mut work)){case .ok(value):{assert(modules::count(ref value)==3 && budget.files==4096 && budget.bytes==67108864 && bytes::equal(manifest::package_value(packages::package(ref value),b"uid"),b"example.catalog"));}case .err(_):{assert(false);}};
    rejected(root,b"BadModule",6,5,0);rejected(root,b"BadCase",6,5,0);rejected(root,b"BrokenSyntax",6,3,0);rejected(root,b"InvalidUtf8",6,2,0);rejected(root,b"UnicodeDir",6,1,0);rejected(root,b"DuplicateDeclared",6,5,1);rejected(root,b"Collision",3,4,0);rejected(root,b"Empty",3,5,0);rejected(root,b"Reparse",3,2,0);
    var bytes::Buffer changed=document(b"BadModule");budget=sources::Budget{.files=2,.bytes=3};work=0;
    let failure=switch(packages::load(root,bytes::view(ref changed),ref mut budget,ref mut work)){case .ok(_):{assert(false);return;}case .err(value):{value}};assert(bytes::resize_zeroed(ref mut changed,0).is_ok);
    let p=packages::error(ref failure);assert(p.phase==6 && p.reason==5 && p.start==7 && p.end==12 && budget.files==2 && budget.bytes==3 && bytes::equal(packages::error_path(ref failure),b"a.fy") && bytes::equal(packages::error_input(ref failure),b"module other;\n"));
    let invalid=b"schema='wrong'\n[package]\nuid='example.catalog'\nname='catalog'\nversion='1.0.0'\nlanguage='fairyc-language/20'\nsource='Good'\n";work=0;
    switch(packages::load(root,invalid,ref mut budget,ref mut work)){case .ok(_):{assert(false);}case .err(e):{let p=packages::error(ref e);assert(p.phase==1 && p.reason==10 && p.field==1 && bytes::equal(packages::error_input(ref e),invalid) && bytes::equal(packages::error_path(ref e),b"fairy.toml") && work==0 && budget.files==2 && budget.bytes==3);}};
    var bytes::Buffer physical=bytes::empty();assert(bytes::append(ref mut physical,root).is_ok && bytes::append(ref mut physical,b"/Package").is_ok);budget=sources::Budget{.files=0,.bytes=0};work=0;
    switch(packages::load_manifest(bytes::view(ref physical),ref mut budget,ref mut work)){case .ok(value):{assert(modules::find(ref value,b"nested::z")==2 && budget.files==3 && budget.bytes==38 && bytes::equal(manifest::package_value(packages::package(ref value),b"source"),b"src"));}case .err(_):{assert(false);}};
}
