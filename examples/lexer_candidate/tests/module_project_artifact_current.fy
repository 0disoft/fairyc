module tests::module_project_artifact_current;
import tests::module_project_artifact_current_fixture as fixture;
import module_project_artifact_current as binding;
import module_project_artifact_model as model;
import module_identity_artifact_model as identities;
import module_source_artifact_model as stamps;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import generic_package_bindings as provenance;
import generic_source_plan as source;
import candidate_artifact_wire as wire;
import std::bytes as bytes;
import std::io as io;
@effects(console) fn binding::Rebound bound(ref u8[] raw,ref fixture::Current current,ref mut usize work){let saved=fixture::restored(raw);return switch(binding::build_sources(saved,fixture::sources(current),fixture::owners(current),fixture::imports(current),fixture::coordinate_table(current),work)){case .ok(p):{p}case .err(code):{discard io::write_stderr_u32(code);assert(false);return bound(raw,current,work);}};}
@effects(console) fn u32 rejected(ref u8[] raw,ref fixture::Current current,ref mut usize work){let saved=fixture::restored(raw);return switch(binding::build_sources(saved,fixture::sources(current),fixture::owners(current),fixture::imports(current),fixture::coordinate_table(current),work)){case .ok(_):{0}case .err(code):{code}};}
@test @effects(filesystem,console)
fn void current_source_rebind_rederives_handles_across_source_and_owner_reordering(){let packet=fixture::packet();let raw=wire::content(ref packet);let original=fixture::restored(raw);let ids=model::source_identities(ref original);let span=fixture::identity_span(raw);var changed=fixture::copied(raw);var usize i=1;
    while(i<=identities::type_count(ids)+identities::frame_count(ids)){let row=if(i<=identities::type_count(ids)){identities::type_at(ids,i)}else{identities::frame_at(ids,i-identities::type_count(ids))};if(row.node!=0){let at=if(i<=identities::type_count(ids)){span.start+16+(i-1)*24}else{span.start+24+identities::type_count(ids)*24+(i-identities::type_count(ids)-1)*24};let view=bytes::view_mut(ref mut changed);var usize j=0;while(j<8){view[at+j]=0;j+=1;}view[at+6]=16;}i+=1;}
    let current=fixture::current(1);var usize work=0;let rebound=bound(bytes::view(ref changed),ref current,ref mut work);assert(binding::module_count(ref rebound)==10);let selected=binding::source_identities(ref rebound);assert(bytes::equal(identities::names(ids),identities::names(selected)));i=0;while(i<10){assert(binding::module_at(ref rebound,i)==9-i);let old=stamps::module_at(model::source_provenance(ref original),i);assert(binding::owner_at(ref rebound,old.owner)==old.owner+7 && provenance::module_at(binding::provenance(ref rebound),9-i).owner==old.owner+7);i+=1;}i=1;var usize named=0;
    while(i<=identities::type_count(selected)+identities::frame_count(selected)){let row=if(i<=identities::type_count(selected)){identities::type_at(selected,i)}else{identities::frame_at(selected,i-identities::type_count(selected))};if(row.node!=0){assert(row.node<=source::count(binding::source_plan(ref rebound)) && row.node!=4096 && bytes::equal(identities::identity(selected,row),source::text(binding::source_plan(ref rebound),source::at(binding::source_plan(ref rebound),row.node-1).id)));named+=1;}i+=1;}assert(named>3);
}
@test @effects(filesystem,console)
fn void current_source_rebind_rejects_drift_wrong_template_ids_and_work_exhaustion(){let packet=fixture::packet();let raw=wire::content(ref packet);let current=fixture::current(0);var usize cost=0;let original=bound(raw,ref current,ref mut cost);assert(cost>0);var usize work=16777216-cost;let exact=bound(raw,ref current,ref mut work);assert(work==16777216 && binding::module_count(ref exact)==10);work=16777216-cost+1;assert(rejected(raw,ref current,ref mut work)==10 && work<=16777216);work=16777217;assert(rejected(raw,ref current,ref mut work)==10 && work==16777217);
    var usize mode=2;while(mode<=4){let drift=fixture::current(mode);work=0;assert(rejected(raw,ref drift,ref mut work)==if(mode==2){12}else{9});mode+=1;}
    let restored=fixture::restored(raw);let ids=model::source_identities(ref restored);let span=fixture::identity_span(raw);let names=span.start+32+identities::type_count(ids)*24+identities::frame_count(ids)*24;let context=ir::cleanup_ir(model::program(ref restored));var usize first=0;var usize second=0;var usize i=1;while(i<=identities::frame_count(ids)){if(identities::frame_at(ids,i).node!=0 && clean::frame_at(context,i).kind==1){if(first==0){first=i;}else{second=i;break;}}i+=1;}assert(first!=0 && second!=0);let a=identities::frame_at(ids,first);let b=identities::frame_at(ids,second);var bad=fixture::copied(raw);let view=bytes::view_mut(ref mut bad);i=0;while(i<67){let old=view[names+a.id.start+i];view[names+a.id.start+i]=view[names+b.id.start+i];view[names+b.id.start+i]=old;i+=1;}work=0;assert(rejected(bytes::view(ref bad),ref current,ref mut work)==9);
    bad=fixture::copied(raw);let before=bytes::view(ref bad)[names+a.id.start+3];bytes::view_mut(ref mut bad)[names+a.id.start+3]=if(before==97){98}else{97};work=0;assert(rejected(bytes::view(ref bad),ref current,ref mut work)==9);
}
