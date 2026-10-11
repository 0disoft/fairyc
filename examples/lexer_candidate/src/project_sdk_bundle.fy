module project_sdk_bundle;
import sdk_authority as authority;
import sdk_scopes as scopes;
import project_source_set as sources;
import project_content_hash as hash;
import project_path as work;
import sha256_stream as sha;
import scanner as scanner;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;

// Input labels do not grant authority: every body must match its existing pin.
pub resource File{usize id;bytes::Buffer body;cleanup{}}
pub fn File file(usize id,bytes::Buffer body){return File{.id,.body};}
pub fn usize file_size(ref File file){return bytes::len(ref file.body);}
pub struct Problem{u32 code;usize file;}
// 1 invalid/duplicate label, 2 incomplete bundle, 3 unauthenticated bytes,
// 4 allocation, 5 implementation/work/source bound, 6 syntax invariant.
pub resource Bundle{bool candidate;v::Vec<syntax::Parsed> modules;bytes::Buffer digest;bytes::Buffer identity;bytes::Buffer paths;cleanup{}}
pub fn ref u8[] uid(){return b"fairyc.std";}
pub fn ref u8[] package_name(){return b"fairyc-std";}
pub fn ref u8[] version(){return b"0.0.0-stage0.86";}
pub fn ref u8[] language(){return b"fairyc-language/20";}
pub fn ref u8[] runtime_abi(){return b"fairyc-runtime/0";}
pub fn ref u8[] package_path(){return b"@sdk/stdlib";}
pub fn ref u8[] node_id(){return b"sdk:stdlib[]";}
pub fn ref u8[] sdk_identity(ref Bundle bundle){if(bundle.candidate){return b"fairyc-sdk-vector-candidate/0";}return b"fairyc-sdk-stdlib/0";}
pub fn usize count(ref Bundle bundle){return v::len<syntax::Parsed>(ref bundle.modules);}
pub fn bool vector(ref Bundle bundle){return count(bundle)==8;}
pub fn bool candidate(ref Bundle bundle){return bundle.candidate;}
pub fn ref u8[] digest(ref Bundle bundle){return bytes::view(ref bundle.digest);}
pub fn ref u8[] source_identity(ref Bundle bundle){return bytes::view(ref bundle.identity);}
pub fn ref u8[] name(usize index){return scopes::name(index+1);}
pub fn ref u8[] path(usize index){if(index==0){return b"std/alloc.fy";}if(index==1){return b"std/bytes.fy";}if(index==2){return b"std/fs.fy";}if(index==3){return b"std/io.fy";}if(index==4){return b"std/num.fy";}if(index==5){return b"std/process.fy";}if(index==6){return b"std/text.fy";}if(index==7){return b"std/vec.fy";}return b"";}
pub fn ref syntax::Parsed parsed(ref Bundle bundle,usize index){assert(index<count(bundle));return v::at<syntax::Parsed>(ref bundle.modules,index);}
pub fn ref u8[] source_path(ref Bundle bundle,usize index){assert(index<count(bundle));var usize start=0;var usize at=0;while(at<index){start+=path(at).len;at+=1;}return bytes::view(ref bundle.paths)[start..start+path(index).len];}
pub fn ref u8[] module_name(ref Bundle bundle,usize index){let tree=parsed(bundle,index);let d=syntax::declaration_at(tree,0);return syntax::source_view(tree)[d.name_start..d.name_end];}
pub fn ref u8[] source(ref Bundle bundle,usize index){return syntax::source_view(parsed(bundle,index));}
pub fn usize find(ref Bundle bundle,ref u8[] value){let usize id=authority::module_id(value);return if(id<=count(bundle)){id}else{0};}
fn Problem fail(u32 code,usize file){return Problem{.code,.file};}

// Pure assembly from owned compiler-bundle bytes. No path search or fallback.
// Retains canonical source order and parsed originals; commits counters only
// after authentication, parsing, exact digest and composed identity all succeed.
pub fn Result<Bundle,Problem> assemble(v::Vec<File> input,bool candidate,ref mut sources::Budget budget,ref mut usize meter){
    let usize count=v::len<File>(ref input);if(count<7 || count>8 || candidate && count!=8){return .err(fail(2,0));}if(budget.files>4096 || budget.bytes>67108864 || count>4096-budget.files){return .err(fail(5,0));}
    var usize[8] positions=[0;8];var usize total=0;var usize at=0;
    while(at<count){let f=v::at<File>(ref input,at);let ref u8[] body=bytes::view(ref f.body);if(!work::charge(meter,body.len+1)){return .err(fail(5,at));}if(f.id==0 || f.id>count || positions[f.id-1]!=0){return .err(fail(1,at));}positions[f.id-1]=at+1;
        if(body.len>8388608 || body.len>67108864-budget.bytes-total){return .err(fail(5,at));}if(!authority::authentic(scopes::name(f.id),body)){return .err(fail(3,at));}total+=body.len;at+=1;
    }
    var sha::State state=sha::empty();if(!hash::begin(ref mut state,count)){return .err(fail(5,0));}var v::Vec<syntax::Parsed> modules=v::empty<syntax::Parsed>();var bytes::Buffer paths=bytes::empty();at=0;
    while(at<count){if(positions[at]==0){return .err(fail(2,at));}let usize original=positions[at]-1;let ref u8[] body=bytes::view(ref v::at<File>(ref input,original).body);if(!work::charge(meter,body.len+path(at).len+1) || !hash::file(ref mut state,path(at),body)){return .err(fail(5,original));}
        let bytes::Buffer copied=switch(bytes::copy(body)){case .ok(value):{value}case .err(_):{return .err(fail(4,original));}};let scanner::Stream stream=switch(scanner::collect(copied,false,8388608,1048576,256,16777216)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==3){4}else{5},original));}};
        if(scanner::diagnostic_count(ref stream)!=0){return .err(fail(6,original));}let syntax::Parsed tree=switch(syntax::parse(stream,512)){case .ok(value):{value}case .err(code):{return .err(fail(if(code==3){4}else{6},original));}};
        if(syntax::problem_count(ref tree)!=0 || syntax::module_count(ref tree)!=1 || syntax::declaration_count(ref tree)==0){return .err(fail(6,original));}let d=syntax::declaration_at(ref tree,0);if(d.kind!=1 || !bytes::equal(body[d.name_start..d.name_end],name(at))){return .err(fail(6,original));}
        if(!v::push<syntax::Parsed>(ref mut modules,tree).is_ok || !bytes::append(ref mut paths,path(at)).is_ok){return .err(fail(4,original));}at+=1;
    }
    let bytes::Buffer digest=switch(hash::finish(ref state)){case .ok(value):{value}case .err(_):{return .err(fail(4,0));}};var bytes::Buffer identity=bytes::empty();let ref u8[] sdk=if(candidate){b"fairyc-sdk-vector-candidate/0"}else{b"fairyc-sdk-stdlib/0"};
    if(!bytes::append(ref mut identity,sdk).is_ok || !bytes::append_byte(ref mut identity,59).is_ok || !bytes::append(ref mut identity,runtime_abi()).is_ok || !bytes::append_byte(ref mut identity,59).is_ok || !bytes::append(ref mut identity,bytes::view(ref digest)).is_ok){return .err(fail(4,0));};
    *budget=sources::Budget{.files=budget.files+count,.bytes=budget.bytes+total};return .ok(Bundle{.candidate,.modules,.digest,.identity,.paths});
}
