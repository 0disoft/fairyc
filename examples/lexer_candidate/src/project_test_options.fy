module project_test_options;
import project_json_string as quoted;
import std::bytes as bytes;
import std::text as text;
import std::vec as v;
import std::io as io;
import project_manifest_values as lexical;
import project_compiler_input as input;
import syntax as syntax;
import sha256_stream as sha;
pub struct TestFlags{bool list;bool json;bool locations;bool workspace;bool locked;bool native;bool release;bool stdin;bool capture;bool timeout;}
pub resource TestOptions{TestFlags flags;bytes::Buffer package;bytes::Buffer stdin_file;v::Vec<bytes::Buffer> files;v::Vec<bytes::Buffer> dirs;v::Vec<bytes::Buffer> exact;cleanup{}}
pub resource TestOptionError{bytes::Buffer text;cleanup{}}
pub fn bool test_append(ref mut bytes::Buffer out,ref u8[] data){return bytes::len(out)<=8388608 && data.len<=8388608-bytes::len(out) && bytes::append(out,data).is_ok;}
pub fn bool test_text(ref mut bytes::Buffer out,ref str data){return test_append(out,text::as_bytes(data));}
pub fn bool test_number(ref mut bytes::Buffer out,usize value){return bytes::append_usize(out,value).is_ok;}
pub fn bool test_quote(ref mut bytes::Buffer out,ref u8[] value){var usize work=0;let p=switch(quoted::quote(value,ref mut work)){case .ok(p):{p}case .err(_):{return false;}};return test_append(out,bytes::view(ref p));}
pub fn Result<TestOptions,TestOptionError> test_option_error(ref str message){var bytes::Buffer out=bytes::empty();discard test_text(ref mut out,message);return .err(TestOptionError{.text=out});}
pub fn Result<TestOptions,TestOptionError> test_option_named(ref str prefix,ref u8[] value,ref str suffix,bool quote){var bytes::Buffer out=bytes::empty();if(!test_text(ref mut out,prefix) || !(if(quote){test_diagnostic_quote(ref mut out,value)}else{test_append(ref mut out,value)}) || !test_text(ref mut out,suffix)){discard v::empty<usize>();}return .err(TestOptionError{.text=out});}
pub fn bool test_add(ref mut v::Vec<bytes::Buffer> values,ref u8[] value){let p=switch(bytes::copy(value)){case .ok(p):{p}case .err(_):{return false;}};return v::push<bytes::Buffer>(values,p).is_ok;}
pub fn bool test_contains(ref v::Vec<bytes::Buffer> values,ref u8[] value){var usize i=0;while(i<v::len<bytes::Buffer>(values)){if(bytes::equal(bytes::view(v::at<bytes::Buffer>(values,i)),value)){return true;}i+=1;}return false;}
pub fn bool test_starts(ref u8[] value,ref u8[] prefix){return value.len>=prefix.len && bytes::equal(value[..prefix.len],prefix);}
pub fn bool test_bool_flag(ref u8[] flag){return bytes::equal(flag,b"--list") || bytes::equal(flag,b"--json") || bytes::equal(flag,b"--locations") || bytes::equal(flag,b"--workspace") || bytes::equal(flag,b"--locked") || bytes::equal(flag,b"--native") || bytes::equal(flag,b"--release") || bytes::equal(flag,b"--stdin") || bytes::equal(flag,b"--capture-output");}
pub fn Result<TestOptions,TestOptionError> parse_test_options(ref v::Vec<bytes::Buffer> argv){
    var TestFlags flags=TestFlags{.list=false,.json=false,.locations=false,.workspace=false,.locked=false,.native=false,.release=false,.stdin=false,.capture=false,.timeout=false};var bytes::Buffer package_name=bytes::empty();var bytes::Buffer input_file=bytes::empty();var v::Vec<bytes::Buffer> file_values=v::empty<bytes::Buffer>();var v::Vec<bytes::Buffer> dir_values=v::empty<bytes::Buffer>();var v::Vec<bytes::Buffer> exact_values=v::empty<bytes::Buffer>();
    var v::Vec<bytes::Buffer> seen=v::empty<bytes::Buffer>();var usize i=0;var bool delimiter=false;
    while(i<v::len<bytes::Buffer>(argv)){let flag=bytes::view(v::at<bytes::Buffer>(argv,i));if(bytes::equal(flag,b"--")){delimiter=true;break;}
        let package=bytes::equal(flag,b"--package");let file=bytes::equal(flag,b"--file");let dir=bytes::equal(flag,b"--dir");let exact=bytes::equal(flag,b"--exact");let input=bytes::equal(flag,b"--stdin-file");let timeout=bytes::equal(flag,b"--timeout-seconds");
        if(package || file || dir || exact || input || timeout){
            if(i+1>=v::len<bytes::Buffer>(argv) || test_starts(bytes::view(v::at<bytes::Buffer>(argv,i+1)),b"--")){return test_option_error(if(package){"command \"test\" requires a package UID after --package"}else if(file){"command \"test\" requires a test path after --file"}else if(dir){"command \"test\" requires a test directory after --dir"}else if(exact){"command \"test\" requires a test identity after --exact"}else if(input){"command \"test\" requires a portable workspace file after --stdin-file"}else{"command \"test\" requires canonical seconds after --timeout-seconds"});}
            let value=bytes::view(v::at<bytes::Buffer>(argv,i+1));
            if((package || input || timeout) && test_contains(ref seen,flag)){return test_option_named("command \"test\" received ",flag," more than once",false);}
            if(file && test_contains(ref file_values,value) || dir && test_contains(ref dir_values,value) || exact && test_contains(ref exact_values,value)){return test_option_named(if(file){"command \"test\" received test file "}else if(dir){"command \"test\" received test directory "}else{"command \"test\" received exact test "},value," more than once",true);}
            if(!test_add(ref mut seen,flag)){return test_option_error("internal allocation failed");}
            if(package){if(!bytes::append(ref mut package_name,value).is_ok){return test_option_error("internal allocation failed");}}
            else if(input){if(!bytes::append(ref mut input_file,value).is_ok){return test_option_error("internal allocation failed");}}
            else if(file){if(!test_add(ref mut file_values,value)){return test_option_error("internal allocation failed");}}
            else if(dir){if(!test_add(ref mut dir_values,value)){return test_option_error("internal allocation failed");}}
            else if(exact){if(!test_add(ref mut exact_values,value)){return test_option_error("internal allocation failed");}}
            else{var usize seconds=0;var usize k=0;var bool valid=value.len>0 && value.len<=2 && value[0]!=48;while(k<value.len){if(value[k]<48 || value[k]>57){valid=false;}else if(k<2){seconds=seconds*10+((value[k]-48) as usize);}else{valid=false;}k+=1;}if(!valid || seconds<1 || seconds>60){return test_option_error("command \"test\" requires --timeout-seconds in canonical range 1 through 60");}flags.timeout=true;}
            i+=2;continue;
        }
        if(!test_bool_flag(flag)){return test_option_error("command \"test\" accepts only optional --list, --json, --locations, --native, --release, --stdin, --stdin-file <portable-workspace-file>, --capture-output, --workspace, --package <package-uid>, --dir <portable-test-directory>, --file <portable-test-path>, --exact <test-identity>, and --locked");}
        if(test_contains(ref seen,flag)){return test_option_named("command \"test\" received ",flag," more than once",false);}if(!test_add(ref mut seen,flag)){return test_option_error("internal allocation failed");}
        if(bytes::equal(flag,b"--list")){flags.list=true;}else if(bytes::equal(flag,b"--json")){flags.json=true;}else if(bytes::equal(flag,b"--locations")){flags.locations=true;}else if(bytes::equal(flag,b"--workspace")){flags.workspace=true;}else if(bytes::equal(flag,b"--locked")){flags.locked=true;}else if(bytes::equal(flag,b"--native")){flags.native=true;}else if(bytes::equal(flag,b"--release")){flags.release=true;}else if(bytes::equal(flag,b"--stdin")){flags.stdin=true;}else{flags.capture=true;}i+=1;
    }
    if(flags.workspace && bytes::len(ref package_name)!=0){return test_option_error("command \"test\" does not combine --workspace with --package");}
    if(v::len<bytes::Buffer>(ref file_values)!=0 && v::len<bytes::Buffer>(ref exact_values)!=0){return test_option_error("command \"test\" does not combine --file with --exact");}
    if(v::len<bytes::Buffer>(ref dir_values)!=0 && (v::len<bytes::Buffer>(ref file_values)!=0 || v::len<bytes::Buffer>(ref exact_values)!=0)){return test_option_error("command \"test\" does not combine --dir with --file or --exact");}
    if(flags.list && (flags.native || flags.release || flags.stdin || bytes::len(ref input_file)!=0 || flags.timeout || flags.capture || v::len<bytes::Buffer>(ref exact_values)!=0)){return test_option_error("command \"test\" does not combine --list with --native, --release, --stdin, --stdin-file, --capture-output, or --exact");}
    if(flags.release && !flags.native){return test_option_error("command \"test\" requires --native when --release is supplied");}
    if(flags.stdin && !flags.native){return test_option_error("command \"test\" requires --native when --stdin is supplied");}
    if(flags.timeout && !flags.native){return test_option_error("command \"test\" requires --native when --timeout-seconds is supplied");}
    if(bytes::len(ref input_file)!=0 && !flags.native){return test_option_error("command \"test\" requires --native when --stdin-file is supplied");}
    if(flags.stdin && bytes::len(ref input_file)!=0){return test_option_error("command \"test\" does not combine --stdin with --stdin-file");}
    if(flags.capture && !flags.native){return test_option_error("command \"test\" requires --native when --capture-output is supplied");}
    if(flags.capture && !flags.json){return test_option_error("command \"test\" requires --json when --capture-output is supplied");}
    if(delimiter && !flags.native){return test_option_error("command \"test\" requires --native before --");}return .ok(TestOptions{.flags,.package=package_name,.stdin_file=input_file,.files=file_values,.dirs=dir_values,.exact=exact_values});
}

pub fn TestFlags flags(ref TestOptions o){return o.flags;}
pub fn ref u8[] package(ref TestOptions o){return bytes::view(ref o.package);}
pub fn ref v::Vec<bytes::Buffer> files(ref TestOptions o){return ref o.files;}
pub fn ref v::Vec<bytes::Buffer> dirs(ref TestOptions o){return ref o.dirs;}
pub fn ref v::Vec<bytes::Buffer> exact(ref TestOptions o){return ref o.exact;}
pub fn ref u8[] error_text(ref TestOptionError e){return bytes::view(ref e.text);}

pub resource TestIdentity{bytes::Buffer identity;bytes::Buffer file;bytes::Buffer uid;usize module_index;usize declaration;cleanup{}}

pub fn TestIdentity identity(bytes::Buffer identity,bytes::Buffer file,bytes::Buffer uid,usize module_index,usize declaration){return TestIdentity{.identity,.file,.uid,.module_index,.declaration};}
pub fn bool test_sort_strings(ref v::Vec<bytes::Buffer> values,ref mut v::Vec<usize> order){var usize i=0;while(i<v::len<bytes::Buffer>(values)){if(!v::push<usize>(order,i).is_ok){return false;}var usize j=i;while(j>0){let before=*v::at<usize>(order,j-1);if(!lexical::before(bytes::view(v::at<bytes::Buffer>(values,i)),bytes::view(v::at<bytes::Buffer>(values,before)))){break;}let slot=v::at_mut<usize>(order,j);*slot=before;j-=1;}let slot=v::at_mut<usize>(order,j);*slot=i;i+=1;}return true;}
fn bool test_hash_number(ref mut sha::State state,usize value){var usize rest=value;var u8[8] data=[0;8];let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";var usize i=0;while(i<8){data[7-i]=(digits[(rest/16)%16]<<4)|digits[rest%16];rest/=256;i+=1;}return rest==0 && sha::update(state,ref data);}
fn bool test_hash(ref u8[] source,bool lines,ref mut bytes::Buffer out){var sha::State state=sha::empty();if(!sha::update(ref mut state,if(lines){b"fairyc.source-line-map\x00\x01"}else{b"fairyc.source-file\x00\x01"}) || !test_hash_number(ref mut state,source.len)){return false;}
    if(lines){var usize count=1;var usize i=0;while(i<source.len){if(source[i]==10){count+=1;}i+=1;}if(!test_hash_number(ref mut state,count) || !test_hash_number(ref mut state,0)){return false;}i=0;while(i<source.len){if(source[i]==10 && !test_hash_number(ref mut state,i+1)){return false;}i+=1;}}
    else if(!sha::update(ref mut state,source)){return false;}return test_append(out,if(lines){b"fylm1-sha256-"}else{b"fyfile1-sha256-"}) && sha::append_hex(ref state,out);
}
fn usize test_line(ref u8[] source,usize start){var usize line=1;var usize i=0;while(i<start){if(source[i]==10){line+=1;}i+=1;}return line;}
fn usize test_column(ref u8[] source,usize start){var usize at=start;while(at>0 && source[at-1]!=10){at-=1;}return start-at+1;}
fn bool test_source_json(ref TestIdentity test,ref input::Input projected,ref mut bytes::Buffer out){let parsed=input::source(projected,test.module_index);let d=syntax::declaration_at(parsed,test.declaration);let source=syntax::source_view(parsed);var bytes::Buffer file_hash=bytes::empty();var bytes::Buffer lines_hash=bytes::empty();
    return test_hash(source,false,ref mut file_hash) && test_hash(source,true,ref mut lines_hash) && test_append(out,b"{\"packageUid\":") && test_quote(out,bytes::view(ref test.uid)) && test_append(out,b",\"path\":") && test_quote(out,bytes::view(ref test.file)) && test_append(out,b",\"line\":") && test_number(out,test_line(source,d.name_start)) && test_append(out,b",\"byteColumn\":") && test_number(out,test_column(source,d.name_start)) && test_append(out,b",\"startByte\":") && test_number(out,d.name_start) && test_append(out,b",\"endByte\":") && test_number(out,d.name_end) && test_append(out,b",\"fileIdentity\":") && test_quote(out,bytes::view(ref file_hash)) && test_append(out,b",\"lineMapIdentity\":") && test_quote(out,bytes::view(ref lines_hash)) && test_append(out,b"}");
}
fn bool test_selected(ref TestOptions options,ref TestIdentity test){let file=bytes::view(ref test.file);if(v::len<bytes::Buffer>(files(options))>0){return test_contains(files(options),file);}if(v::len<bytes::Buffer>(dirs(options))>0){var usize i=0;while(i<v::len<bytes::Buffer>(dirs(options))){let dir=bytes::view(v::at<bytes::Buffer>(dirs(options),i));if(file.len>dir.len && file[dir.len]==47 && test_starts(file,dir)){return true;}i+=1;}return false;}return true;}
@effects(console)
pub fn i32 test_render(ref TestOptions options,ref v::Vec<TestIdentity> tests,ref input::Input projected,u32 lock_code){
    // Discovery order is irrelevant: publish exact identity-byte order.
    var v::Vec<bytes::Buffer> identities=v::empty<bytes::Buffer>();var usize i=0;while(i<v::len<TestIdentity>(tests)){if(!test_add(ref mut identities,bytes::view(ref v::at<TestIdentity>(tests,i).identity))){return 4;}i+=1;}var v::Vec<usize> order=v::empty<usize>();if(!test_sort_strings(ref identities,ref mut order)){return 4;}
    var v::Vec<usize> chosen=v::empty<usize>();i=0;while(i<v::len<usize>(ref order)){let at=*v::at<usize>(ref order,i);if(test_selected(options,v::at<TestIdentity>(tests,at)) && !v::push<usize>(ref mut chosen,at).is_ok){return 4;}i+=1;}
    var bytes::Buffer out=bytes::empty();let label=lock_code;let lock_text=if(label==1){b"unchanged"}else if(label==2){b"updated"}else{b"verified"};
    if(flags(options).json){if(!test_append(ref mut out,b"{\"schemaVersion\":\"") || !test_append(ref mut out,if(flags(options).locations){b"fairyc.test-list/v3"}else if(flags(options).workspace){b"fairyc.test-list/v2"}else{b"fairyc.test-list/v1"}) || !test_append(ref mut out,b"\",\"tests\":[")){return 4;}
        i=0;while(i<v::len<usize>(ref chosen)){let test=v::at<TestIdentity>(tests,*v::at<usize>(ref chosen,i));if(i>0 && !test_append(ref mut out,b",")){return 4;}if(!test_quote(ref mut out,bytes::view(ref test.identity))){return 4;}i+=1;}if(!test_append(ref mut out,b"]")){return 4;}
        if(flags(options).locations){if(!test_append(ref mut out,b",\"locations\":[")){return 4;}i=0;while(i<v::len<usize>(ref chosen)){let test=v::at<TestIdentity>(tests,*v::at<usize>(ref chosen,i));if(i>0 && !test_append(ref mut out,b",")){return 4;}if(!test_append(ref mut out,b"{\"identity\":") || !test_quote(ref mut out,bytes::view(ref test.identity)) || !test_append(ref mut out,b",\"source\":") || !test_source_json(test,projected,ref mut out) || !test_append(ref mut out,b"}")){return 4;}i+=1;}if(!test_append(ref mut out,b"]")){return 4;}}
        if(!test_append(ref mut out,b",\"count\":") || !test_number(ref mut out,v::len<usize>(ref chosen)) || !test_append(ref mut out,b",\"lock\":") || !test_quote(ref mut out,lock_text) || !test_append(ref mut out,b"}\n")){return 4;}
    }else{i=0;while(i<v::len<usize>(ref chosen)){let test=v::at<TestIdentity>(tests,*v::at<usize>(ref chosen,i));if(!test_append(ref mut out,b"test ") || !test_append(ref mut out,bytes::view(ref test.identity))){return 4;}if(flags(options).locations){let parsed=input::source(projected,test.module_index);let d=syntax::declaration_at(parsed,test.declaration);let source=syntax::source_view(parsed);if(!test_append(ref mut out,b" at ") || !test_append(ref mut out,bytes::view(ref test.file)) || !test_append(ref mut out,b":") || !test_number(ref mut out,test_line(source,d.name_start)) || !test_append(ref mut out,b":") || !test_number(ref mut out,test_column(source,d.name_start))){return 4;}}if(!test_append(ref mut out,b"\n")){return 4;}i+=1;}
        if(!test_append(ref mut out,b"test list: ") || !test_number(ref mut out,v::len<usize>(ref chosen)) || !test_append(ref mut out,b" discovered; lock ") || !test_append(ref mut out,lock_text) || !test_append(ref mut out,b"\n")){return 4;}}
    return if(switch(io::write_bytes(bytes::view(ref out))){case .ok(n):{n==bytes::len(ref out)}case .err(_):{false}}){0}else{4};
}

pub fn bool test_diagnostic_quote(ref mut bytes::Buffer out,ref u8[] value){var usize work=0;let quoted=switch(quoted::quote_diagnostic(value,ref mut work)){case .ok(p):{p}case .err(_):{return false;}};return test_append(out,bytes::view(ref quoted));}
