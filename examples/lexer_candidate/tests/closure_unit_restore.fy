module tests::closure_unit_restore;
import module_unit_artifact_admission as admission;
import module_unit_artifact_identities as identities;
import module_identity_artifact_model as ids;
import module_project_artifact_bodies as recipes;
import module_project_compile as compile;
import module_identified_ir as identified;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean;
import module_program_artifact_encode as encode;
import module_program_artifact_decode as decode;
import module_program_artifact_wire as budget;
import candidate_artifact_wire as wire;
import module_signatures as signatures;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import project_source_plan as planning;
import generic_source_plan as plan;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);copied(raw)}};}
fn syntax::Parsed parsed(ref u8[] raw){let scanned=switch(scanner::collect(copied(raw),false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};return switch(syntax::parse(scanned,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(b"module empty;")}};}
pub resource Fixture{v::Vec<syntax::Parsed> sources;catalog::Catalog scope;signatures::Program metadata;plan::Plan plan;cleanup{}}
fn Fixture fixture(ref u8[] text){
    var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);var owners=v::empty<usize>();assert(v::push<usize>(ref mut owners,0).is_ok);let imports=v::empty<catalog::Binding>();
    var rows=v::empty<coordinates::Coordinate>();assert(v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=77}).is_ok);let table=coordinates::assemble(copied(b"fypc1-sha256-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),rows);
    let scope=switch(catalog::build_scoped(ref sources,ref owners,ref imports)){case .ok(p):{p}case .err(_):{assert(false);return fixture(text);}};let signature_scope=switch(catalog::build_scoped(ref sources,ref owners,ref imports)){case .ok(p):{p}case .err(_):{assert(false);return fixture(text);}};
    let metadata=switch(signatures::build_catalog(ref sources,signature_scope)){case .ok(p):{p}case .err(_):{assert(false);return fixture(text);}};var usize work=0;let original=switch(planning::build_sources(ref sources,ref owners,ref imports,ref table,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return fixture(text);}};return Fixture{.sources,.scope,.metadata,.plan=original};
}
fn identified::Program unit(ref Fixture p){var usize work=0;return switch(compile::range_identified(ref p.sources,ref p.metadata,ref p.plan,0,1,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);unit(p)}};}
fn wire::Payload encoded(ref identified::Program p){var usize work=0;return switch(encode::encode_closures(identified::ir_program(p),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);encoded(p)}};}
fn u32 rejected(ref u8[] raw,ref Fixture p,u32 mode,usize first,usize stop){var usize source=0;var usize artifact=0;return switch(if(mode==1){admission::restore(raw,ref p.sources,ref p.scope,ref p.metadata,ref p.plan,first,stop,ref mut source,ref mut artifact)}else if(mode==2){admission::restore_aggregates(raw,ref p.sources,ref p.scope,ref p.metadata,ref p.plan,first,stop,ref mut source,ref mut artifact)}else{admission::restore_closures(raw,ref p.sources,ref p.scope,ref p.metadata,ref p.plan,first,stop,ref mut source,ref mut artifact)}){case .ok(_):{0}case .err(code):{code}};}
@effects(console) pub fn void verify(ref u8[] name,ref u8[] text){
    let f=fixture(text);let original=unit(ref f);let data=encoded(ref original);let raw=wire::content(ref data);assert(bytes::equal(raw[..8],b"FYPROG02"));var usize source=0;var usize artifact=0;
    let fresh=switch(admission::restore_closures(raw,ref f.sources,ref f.scope,ref f.metadata,ref f.plan,0,1,ref mut source,ref mut artifact)){case .ok(p):{p}case .err(e):{discard io::write_stderr_bytes(name);discard io::write_stderr_u32(e);assert(false);return;}};let p=identified::ir_program(ref fresh);assert(source!=0 && artifact!=0 && identified::has_borrows(ref fresh) && proof::check(p).code==0 && clean::closures(ir::cleanup_ir(p)) && !clean::project(ir::cleanup_ir(p)));
    let again=encoded(ref fresh);assert(bytes::equal(raw,wire::content(ref again)));artifact=0;let snapshot=switch(identities::extract_closures(ref fresh,ref mut artifact)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize leaf=0;var usize i=1;while(i<=clean::type_count(ir::cleanup_ir(p))){if(clean::type_at(ir::cleanup_ir(p),i).kind==23){assert(ids::type_at(ref snapshot,i).node==0);leaf+=1;}i+=1;}assert(leaf==1);
    // Direct source/body recipe comparison is independently usable without the
    // still-closed whole coverage/call recipe publication lane.
    let checked=switch(compile::check_range(ref f.sources,ref f.metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};artifact=0;assert(recipes::match(p,ref snapshot,compile::checked_state(ref checked),ref f.plan,ref f.sources,ref f.metadata,ref mut artifact).is_ok);
    assert(rejected(raw,ref f,1,0,1)==1001 && rejected(raw,ref f,2,0,1)==1001);assert(rejected(raw[..raw.len-1],ref f,0,0,1)!=0);var bad=copied(raw);assert(bytes::append_byte(ref mut bad,0).is_ok && rejected(bytes::view(ref bad),ref f,0,0,1)==1001);assert(rejected(raw,ref f,0,0,0)!=0 && rejected(raw,ref f,0,0,2)!=0);
    source=0;artifact=budget::work_limit()+1;assert(switch(admission::restore_closures(raw,ref f.sources,ref f.scope,ref f.metadata,ref f.plan,0,1,ref mut source,ref mut artifact)){case .ok(_):{false}case .err(e):{e==1002}} && artifact==budget::work_limit()+1 && source==0);
}
// Both recipes pass detached verification. Their source-owned helper/body/
// callable facts must still fail admission against the other original source.
pub fn void mismatch(ref u8[] first,ref u8[] second){let a=fixture(first);let b=fixture(second);let owner=unit(ref a);let other=unit(ref b);let data=encoded(ref owner);let changed=encoded(ref other);let raw=wire::content(ref data);let forged=wire::content(ref changed);assert(!bytes::equal(raw,forged));var usize work=0;assert(decode::decode_closures(forged,ref mut work).is_ok);assert(rejected(forged,ref a,0,0,1)==1001 && rejected(raw,ref b,0,0,1)==1001);}
// Deliberately stale metadata/plan never become the current source authority.
pub fn void stale(ref u8[] first,ref u8[] second){var a=fixture(first);let owner=unit(ref a);let data=encoded(ref owner);var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(second)).is_ok);var usize source=0;var usize artifact=0;assert(!admission::restore_closures(wire::content(ref data),ref sources,ref a.scope,ref a.metadata,ref a.plan,0,1,ref mut source,ref mut artifact).is_ok);}
pub fn void scalar_closed(){let f=fixture(b"module main;fn i32 main(){return 42;}");let p=unit(ref f);var usize work=0;let data=switch(encode::encode(identified::ir_program(ref p),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(rejected(wire::content(ref data),ref f,0,0,1)==1001);work=0;assert(switch(identities::extract_closures(ref p,ref mut work)){case .ok(_):{false}case .err(e):{e==1001}});}
