module tests::module_source_artifact;
import tests::module_source_artifact_fixture as fixture;
import module_source_artifact_model as model;
import module_source_artifact_encode as encoder;
import module_source_artifact_decode as decoder;
import module_source_artifact_rebind as rebinding;
import generic_package_bindings as bindings;
import candidate_artifact_wire as wire;
import file_source as files;
import syntax as syntax;
import std::bytes as bytes;
import std::vec as v;
import std::process as process;
import std::io as io;
fn bytes::Buffer copied(ref u8[] raw){return switch(bytes::copy(raw)){case .ok(p):{p}case .err(_):{assert(false);return copied(raw);}};}
@effects(filesystem) fn bytes::Buffer golden(){var u8[4096] path=[0;4096];let n=switch(process::arg(1,ref mut path)){case .ok(n):{n}case .err(_):{assert(false);0}};let ref u8[] raw=ref path;return switch(files::manifest(raw[..n])){case .ok(p):{p}case .err(_):{assert(false);return golden();}};}
@effects(console) fn model::Snapshot decoded(ref u8[] raw,ref mut usize work){return switch(decoder::decode(raw,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return decoded(raw,work);}};}
@effects(console) fn wire::Payload encoded(ref model::Snapshot p,ref mut usize work){return switch(encoder::encode(p,work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return encoded(p,work);}};}
fn model::Snapshot snapshot(ref bindings::Bindings p,ref mut usize work){return switch(model::extract(p,work)){case .ok(p):{p}case .err(_):{assert(false);return snapshot(p,work);}};}
@effects(filesystem,console) fn model::Snapshot detached(){let raw=golden();var usize work=0;return decoded(bytes::view(ref raw),ref mut work);}
fn u32 rejected(ref u8[] raw){var usize work=0;return switch(decoder::decode(raw,ref mut work)){case .ok(_):{0}case .err(code):{code}};}
fn void same(ref model::Snapshot a,ref model::Snapshot b){assert(model::count(a)==model::count(b) && model::import_count(a)==model::import_count(b) && bytes::equal(model::names(a),model::names(b)));var usize i=0;while(i<model::count(a)){let x=model::module_at(a,i);let y=model::module_at(b,i);assert(x.owner==y.owner && x.namespace.start==y.namespace.start && x.namespace.end==y.namespace.end && x.path.start==y.path.start && x.path.end==y.path.end && x.first==y.first && x.count==y.count && x.source_size==y.source_size);var usize j=0;while(j<8){assert(x.digest[j]==y.digest[j]);j+=1;}i+=1;}i=0;while(i<model::import_count(a)){let x=model::import_at(a,i);let y=model::import_at(b,i);assert(x.module_index==y.module_index && x.declaration==y.declaration && x.target==y.target && x.alias.start==y.alias.start && x.alias.end==y.alias.end && x.binding.start==y.binding.start && x.binding.end==y.binding.end);i+=1;}}
@test @effects(filesystem,console)
fn void source_provenance_matches_independent_golden_and_detaches(){let original=fixture::original();var usize work=0;let p=snapshot(ref original,ref mut work);work=0;let data=encoded(ref p,ref mut work);let expected=golden();assert(bytes::equal(wire::content(ref data),bytes::view(ref expected)));let restored=detached();same(ref p,ref restored);work=0;let again=encoded(ref restored,ref mut work);assert(bytes::equal(wire::content(ref data),wire::content(ref again)) && model::count(ref restored)==3 && model::import_count(ref restored)==2);let first=model::import_at(ref restored,0);assert(first.declaration==3 && bytes::equal(model::text(ref restored,first.alias),b"a0"));
    let empty=bindings::empty();work=0;let blank=snapshot(ref empty,ref mut work);work=0;let blank_data=encoded(ref blank,ref mut work);assert(wire::content(ref blank_data).len==32);work=0;let copy=decoded(wire::content(ref blank_data),ref mut work);assert(model::count(ref copy)==0 && model::names(ref copy).len==0);
}
@test @effects(filesystem,console)
fn void source_provenance_codec_preserves_exact_work_and_rejects_exhaustion(){let p=detached();var usize cost=0;let data=encoded(ref p,ref mut cost);var usize work=16777216-cost;let exact=encoded(ref p,ref mut work);assert(work==16777216 && bytes::equal(wire::content(ref data),wire::content(ref exact)));work=16777216-cost+1;assert(switch(encoder::encode(ref p,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);
    cost=0;let copy=decoded(wire::content(ref data),ref mut cost);work=16777216-cost;let fit=decoded(wire::content(ref data),ref mut work);assert(work==16777216);same(ref copy,ref fit);work=16777216-cost+1;assert(switch(decoder::decode(wire::content(ref data),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);work=16777217;assert(switch(decoder::decode(wire::content(ref data),ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work==16777217);
}
@test @effects(filesystem,console)
fn void source_rebinding_accepts_source_order_and_owner_renumbering(){let saved=detached();let current=fixture::input(1);var usize work=0;let p=switch(rebinding::rebind(ref saved,fixture::sources(ref current),fixture::owners(ref current),fixture::imports(ref current),fixture::coordinate_table(ref current),ref mut work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return;}};assert(bindings::module_at(ref p,1).owner==7 && bindings::module_at(ref p,2).owner==8 && bindings::import_at(ref p,0).module_index==1 && bindings::import_at(ref p,0).target==2);
    let original=fixture::original();let old_input=fixture::input(0);work=0;let expected=switch(bindings::definition(ref original,0,v::at<syntax::Parsed>(fixture::sources(ref old_input),0),3,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};work=0;let actual=switch(bindings::definition(ref p,1,v::at<syntax::Parsed>(fixture::sources(ref current),1),3,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(bytes::equal(bytes::view(ref expected),bytes::view(ref actual)));
}
@test @effects(filesystem,console)
fn void source_rebinding_rejects_stale_bytes_provider_and_coordinate_drift(){let saved=detached();var usize mode=2;while(mode<=6){let current=fixture::input(mode);var usize work=17;assert(switch(rebinding::rebind(ref saved,fixture::sources(ref current),fixture::owners(ref current),fixture::imports(ref current),fixture::coordinate_table(ref current),ref mut work)){case .ok(_):{false}case .err(code):{code==if(mode==2){12}else{9}}} && work>17);mode+=1;}
    let current=fixture::input(0);var usize cost=0;let p=switch(rebinding::rebind(ref saved,fixture::sources(ref current),fixture::owners(ref current),fixture::imports(ref current),fixture::coordinate_table(ref current),ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=16777216-cost;let fit=switch(rebinding::rebind(ref saved,fixture::sources(ref current),fixture::owners(ref current),fixture::imports(ref current),fixture::coordinate_table(ref current),ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(work==16777216 && bindings::count(ref fit)==3);work=16777216-cost+1;assert(switch(rebinding::rebind(ref saved,fixture::sources(ref current),fixture::owners(ref current),fixture::imports(ref current),fixture::coordinate_table(ref current),ref mut work)){case .ok(_):{false}case .err(code):{code==10}} && work<=16777216);
}
@test @effects(filesystem)
fn void source_provenance_rejects_truncation_invalid_spans_and_plumbing(){let data=golden();let raw=bytes::view(ref data);assert(raw.len==998);var usize i=0;while(i<raw.len){assert(rejected(raw[..i])!=0);i+=1;}var trailing=copied(raw);assert(bytes::append_byte(ref mut trailing,0).is_ok && rejected(bytes::view(ref trailing))==1001);
    var usize mode=0;while(mode<15){var bad=copied(raw);let view=bytes::view_mut(ref mut bad);if(mode==0){view[7]=49;}else if(mode==1){view[8]=255;}else if(mode==2){view[16]=1;}else if(mode==3){view[31]=1;}else if(mode==4){view[39]=0;}else if(mode==5){view[56]=1;}else if(mode==6){view[71]=255;}else if(mode==7){view[72]=1;}else if(mode==8){view[80]=1;}else if(mode==9){view[151]=0;}else if(mode==10){view[400]=255;}else if(mode==11){view[415]=1;}else if(mode==12){view[423]=1;}else if(mode==13){view[431]=2;}else{view[520]=255;}assert(rejected(bytes::view(ref bad))!=0);mode+=1;}
    var bad=copied(raw);bytes::view_mut(ref mut bad)[528]=47;assert(rejected(bytes::view(ref bad))==1001);
}
