module project_test_cli;
import project_check as check;
import project_check_text as output;
import std::bytes as bytes;
import std::process as process;
import std::io as io;
import public_cli_data as public;
import project_workspace_lock as loader;
import project_lock_policy as locks;
import project_module_visibility as visibility;
import project_workspace_graph as graph;
import project_instance_graph as instances;
import project_workspace_input as workspace_input;
import project_workspace_catalog as workspace_catalog;
import project_manifest as manifest;
import project_manifest_values as lexical;
import project_compiler_input as input;
import project_compiler_core as compiler;
import project_coordinate_graph as coordinates;
import project_source_set as sources;
import generic_signature_diagnostics as signatures;
import syntax as syntax;
import std::vec as v;

import project_check_cli as checking;
import project_test_options as test_options;
import native_cli as native;
import module_unit_interpreter as interpreter;
struct TestAttributeError{usize start;usize end;}

@effects(console) fn i32 failure(ref str message,i32 code){return if(io::write_stderr(message).is_ok){code}else{4};}
@effects(console)
fn i32 test_invocation(ref u8[] message){return if(io::write_stderr("fairyc: ").is_ok && io::write_stderr_bytes(message).is_ok && io::write_stderr("\n\n").is_ok && io::write_stderr(public::help()).is_ok){2}else{4};}
@effects(console)
fn i32 test_error(check::Error e){return checking::test_render_error(e);}
fn ref workspace_catalog::Catalog test_catalog(ref loader::Loaded loaded){return workspace_input::local_catalog(instances::input(visibility::instance_graph(loader::workspace(loaded))));}
fn ref u8[] test_package_path(ref loader::Loaded loaded,usize owner){let g=visibility::workspace_graph(loader::workspace(loaded));return workspace_catalog::path(test_catalog(loaded),graph::node(g,owner).package_index);}
fn ref u8[] test_package_uid(ref loader::Loaded loaded,usize owner){let g=visibility::workspace_graph(loader::workspace(loaded));return manifest::package_value(workspace_catalog::package(test_catalog(loaded),graph::node(g,owner).package_index),b"uid");}
fn bool test_path(ref mut bytes::Buffer out,ref u8[] package,ref u8[] file){return (bytes::equal(package,b".") || test_options::test_append(out,package) && test_options::test_append(out,b"/")) && test_options::test_append(out,file);}
fn bool test_module_name(ref syntax::Parsed parsed,ref mut bytes::Buffer out){let d=syntax::declaration_at(parsed,0);var usize i=0;while(i<syntax::token_count(parsed)){let t=syntax::token_info(parsed,i);if(d.name_start<=t.start && t.end<=d.name_end && t.kind!=2 && !test_options::test_append(out,syntax::source_view(parsed)[t.start..t.end])){return false;}i+=1;}return true;}






@effects(console)
fn i32 test_missing(ref v::Vec<bytes::Buffer> requested,ref v::Vec<bytes::Buffer> available,bool directory){var v::Vec<bytes::Buffer> missing=v::empty<bytes::Buffer>();var usize i=0;while(i<v::len<bytes::Buffer>(requested)){let p=bytes::view(v::at<bytes::Buffer>(requested,i));if(!test_options::test_contains(available,p) && !test_options::test_add(ref mut missing,p)){return 4;}i+=1;}if(v::len<bytes::Buffer>(ref missing)==0){return 0;}
    var v::Vec<usize> sorted=v::empty<usize>();if(!test_options::test_sort_strings(ref missing,ref mut sorted)){return 4;}var bytes::Buffer text=bytes::empty();if(!test_options::test_append(ref mut text,if(directory){b"command \"test\" test directory selection was not found: "}else{b"command \"test\" test file selection was not found: "})){return 4;}i=0;while(i<v::len<usize>(ref sorted)){if(i>0 && !test_options::test_append(ref mut text,b", ")){return 4;}if(!test_options::test_diagnostic_quote(ref mut text,bytes::view(v::at<bytes::Buffer>(ref missing,*v::at<usize>(ref sorted,i))))){return 4;}i+=1;}return test_invocation(bytes::view(ref text));
}


@effects(console)
fn Result<v::Vec<bytes::Buffer>,i32> test_select_packages(ref loader::Loaded production,ref test_options::TestOptions options){
var v::Vec<bytes::Buffer> selected=v::empty<bytes::Buffer>();let g=visibility::workspace_graph(loader::workspace(production));let local_count=graph::local_node_count(g);
    if(!test_options::flags(options).workspace){var usize owner=0;var bool found=test_options::package(options).len==0;if(!found){while(owner<local_count){if(bytes::equal(test_package_uid(production,owner),test_options::package(options))){found=true;break;}owner+=1;}}
        if(!found){let c=test_catalog(production);var usize pi=0;while(pi<workspace_catalog::count(c)){let uid=manifest::package_value(workspace_catalog::package(c,pi),b"uid");if(bytes::equal(uid,test_options::package(options))){var bytes::Buffer message=bytes::empty();if(!test_options::test_append(ref mut message,b"command \"test\" cannot select package ") || !test_options::test_diagnostic_quote(ref mut message,uid) || !test_options::test_append(ref mut message,b": workspace package ") || !test_options::test_diagnostic_quote(ref mut message,workspace_catalog::path(c,pi)) || !test_options::test_append(ref mut message,b" is not root-reachable")){return .err(4);}return .err(test_invocation(bytes::view(ref message)));}pi+=1;}}
        if(!found){var bytes::Buffer message=bytes::empty();if(!test_options::test_append(ref mut message,b"command \"test\" package ") || !test_options::test_diagnostic_quote(ref mut message,test_options::package(options)) || !test_options::test_append(ref mut message,b" was not found in the local workspace")){return .err(4);}return .err(test_invocation(bytes::view(ref message)));}
        if(!test_options::test_add(ref mut selected,test_package_path(production,owner))){return .err(4);}
    }else{var usize owner=0;while(owner<local_count){let p=test_package_path(production,owner);if(test_options::test_contains(ref selected,p)){var bytes::Buffer message=bytes::empty();if(!test_options::test_append(ref mut message,b"command \"test\" cannot select workspace packages: workspace package ") || !test_options::test_diagnostic_quote(ref mut message,p) || !test_options::test_append(ref mut message,b" has multiple resolved instances")){return .err(4);}return .err(test_invocation(bytes::view(ref message)));}if(!test_options::test_add(ref mut selected,p)){return .err(4);}owner+=1;}}
    // Explicit package selection also rejects multiple feature instances.
    if(!test_options::flags(options).workspace){var usize count=0;var usize owner=0;while(owner<local_count){if(bytes::equal(test_package_path(production,owner),bytes::view(v::at<bytes::Buffer>(ref selected,0)))){count+=1;}owner+=1;}if(count!=1){var bytes::Buffer message=bytes::empty();if(!test_options::test_append(ref mut message,b"command \"test\" cannot select package ") || !test_options::test_diagnostic_quote(ref mut message,if(test_options::package(options).len==0){test_package_uid(production,0)}else{test_options::package(options)}) || !test_options::test_append(ref mut message,b": workspace package ") || !test_options::test_diagnostic_quote(ref mut message,bytes::view(v::at<bytes::Buffer>(ref selected,0))) || !test_options::test_append(ref mut message,b" has multiple resolved instances")){return .err(4);}return .err(test_invocation(bytes::view(ref message)));}}
    return .ok(selected);
}

fn bool test_has_parameters(ref syntax::Parsed parsed,syntax::Declaration declaration){var usize at=declaration.entry_start;while(at<declaration.entry_start+declaration.entry_count){if(syntax::entry_at(parsed,at).kind==1){return true;}at+=1;}return false;}
fn i32 test_declarations(ref syntax::Parsed parsed,ref test_options::TestOptions options,ref u8[] uid,ref u8[] file,usize source_index,ref mut v::Vec<test_options::TestIdentity> tests,ref mut v::Vec<TestAttributeError> invalid){let source=syntax::source_view(parsed);var usize declaration=0;
        while(declaration<syntax::declaration_count(parsed)){let d=syntax::declaration_at(parsed,declaration);var usize j=0;var usize count=0;var usize first=0;
            while(j<d.attribute_count){let a=syntax::attribute_at(parsed,d.attribute_start+j);if(bytes::equal(source[a.name_start..a.name_end],b"test")){if(count==0){first=d.attribute_start+j;}count+=1;}j+=1;}
            if(count>0){let a=syntax::attribute_at(parsed,first);let valid=d.kind==3 && count==1 && a.argument_count==0 && !d.public && d.unsafe_start==d.unsafe_end && d.owner_start==d.owner_end && !test_has_parameters(parsed,d) && bytes::equal(syntax::type_text(parsed,d.result_type),b"void");
                if(!valid){if(d.kind==3){if(!v::push<TestAttributeError>(invalid,TestAttributeError{.start=a.start,.end=a.end}).is_ok){return 4;}}else{j=0;while(j<d.attribute_count){let bad=syntax::attribute_at(parsed,d.attribute_start+j);if(bytes::equal(source[bad.name_start..bad.name_end],b"test") && !v::push<TestAttributeError>(invalid,TestAttributeError{.start=bad.start,.end=bad.end}).is_ok){return 4;}j+=1;}}}
                else{var bytes::Buffer identity=bytes::empty();if(test_options::flags(options).workspace && (!test_options::test_append(ref mut identity,uid) || !test_options::test_append(ref mut identity,b"::")) || !test_module_name(parsed,ref mut identity) || !test_options::test_append(ref mut identity,b"::") || !test_options::test_append(ref mut identity,source[d.name_start..d.name_end])){return 4;}
                    let file_copy=switch(bytes::copy(file)){case .ok(p):{p}case .err(_):{return 4;}};let uid=switch(bytes::copy(uid)){case .ok(p):{p}case .err(_):{return 4;}};if(!v::push<test_options::TestIdentity>(tests,test_options::identity(identity,file_copy,uid,source_index,declaration)).is_ok){return 4;}}
            }
            j=0;while(j<d.cleanup_attribute_count){let a=syntax::attribute_at(parsed,d.cleanup_attribute_start+j);if(bytes::equal(source[a.name_start..a.name_end],b"test") && !v::push<TestAttributeError>(invalid,TestAttributeError{.start=a.start,.end=a.end}).is_ok){return 4;}j+=1;}declaration+=1;
        }
    return 0;
}

@effects(filesystem,console,environment,clock)
pub fn i32 run_test_list(ref u8[] root,ref test_options::TestOptions options){
    var sources::Budget budget=sources::Budget{.files=0,.bytes=0};var usize meter=0;
    let production=switch(loader::load(root,test_options::flags(options).locked,ref mut budget,ref mut meter)){case .ok(p):{p}case .err(e):{return test_error(check::load_failure(e));}};
    let lock=loader::state(ref production);let selected=switch(test_select_packages(ref production,options)){case .ok(p):{p}case .err(code):{return code;}};
    var sources::Budget overlay_budget=sources::Budget{.files=0,.bytes=0};meter=0;let resolved=switch(visibility::load_tests(root,false,ref selected,ref mut overlay_budget,ref mut meter)){case .ok(p):{p}case .err(e):{return test_error(check::load_failure(loader::test_failure(e)));}};let loaded=loader::from_tests(resolved,lock);
    let table=switch(coordinates::collect(visibility::workspace_graph(loader::workspace(ref loaded)),ref mut meter)){case .ok(p):{p}case .err(e):{return test_error(check::projection_failure(loaded,input::Problem{.code=e.code,.reason=2,.owner=e.owner,.module_index=0,.declaration=0}));}};
    var projected=switch(input::project_tests(loader::workspace(ref loaded),ref mut meter)){case .ok(p):{p}case .err(e):{return test_error(check::projection_failure(loaded,e));}};
    var v::Vec<test_options::TestIdentity> tests=v::empty<test_options::TestIdentity>();var v::Vec<bytes::Buffer> files=v::empty<bytes::Buffer>();var v::Vec<bytes::Buffer> dirs=v::empty<bytes::Buffer>();var bytes::Buffer diagnostics=bytes::empty();var bool diagnosed=false;var usize diagnosed_owner=0;var usize source_cursor=0;var v::Vec<usize> source_order=v::empty<usize>();
    var usize sort_i=0;while(sort_i<input::count(ref projected)){if(!v::push<usize>(ref mut source_order,sort_i).is_ok){return 4;}var usize j=sort_i;while(j>0){let previous=*v::at<usize>(ref source_order,j-1);let a=input::origin(ref projected,sort_i);let b=input::origin(ref projected,previous);let sg=visibility::workspace_graph(loader::workspace(ref loaded));var bool before=false;
        if(!graph::is_sdk(sg,a.owner) && !graph::is_sdk(sg,b.owner)){let au=test_package_uid(ref loaded,a.owner);let bu=test_package_uid(ref loaded,b.owner);before=if(test_options::flags(options).workspace && !bytes::equal(au,bu)){lexical::before(au,bu)}else{lexical::before(input::file(ref projected,sort_i),input::file(ref projected,previous))};}if(!before){break;}let slot=v::at_mut<usize>(ref mut source_order,j);*slot=previous;j-=1;}let slot=v::at_mut<usize>(ref mut source_order,j);*slot=sort_i;sort_i+=1;}
    while(source_cursor<v::len<usize>(ref source_order)){let source_index=*v::at<usize>(ref source_order,source_cursor);let origin=input::origin(ref projected,source_index);let source_file=input::file(ref projected,source_index);if(graph::is_sdk(visibility::workspace_graph(loader::workspace(ref loaded)),origin.owner) || !test_options::test_contains(ref selected,test_package_path(ref loaded,origin.owner)) || !graph::module_is_test(visibility::workspace_graph(loader::workspace(ref loaded)),origin.owner,origin.module_index)){source_cursor+=1;continue;}
        if(diagnosed && origin.owner!=diagnosed_owner){break;}var bytes::Buffer file=bytes::empty();if(!test_path(ref mut file,test_package_path(ref loaded,origin.owner),source_file) || !test_options::test_add(ref mut files,bytes::view(ref file))){return 4;}
        var usize end=bytes::len(ref file);var usize minimum=if(bytes::equal(test_package_path(ref loaded,origin.owner),b".")){5}else{test_package_path(ref loaded,origin.owner).len+6};while(end>minimum){end-=1;if(bytes::view(ref file)[end]==47){let p=bytes::view(ref file)[..end];if(!test_options::test_contains(ref dirs,p) && !test_options::test_add(ref mut dirs,p)){return 4;}if(end==minimum){break;}}}
        let parsed=graph::module_parser(visibility::workspace_graph(loader::workspace(ref loaded)),origin.owner,origin.module_index);let source=syntax::source_view(parsed);var v::Vec<TestAttributeError> invalid=v::empty<TestAttributeError>();var usize declaration=0;
        let declarations=test_declarations(parsed,options,test_package_uid(ref loaded,origin.owner),bytes::view(ref file),source_index,ref mut tests,ref mut invalid);if(declarations!=0){return declarations;}
        var usize i=1;while(i<v::len<TestAttributeError>(ref invalid)){let value=*v::at<TestAttributeError>(ref invalid,i);var usize j=i;while(j>0 && v::at<TestAttributeError>(ref invalid,j-1).start>value.start){let previous=*v::at<TestAttributeError>(ref invalid,j-1);let slot=v::at_mut<TestAttributeError>(ref mut invalid,j);*slot=previous;j-=1;}let slot=v::at_mut<TestAttributeError>(ref mut invalid,j);*slot=value;i+=1;}
        i=0;while(i<v::len<TestAttributeError>(ref invalid)){let d=*v::at<TestAttributeError>(ref invalid,i);let package=if(test_options::flags(options).workspace){test_package_path(ref loaded,origin.owner)}else{b"."};if(!output::prefix(ref mut diagnostics,package,source_file,source,d.start) || !test_options::test_append(ref mut diagnostics,b"E9001: `@test` must appear exactly once without arguments on a private safe zero-argument free function returning `void` in isolated test source. [bytes ") || !test_options::test_number(ref mut diagnostics,d.start) || !test_options::test_append(ref mut diagnostics,b"..") || !test_options::test_number(ref mut diagnostics,d.end) || !test_options::test_append(ref mut diagnostics,b", repair=none]\n")){return 4;}i+=1;}if(v::len<TestAttributeError>(ref invalid)>0){diagnosed=true;diagnosed_owner=origin.owner;}source_cursor+=1;
    }
    if(bytes::len(ref diagnostics)>0){return if(io::write_stderr_bytes(bytes::view(ref diagnostics)).is_ok){3}else{4};}
    let missing_dirs=test_missing(test_options::dirs(options),ref dirs,true);if(missing_dirs!=0){return missing_dirs;}let missing_files=test_missing(test_options::files(options),ref files,false);if(missing_files!=0){return missing_files;}
    var v::Vec<usize> chosen=v::empty<usize>();if(!test_options::test_choose(options,ref tests,ref mut chosen)){return 4;}
    var v::Vec<bytes::Buffer> identities=v::empty<bytes::Buffer>();var usize exact_i=0;while(exact_i<v::len<test_options::TestIdentity>(ref tests)){if(!test_options::test_add(ref mut identities,test_options::test_name(v::at<test_options::TestIdentity>(ref tests,exact_i)))){return 4;}exact_i+=1;}
    let missing_exact=test_missing_exact(test_options::exact(options),ref identities);if(missing_exact!=0){return missing_exact;}
    var usize source_work=0;var usize link_work=0;var v::Vec<signatures::Reports> reports=v::empty<signatures::Reports>();
    let complete=switch(compiler::build_complete_reporting(ref projected,ref table,ref mut reports,ref mut source_work,ref mut link_work)){case .ok(p):{p}case .err(e):{return test_error(check::compile_failure_reports(loaded,projected,e,reports));}};
    if(test_options::flags(options).stdin && v::len<usize>(ref chosen)!=1){return test_invocation(b"command \"test\" requires exactly one selected test when --stdin is supplied");}
    if(!test_options::flags(options).list && !test_options::flags(options).native){return interpreter::tests(ref complete,options,ref tests,ref chosen,ref projected,locks::state_code(lock));}
    if(!test_options::flags(options).list){return native::tests(complete,ref projected,ref table,options,ref tests,ref chosen,root,locks::state_code(lock));}discard complete;
    return test_options::test_render(options,ref tests,ref projected,locks::state_code(lock));
}

@effects(console) fn i32 test_missing_exact(ref v::Vec<bytes::Buffer> requested,ref v::Vec<bytes::Buffer> available){var v::Vec<bytes::Buffer> missing=v::empty<bytes::Buffer>();var usize i=0;while(i<v::len<bytes::Buffer>(requested)){let value=bytes::view(v::at<bytes::Buffer>(requested,i));if(!test_options::test_contains(available,value) && !test_options::test_add(ref mut missing,value)){return 4;}i+=1;}if(v::len<bytes::Buffer>(ref missing)==0){return 0;}var v::Vec<usize> order=v::empty<usize>();if(!test_options::test_sort_strings(ref missing,ref mut order)){return 4;}var bytes::Buffer message=bytes::empty();if(!test_options::test_append(ref mut message,b"command \"test\" exact test selection was not found: ")){return 4;}i=0;while(i<v::len<usize>(ref order)){if(i>0 && !test_options::test_append(ref mut message,b", ")){return 4;}if(!test_options::test_diagnostic_quote(ref mut message,bytes::view(v::at<bytes::Buffer>(ref missing,*v::at<usize>(ref order,i))))){return 4;}i+=1;}return test_invocation(bytes::view(ref message));}
@effects(filesystem,console,environment,clock)
pub fn i32 run(ref u8[] root,ref v::Vec<bytes::Buffer> argv){let selected=switch(test_options::parse_test_options(argv)){case .ok(p):{p}case .err(e):{return test_invocation(test_options::error_text(ref e));}};return run_test_list(root,ref selected);}
@effects(filesystem,console,environment,clock)
pub fn i32 arguments(){var v::Vec<bytes::Buffer> argv=v::empty<bytes::Buffer>();var usize i=2;while(i<process::arg_count()){var bytes::Buffer data=bytes::empty();if(!bytes::resize_zeroed(ref mut data,4096).is_ok){return 4;}let n=switch(process::arg(i,bytes::view_mut(ref mut data))){case .ok(p):{p}case .err(_):{return failure("fairyc: argument exceeds the supported byte limit\n",2);}};if(!test_options::test_add(ref mut argv,bytes::view(ref data)[..n])){return 4;}i+=1;}return run(b".",ref argv);}
