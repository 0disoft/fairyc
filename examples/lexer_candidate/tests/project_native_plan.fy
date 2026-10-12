module tests::project_native_plan;
import project_emit as emit;
import project_emit_text as diagnostics;
import project_lock_policy as locks;
import native_packet as packet;
import file_source as files;
import std::process as process;
import std::bytes as bytes;
import std::fs as fs;
import std::text as text;

@effects(filesystem) fn bytes::Buffer root(){var u8[4096] data=[0;4096];let n=switch(process::arg(1,ref mut data)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] view=ref data;return switch(files::manifest(view[..n])){case .ok(p):{p}case .err(_):{assert(false);return root();}};}
fn bytes::Buffer joined(ref u8[] root,ref u8[] leaf){var bytes::Buffer out=bytes::empty();assert(bytes::append(ref mut out,root).is_ok && bytes::append_byte(ref mut out,47).is_ok && bytes::append(ref mut out,leaf).is_ok);return out;}
@effects(filesystem) fn void successful(ref u8[] path,bool locked,u32 state,ref u8[] name){let plan=switch(emit::evaluate(path,locked)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(emit::package_name(ref plan),name));assert(locks::state_code(emit::lock_state(ref plan))==state);switch(packet::size(emit::output_bundle(ref plan))){case .ok(n):{assert(n>10);}case .err(_):{assert(false);}};}
@test @effects(filesystem) fn void retains_root_name_and_each_observed_lock_state(){let base=root();let ordinary=joined(bytes::view(ref base),b"ordinary");let lock_path=joined(bytes::view(ref ordinary),b"fairy.lock");let name=switch(text::validate_utf8(bytes::view(ref lock_path))){case .ok(p):{p}case .err(_):{assert(false);return;}};let file=switch(fs::create_truncate_utf8(name)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(fs::close(file).is_ok);successful(bytes::view(ref ordinary),false,2,b"native-plan-root");successful(bytes::view(ref ordinary),false,1,b"native-plan-root");successful(bytes::view(ref ordinary),true,3,b"native-plan-root");}
@test @effects(filesystem) fn void locked_and_entry_failures_keep_owned_diagnostics(){let base=root();let missing=joined(bytes::view(ref base),b"missing-lock");switch(emit::evaluate(bytes::view(ref missing),true)){case .ok(_):{assert(false);}case .err(e):{assert(emit::phase(ref e)==1 && diagnostics::exit_code(ref e)==7);var bytes::Buffer text=bytes::empty();assert(diagnostics::error(ref e,ref mut text) && bytes::len(ref text)>0);}};let invalid=joined(bytes::view(ref base),b"invalid-entry");switch(emit::evaluate(bytes::view(ref invalid),false)){case .ok(_):{assert(false);}case .err(e):{assert(emit::phase(ref e)==2 && diagnostics::exit_code(ref e)==2);var bytes::Buffer text=bytes::empty();assert(diagnostics::error(ref e,ref mut text) && bytes::len(ref text)>0);}};}
