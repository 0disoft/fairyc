module project_check_text;
import project_check as check;
import project_check_status as status;
import project_workspace_lock as loader;
import project_module_visibility as visibility;
import project_instance_graph as instances;
import project_workspace_graph as graph;
import project_workspace_input as workspace;
import project_workspace_catalog as catalog;
import project_package_input as packages;
import project_compiler_input as input;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::text as text;

pub fn bool append(ref mut bytes::Buffer out,ref u8[] value){return bytes::len(out)<=1048576 && value.len<=1048576-bytes::len(out) && bytes::append(out,value).is_ok;}
pub fn bool number(ref mut bytes::Buffer out,usize n){return bytes::len(out)<=1048544 && bytes::append_usize(out,n).is_ok;}
fn bool word(ref mut bytes::Buffer out,ref str value){return append(out,text::as_bytes(value));}
fn bool range(ref mut bytes::Buffer out,usize start,usize end){return append(out,b" [bytes ") && number(out,start) && append(out,b"..") && number(out,end) && append(out,b"]\n");}
fn bool repair_range(ref mut bytes::Buffer out,usize start,usize end,ref u8[] repair){return append(out,b" [bytes ") && number(out,start) && append(out,b"..") && number(out,end) && append(out,b", repair=") && append(out,repair) && append(out,b"]\n");}
// Source failure recovery uses the retained bytes only; it never rereads disk.
fn bool code_text(ref mut bytes::Buffer out,u32 code){return append(out,b"E") && (if(code<10){append(out,b"000")}else if(code<100){append(out,b"00")}else if(code<1000){append(out,b"0")}else{true}) && number(out,code as usize);}
fn ref u8[] code_summary(u32 code){if(code==1){return b"invalid UTF-8 or unsupported identifier syntax";}if(code==2){return b"module declaration does not match the source-root-relative file path";}if(code==3){return b"invalid Stage 0 import";}if(code==4){return b"invalid numeric literal";}if(code==5){return b"invalid string escape or unterminated string";}if(code==6){return b"C-string contains an interior NUL";}if(code==9){return b"comparison operators require explicit grouping";}if(code==10){return b"frontend resource budget exhausted";}return b"unexpected token";}
struct SourceDiagnostic{u32 code;usize start;usize end;u32 expected;bool exit_repair;}
fn SourceDiagnostic parser_diagnostic(ref syntax::Parsed parsed,usize at){let p=syntax::problem_at(parsed,at);var usize start=p.start;if(p.expected==8){var usize previous=p.token;while(previous>0){previous-=1;let t=syntax::token_info(parsed,previous);if(t.kind!=2){start=t.end;break;}}}return SourceDiagnostic{.code=if(p.expected!=0){7}else{8},.start,.end=if(p.expected!=0){start}else{p.end},.expected=p.expected,.exit_repair=false};}
fn bool frontend_line(ref mut bytes::Buffer out,ref u8[] package,ref u8[] file,ref u8[] source,SourceDiagnostic d){if(!prefix(out,package,file,source,d.start) || !code_text(out,d.code) || !append(out,b": ")){return false;}
    if(d.code==7){if(!append(out,b"expected \"") || !append(out,syntax::expected_text(d.expected)) || !append(out,b"\"")){return false;}}
    else if(!append(out,code_summary(d.code))){return false;}
    return repair_range(out,d.start,d.end,if(d.code==7){b"insert"}else if(d.code==8){b"skip"}else if(d.exit_repair){b"exit"}else{b"none"});
}
fn bool source_diagnostics(ref packages::Error e,ref u8[] package,ref mut bytes::Buffer out){let source=packages::error_input(e);let owned=switch(bytes::copy(source)){case .ok(value):{value}case .err(_):{return false;}};
    let tokens=switch(scanner::collect(owned,false,8388608,1048576,256,16777216)){case .ok(value):{value}case .err(_):{return false;}};
    let parsed=switch(syntax::parse_diagnostics(tokens,512)){case .ok(value):{value}case .err(_):{return false;}};let stream=syntax::source_stream(ref parsed);var usize lexical=0;var usize grammar=0;
    while(lexical<scanner::diagnostic_count(stream) || grammar<syntax::problem_count(ref parsed)){
        var bool use_lexer=grammar==syntax::problem_count(ref parsed);if(lexical==scanner::diagnostic_count(stream)){use_lexer=false;}else if(!use_lexer){let l=scanner::diagnostic_info(stream,lexical);let r=parser_diagnostic(ref parsed,grammar);use_lexer=l.start<r.start || l.start==r.start && (l.end<r.end || l.end==r.end && l.code<r.code);}
        let d=if(use_lexer){let l=scanner::diagnostic_info(stream,lexical);SourceDiagnostic{.code=l.code,.start=l.start,.end=l.end,.expected=0,.exit_repair=l.exit_repair}}else{parser_diagnostic(ref parsed,grammar)};
        if(!frontend_line(out,package,packages::error_path(e),source,d)){return false;}if(use_lexer){lexical+=1;}else{grammar+=1;}
    }return lexical+grammar>0;
}
pub fn bool success(check::Stats s,ref mut bytes::Buffer out){return append(out,b"check ok: ") && number(out,s.instances) && append(out,b" instance(s), ") && number(out,s.files) && append(out,b" source file(s), ") && number(out,s.functions) && append(out,b" MIR function(s), lock ") && append(out,if(s.lock==1){b"unchanged"}else if(s.lock==2){b"updated"}else{b"verified"}) && append(out,b"\n");}
fn bool path(ref mut bytes::Buffer out,ref u8[] package,ref u8[] file){return (bytes::equal(package,b".") || package.len==0 || append(out,package) && append(out,b"/")) && append(out,file);}
pub fn ref u8[] package_path(ref loader::Loaded loaded,usize owner){let g=visibility::workspace_graph(loader::workspace(loaded));if(graph::is_sdk(g,owner)){return b"@sdk/stdlib";}let local=graph::local_graph(g);return catalog::path(workspace::local_catalog(instances::input(local)),graph::node(g,owner).package_index);}
fn bool location(ref mut bytes::Buffer out,ref u8[] package,ref u8[] file,ref u8[] source,usize start){if(start>source.len){return false;}var usize line=1;var usize column=1;var usize at=0;while(at<start){if(source[at]==10){line+=1;column=1;}else{column+=1;}at+=1;}return path(out,package,file) && append(out,b":") && number(out,line) && append(out,b":") && number(out,column);}
pub fn bool prefix(ref mut bytes::Buffer out,ref u8[] package,ref u8[] file,ref u8[] source,usize start){return location(out,package,file,source,start) && append(out,b": ");}
fn bool related(ref mut bytes::Buffer out,ref u8[] role,ref u8[] package,ref u8[] file,ref u8[] source,usize start,usize end){return start<=end && end<=source.len && append(out,b"  ") && append(out,role) && append(out,b": ") && location(out,package,file,source,start) && range(out,start,end);}
fn bool private_declaration(ref check::Error e,ref mut bytes::Buffer out){let p=check::compiler_error(e);let sources=check::compiler_input(e);let original=input::origin(sources,p.module_index);let source=syntax::source_view(input::source(sources,p.module_index));let spelling=source[p.start..p.end];var usize split=0;while(split<spelling.len && spelling[split]!=58){split+=1;}if(split+2>=spelling.len){return true;}let loaded=check::compiler_workspace(e);let resolved=loader::workspace(loaded);let g=visibility::workspace_graph(resolved);var usize at=0;
    while(at<visibility::lookup_count(resolved)){let l=visibility::lookup(resolved,at);if(l.owner==original.owner && l.source_module==original.module_index){let d=syntax::declaration_at(input::source(sources,p.module_index),l.declaration);if(bytes::equal(source[d.name_start..d.name_end],spelling[..split])){let parsed=graph::module_parser(g,l.provider,l.provider_module);let bytes=syntax::source_view(parsed);var usize index=1;while(index<syntax::declaration_count(parsed)){let declaration=syntax::declaration_at(parsed,index);if(bytes::equal(bytes[declaration.name_start..declaration.name_end],spelling[split+2..])){return related(out,b"declaration",package_path(loaded,l.provider),graph::module_file(g,l.provider,l.provider_module),bytes,declaration.name_start,declaration.name_end);}index+=1;}}}at+=1;}return true;
}
fn bool compilation(ref check::Error e,ref mut bytes::Buffer out){let p=check::compiler_error(e);let sources=check::compiler_input(e);if(p.module_index>=input::count(sources)){return append(out,b"fairyc: internal compiler invariant failed\n");}let parsed=input::source(sources,p.module_index);let source=syntax::source_view(parsed);let package=package_path(check::compiler_workspace(e),input::origin(sources,p.module_index).owner);
    if(p.phase==2 && p.code==5){return append(out,b"fairyc: semantic subset does not support type ") && append(out,source[p.start..p.end]) && append(out,b" at ") && append(out,input::file(sources,p.module_index)) && range(out,p.start,p.end);}
    if(!prefix(out,package,input::file(sources,p.module_index),source,p.start)){return false;}
    if(p.code==6){if(!append(out,b"E0103: item is not visible from this module")){return false;}}
    else if(p.code==1012){if(!append(out,b"E1012: loop control is outside a loop")){return false;}}
    else if(p.code==1101){if(!append(out,b"E1101: resource is used after ownership was moved")){return false;}}
    else if(p.code==1102){if(!append(out,b"E1102: resource representation cannot be moved or overwritten")){return false;}}
    else if(p.code==1103){if(!append(out,b"E1103: resource field cannot be moved separately")){return false;}}
    else if(p.code==1104 || p.code==1105){if(!append(out,if(p.code==1104){b"E1104: resource ownership differs between branches"}else{b"E1105: resource ownership differs at loop entry"})){return false;}}
    else if(p.code==1006 || p.code==1009){var bool returned=p.code==1009;var usize at=1;while(at<=syntax::statement_count(parsed)){let s=syntax::statement_at(parsed,at);if(s.kind==3 && s.expression!=0){let value=syntax::expression_at(parsed,s.expression);returned=returned || value.start==p.start && value.end==p.end;}at+=1;}if(!append(out,if(returned){b"E1009: return value does not match the declared result type"}else{b"E1006: incompatible expected and actual types"})){return false;}}
    else{if(!append(out,b"source is outside the supported compiler subset")){return false;}}
    if(!range(out,p.start,p.end)){return false;}let file=input::file(sources,p.module_index);
    if(p.code==6){return private_declaration(e,out);}
    if(p.code==1006 || p.code==1009){var usize at=1;while(at<=syntax::statement_count(parsed)){let s=syntax::statement_at(parsed,at);if(s.expression!=0){let value=syntax::expression_at(parsed,s.expression);if(value.start==p.start && value.end==p.end){if(s.kind==3){var usize index=1;while(index<syntax::declaration_count(parsed)){let d=syntax::declaration_at(parsed,index);if(d.kind==3 && d.start<=p.start && p.end<=d.end){let t=syntax::type_at(parsed,d.result_type);return related(out,b"expected-type",package,file,source,t.start,t.end);}index+=1;}}else if(s.kind==4 && s.target!=0){let target=syntax::expression_at(parsed,s.target);return related(out,b"expected-type",package,file,source,target.start,target.end);}}}at+=1;}}
    if(p.declaration_end>p.declaration_start && !related(out,b"declaration",package,file,source,p.declaration_start,p.declaration_end)){return false;}
    if(p.move_end>p.move_start && !related(out,b"move",package,file,source,p.move_start,p.move_end)){return false;}
    if(p.header_end>p.header_start && !related(out,b"loop-header",package,file,source,p.header_start,p.header_end)){return false;}return true;
}
fn bool package_input(ref packages::Error e,ref u8[] package,ref mut bytes::Buffer out){if(status::package_input(e)==3){return source_diagnostics(e,package,out);}return append(out,b"fairyc: package workspace failure: ") && path(out,package,packages::error_path(e)) && append(out,b": invalid package input\n");}
fn bool loaded(ref loader::Error e,ref mut bytes::Buffer out){let load_phase=loader::phase(e);if(load_phase==2){let p=loader::policy_error(e);if(p.stage==4){return append(out,b"fairyc: package workspace failure: lockfile is stale for the current workspace\n");}if(p.stage==3){return append(out,b"fairyc: package workspace failure: locked workspace requires a valid fairy.lock: ") && append(out,if(p.read_stage==2){b"lockfile is not a valid lock document\n"}else{b"lockfile is not a bounded regular file\n"});}return append(out,b"fairyc: package workspace failure: lockfile could not be published\n");}if(load_phase==3){return append(out,b"fairyc: internal compiler processing failed\n");}
    let visible=loader::input_error(e);let visible_phase=visibility::load_phase(visible);if(visible_phase==2){let failure=visibility::visibility_error(visible);return append(out,b"fairyc: package workspace failure: ") && word(out,visibility::reason(visibility::error(failure).code)) && append(out,b"\n");}if(visible_phase==3){return append(out,b"fairyc: internal compiler processing failed\n");}if(visible_phase==4){return append(out,b"fairyc: package workspace failure: invalid bundled standard library\n");}
    let local=visibility::input_error(visible);let local_phase=instances::load_phase(local);if(local_phase==2){return append(out,b"fairyc: package workspace failure: ") && word(out,instances::reason(instances::error(instances::graph_error(local)).code)) && append(out,b"\n");}if(local_phase==3){return append(out,b"fairyc: internal compiler processing failed\n");}
    let w=instances::input_error(local);if(!workspace::has_catalog_error(w)){return append(out,b"fairyc: package workspace failure: ") && word(out,workspace::reason(workspace::error(w).code)) && append(out,b"\n");}
    let c=workspace::nested_catalog_error(w);if(catalog::has_input_error(c)){return package_input(catalog::input_error(c),catalog::error_path(c),out);}return append(out,b"fairyc: package workspace failure: ") && word(out,catalog::reason(catalog::error(c).code)) && append(out,b"\n");
}
pub fn bool error(ref check::Error e,ref mut bytes::Buffer out){if(status::error(e)==4){return append(out,b"fairyc: internal compiler invariant failed\n");}let phase=check::phase(e);if(phase==1){return loaded(check::load_error(e),out);}if(phase==3){return compilation(e,out);}return append(out,b"fairyc: internal compiler processing failed\n");}
