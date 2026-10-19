module windows_msvc_toolchain;
import windows_host_paths as host;
import windows_run_capture as capture;
import windows_run_control as environment;
import file_source as files;
import std::bytes as bytes;
import std::vec as v;
import std::text as text;
import std::fs as fs;

// 1 input/path, 2 allocation, 3 installation, 4 child process, 5 tool version,
// 6 SDK, 7 environment. No compiler or shell is invoked by a successful resolve.
pub struct Problem {u32 stage;u32 status;u32 os_code;}
pub resource Toolchain {bytes::Buffer compiler;bytes::Buffer version;bytes::Buffer msvc;bytes::Buffer sdk;bytes::Buffer sdk_version;bytes::Buffer environment;cleanup{}}
fn Problem fail(u32 stage,u32 status,u32 os_code){return Problem{.stage,.status,.os_code};}
pub fn ref u8[] compiler(ref Toolchain t){return bytes::view(ref t.compiler);}
pub fn ref u8[] version(ref Toolchain t){return bytes::view(ref t.version);}
pub fn ref u8[] msvc(ref Toolchain t){return bytes::view(ref t.msvc);}
pub fn ref u8[] sdk(ref Toolchain t){return bytes::view(ref t.sdk);}
pub fn ref u8[] sdk_version(ref Toolchain t){return bytes::view(ref t.sdk_version);}
pub fn ref u8[] block(ref Toolchain t){return bytes::view(ref t.environment);}
fn Result<bytes::Buffer,Problem> copy(ref u8[] data){return switch(bytes::copy(data)){case .ok(p):{.ok(p)}case .err(_):{.err(fail(2,0,0))}};}
pub fn Result<bytes::Buffer,Problem> joined(ref u8[] root,ref u8[] leaf){
    if(root.len==0 || root.len>4095 || leaf.len>4095 || root.len+leaf.len+1>4095){return .err(fail(1,0,0));}var bytes::Buffer out=copy(root)?;
    if(root[root.len-1]!=47 && root[root.len-1]!=92 && !bytes::append_byte(ref mut out,47).is_ok || !bytes::append(ref mut out,leaf).is_ok){return .err(fail(2,0,0));}return .ok(out);
}
pub fn bool absolute(ref u8[] path){return path.len>2 && ((path[0]>=65 && path[0]<=90 || path[0]>=97 && path[0]<=122) && path[1]==58 && (path[2]==47 || path[2]==92) || path[0]==92 && path[1]==92);}
pub fn bool component(ref u8[] value){if(value.len==0 || value.len>128 || bytes::equal(value,b".") || bytes::equal(value,b"..")){return false;}var usize i=0;while(i<value.len){if(value[i]==47 || value[i]==92 || value[i]==58 || value[i]==0 || value[i]==10 || value[i]==13){return false;}i+=1;}return true;}
fn ref u8[] trimmed(ref u8[] raw){var usize start=0;var usize end=raw.len;while(start<end && (raw[start]==32 || raw[start]==9 || raw[start]==10 || raw[start]==13)){start+=1;}while(end>start && (raw[end-1]==32 || raw[end-1]==9 || raw[end-1]==10 || raw[end-1]==13)){end-=1;}return raw[start..end];}
fn u32 word(ref u8[] raw,usize i){return (raw[i*2] as u32)+(raw[i*2+1] as u32)*256;}
fn u8 lower(u8 c){return if(c>=65 && c<=90){c+32}else{c};}
pub fn Result<bytes::Buffer,Problem> value(ref u8[] block,ref u8[] name){
    if(block.len<4 || block.len>1048576 || block.len%2!=0 || word(block,block.len/2-1)!=0 || word(block,block.len/2-2)!=0){return .err(fail(7,1,0));}
    var usize cursor=0;var bytes::Buffer result=bytes::empty();var bool found=false;
    while(cursor<block.len/2 && word(block,cursor)!=0){let usize first=cursor;while(cursor<block.len/2 && word(block,cursor)!=0){cursor+=1;}if(cursor==block.len/2){return .err(fail(7,1,0));}cursor+=1;
        let entry=switch(host::decode(block[first*2..cursor*2])){case .ok(p):{p}case .err(_):{return .err(fail(7,1,0));}};let raw=bytes::view(ref entry);var usize separator=1;while(separator<raw.len && raw[separator]!=61){separator+=1;}if(separator==raw.len){return .err(fail(7,1,0));}
        var bool same=separator==name.len;var usize i=0;while(same && i<name.len){if(lower(raw[i])!=lower(name[i])){same=false;}i+=1;}
        if(same){if(found){return .err(fail(7,1,0));}result=copy(raw[separator+1..])?;found=true;}
    }
    if(cursor!=block.len/2-1 && !(cursor==0 && block.len==4)){return .err(fail(7,1,0));}return .ok(result);
}
@effects(filesystem) pub fn bool exists(ref u8[] path,bool directory){
    let name=switch(text::validate_utf8(path)){case .ok(p):{p}case .err(_):{return false;}};let native=switch(text::cstring_from_text(name)){case .ok(p):{p}case .err(_):{return false;}};
    return switch(fs::path_kind(text::cstring_as_cstr(ref native))){case .ok(k):{if(directory){fs::path_kind_is_directory(k)}else{fs::path_kind_is_file(k)}}case .err(_):{false}};
}
fn bool after(ref u8[] a,ref u8[] b){var usize i=0;while(i<a.len && i<b.len){if(a[i]!=b[i]){return a[i]>b[i];}i+=1;}return a.len>b.len;}
@effects(filesystem) fn bool complete_sdk(ref u8[] root,ref u8[] version){
    let include=switch(joined(root,b"Include")){case .ok(p):{p}case .err(_):{return false;}};let base=switch(joined(bytes::view(ref include),version)){case .ok(p):{p}case .err(_):{return false;}};
    let lib=switch(joined(root,b"Lib")){case .ok(p):{p}case .err(_):{return false;}};let library=switch(joined(bytes::view(ref lib),version)){case .ok(p):{p}case .err(_):{return false;}};
    var usize i=0;while(i<5){let path=switch(joined(if(i<3){bytes::view(ref base)}else{bytes::view(ref library)},if(i==0){b"ucrt"}else if(i==1){b"shared"}else if(i==2){b"um"}else if(i==3){b"ucrt/x64"}else{b"um/x64"})){case .ok(p):{p}case .err(_):{return false;}};if(!exists(bytes::view(ref path),true)){return false;}i+=1;}return true;
}
@effects(filesystem) pub fn Result<bytes::Buffer,Problem> latest_sdk(ref u8[] root){
    let include=joined(root,b"Include")?;let name=switch(text::validate_utf8(bytes::view(ref include))){case .ok(p):{p}case .err(_):{return .err(fail(6,1,0));}};
    var fs::ReadDir directory=switch(fs::read_dir_utf8(name)){case .ok(p):{p}case .err(e):{return .err(fail(6,e.status,e.os_code));}};var u8[1024] raw=[0;1024];var bytes::Buffer selected=bytes::empty();var usize count=0;
    while(count<4096){let entry=switch(fs::next_dir(ref mut directory,ref mut raw)){case .ok(p):{p}case .err(e):{return .err(fail(6,e.status,e.os_code));}};if(!entry.has_entry){if(!fs::close_dir(directory).is_ok || bytes::len(ref selected)==0){return .err(fail(6,1,0));}return .ok(selected);}count+=1;
        let ref u8[] view=ref raw;let candidate=view[..entry.name_length];if(fs::path_kind_is_directory(entry.kind) && candidate.len>=3 && bytes::equal(candidate[..3],b"10.") && component(candidate) && after(candidate,bytes::view(ref selected)) && complete_sdk(root,candidate)){selected=copy(candidate)?;}
    }return .err(fail(6,2,0));
}
fn Result<usize,Problem> argument(ref mut v::Vec<bytes::Buffer> args,ref u8[] value){let data=copy(value)?;return switch(v::push<bytes::Buffer>(args,data)){case .ok(n):{.ok(n)}case .err(_):{.err(fail(2,0,0))}};}
@effects(filesystem,environment) fn Result<bytes::Buffer,Problem> program_files(ref u8[] inherited){
    var usize i=0;while(i<3){var bytes::Buffer candidate=bytes::empty();if(i<2){candidate=value(inherited,if(i==0){b"ProgramFiles(x86)"}else{b"ProgramFiles"})?;if(i==1 && bytes::len(ref candidate)>0 && !bytes::append(ref mut candidate,b" (x86)").is_ok){return .err(fail(2,0,0));}}
        else{let image=switch(host::executable()){case .ok(p):{p}case .err(_):{return .err(fail(3,1,0));}};if(bytes::len(ref image)>=3 && bytes::view(ref image)[1]==58){candidate=copy(bytes::view(ref image)[..3])?;if(!bytes::append(ref mut candidate,b"Program Files (x86)").is_ok){return .err(fail(2,0,0));}}}
        if(bytes::len(ref candidate)>0 && absolute(bytes::view(ref candidate))){let probe=joined(bytes::view(ref candidate),b"Microsoft Visual Studio/Installer/vswhere.exe")?;if(exists(bytes::view(ref probe),false)){return .ok(candidate);}}i+=1;
    }return .err(fail(3,1,0));
}
@effects(filesystem,console,environment,clock) pub fn Result<Toolchain,Problem> resolve(){
    let inherited=switch(environment::snapshot()){case .ok(p):{p}case .err(e):{return .err(fail(7,e.kind,e.os_code));}};let base=program_files(bytes::view(ref inherited))?;let vswhere=joined(bytes::view(ref base),b"Microsoft Visual Studio/Installer/vswhere.exe")?;let cwd=switch(host::current()){case .ok(p):{p}case .err(e):{return .err(fail(1,e.kind,e.os_code));}};
    var v::Vec<bytes::Buffer> args=v::empty<bytes::Buffer>();let usize a=argument(ref mut args,b"-latest")?;let usize b=argument(ref mut args,b"-products")?;let usize c=argument(ref mut args,b"*")?;let usize d=argument(ref mut args,b"-requires")?;let usize e=argument(ref mut args,b"Microsoft.VisualStudio.Component.VC.Tools.x86.x64")?;let usize f=argument(ref mut args,b"-property")?;let usize g=argument(ref mut args,b"installationPath")?;let usize h=argument(ref mut args,b"-utf8")?;
    let output=switch(capture::run_input(bytes::view(ref vswhere),ref args,bytes::view(ref cwd),60000,1,b"")){case .ok(p):{p}case .err(e):{return .err(fail(4,e.kind,e.os_code));}};if(capture::status(ref output)!=0){return .err(fail(4,capture::status(ref output),0));}
    let installation=trimmed(capture::stdout(ref output));if(!absolute(installation) || !text::validate_utf8(installation).is_ok){return .err(fail(3,1,0));}
    let version_path=joined(installation,b"VC/Auxiliary/Build/Microsoft.VCToolsVersion.default.txt")?;let raw=switch(files::manifest(bytes::view(ref version_path))){case .ok(p):{p}case .err(e):{return .err(fail(5,e,0));}};let version=copy(trimmed(bytes::view(ref raw)))?;if(!component(bytes::view(ref version))){return .err(fail(5,1,0));}
    let tools=joined(installation,b"VC/Tools/MSVC")?;let msvc=joined(bytes::view(ref tools),bytes::view(ref version))?;let compiler=joined(bytes::view(ref msvc),b"bin/Hostx64/x64/cl.exe")?;if(!exists(bytes::view(ref compiler),false)){return .err(fail(5,1,0));}
    let sdk=joined(bytes::view(ref base),b"Windows Kits/10")?;let sdk_version=latest_sdk(bytes::view(ref sdk))?;
    let msvc_bin=joined(bytes::view(ref msvc),b"bin/Hostx64/x64")?;let bins=joined(bytes::view(ref sdk),b"bin")?;let sdk_bin_version=joined(bytes::view(ref bins),bytes::view(ref sdk_version))?;let sdk_bin=joined(bytes::view(ref sdk_bin_version),b"x64")?;
    if(!exists(bytes::view(ref msvc_bin),true) || !exists(bytes::view(ref sdk_bin),true)){return .err(fail(5,1,0));}
    var bytes::Buffer includes=bytes::empty();var bytes::Buffer libraries=bytes::empty();var usize i=0;
    while(i<7){let parent=if(i==0 || i==4){bytes::view(ref msvc)}else{bytes::view(ref sdk)};var bytes::Buffer leaf=bytes::empty();
        if(i==0 || i==4){leaf=copy(if(i==0){b"include"}else{b"lib/x64"})?;}else{if(!bytes::append(ref mut leaf,if(i<4){b"Include/"}else{b"Lib/"}).is_ok || !bytes::append(ref mut leaf,bytes::view(ref sdk_version)).is_ok || !bytes::append(ref mut leaf,if(i==1){b"/ucrt"}else if(i==2){b"/shared"}else if(i==3){b"/um"}else if(i==5){b"/ucrt/x64"}else{b"/um/x64"}).is_ok){return .err(fail(2,0,0));}}
        let path=joined(parent,bytes::view(ref leaf))?;if(!exists(bytes::view(ref path),true)){return .err(fail(5,1,0));}let dest=if(i<4){ref mut includes}else{ref mut libraries};if(bytes::len(dest)>0 && !bytes::append_byte(dest,59).is_ok || !bytes::append(dest,bytes::view(ref path)).is_ok){return .err(fail(2,0,0));}i+=1;
    }
    var bytes::Buffer path=copy(bytes::view(ref msvc_bin))?;let prior=value(bytes::view(ref inherited),b"PATH")?;if(!bytes::append_byte(ref mut path,59).is_ok || !bytes::append(ref mut path,bytes::view(ref sdk_bin)).is_ok || !bytes::append_byte(ref mut path,59).is_ok || !bytes::append(ref mut path,bytes::view(ref prior)).is_ok){return .err(fail(2,0,0));}
    let first=switch(environment::with_value(bytes::view(ref inherited),b"PATH",bytes::view(ref path))){case .ok(p):{p}case .err(e):{return .err(fail(7,e.kind,e.os_code));}};
    let second=switch(environment::with_value(bytes::view(ref first),b"INCLUDE",bytes::view(ref includes))){case .ok(p):{p}case .err(e):{return .err(fail(7,e.kind,e.os_code));}};
    let environment=switch(environment::with_value(bytes::view(ref second),b"LIB",bytes::view(ref libraries))){case .ok(p):{p}case .err(e):{return .err(fail(7,e.kind,e.os_code));}};
    return .ok(Toolchain{.compiler,.version,.msvc,.sdk,.sdk_version,.environment});
}
