module tests::module_program_artifact_owner;
import tests::module_body_artifact_sample as sample;
import tests::module_body_artifact_cleanup as fixture_clean;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_verify as clean_proof;
import module_unit_cleanup_mir_model as clean;
import module_unit_interfaces as interfaces;
import module_body_artifact_encode as body_encoder;
import module_body_artifact_decode as body_decoder;
import candidate_artifact_wire as wire;
import module_unit_sdk_sources as sdk;
import module_state_layout as state;
import std::bytes as bytes;
import std::vec as v;

fn Result<ir::Program,u32> restored(bool project,bool omit_body,bool with_state,ref mut usize work){
    let original=sample::build();let cleaned=if(project){fixture_clean::project()}else{fixture_clean::build()};
    var v::Vec<ir::Body> bodies=v::empty<ir::Body>();
    if(!omit_body){var usize codec_work=0;let payload=switch(body_encoder::encode(ref original,1,ref mut codec_work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        codec_work=0;let body=switch(body_decoder::decode(wire::content(ref payload),ref cleaned,1,ref mut codec_work)){case .ok(p):{p}case .err(code):{return .err(code);}};
        if(!v::push<ir::Body>(ref mut bodies,body).is_ok){return .err(1003);}
    }
    return switch(ir::assemble_artifact(if(with_state){valid_state()}else{state::empty()},sdk::empty(),cleaned,bodies,v::empty<ir::Layout>(),v::empty<interfaces::Field>(),v::empty<interfaces::Callable>(),v::empty<usize>(),bytes::empty(),work)){case .ok(p):{.ok(p)}case .err(e):{.err(e.code)}};
}
fn ir::Program assembled(bool project,ref mut usize work){return switch(restored(project,false,false,work)){case .ok(p):{p}case .err(_):{assert(false);return assembled(project,work);}};}

@test
fn void restored_program_owner_reenters_full_verification_with_zero_history(){var usize work=0;let p=assembled(false,ref mut work);assert(ir::count(ref p)==1 && ir::work(ref p)==0 && !clean::project(ir::cleanup_ir(ref p)) && proof::check(ref p).code==0);
    var usize clean_work=0;assert(clean_proof::check_metered(ir::cleanup_ir(ref p),ref mut clean_work).code==0 && clean_work>8192 && work>clean_work+8192);
    var usize measured=0;assert(proof::check_metered(ref p,ref mut measured).code==0 && measured==work);
    let cost=work;work=16777216-cost;let exact=assembled(false,ref mut work);assert(work==16777216 && ir::work(ref exact)==0);
    work=16777216-cost+1;assert(switch(restored(false,false,false,ref mut work)){case .ok(_):{false}case .err(code):{code==1002}} && work<=16777216);
    work=16777217;assert(proof::check_metered(ref p,ref mut work).code==1002 && work==16777217);
}
@test
fn void project_checkpoints_report_the_same_fresh_cost(){var usize ordinary=0;let a=assembled(false,ref mut ordinary);var usize project=0;let b=assembled(true,ref mut project);
    assert(clean::project(ir::cleanup_ir(ref b)) && ir::work(ref b)==0 && project==ordinary);
    var usize measured=19;assert(proof::check_metered(ref b,ref mut measured).code==0 && measured==project+19);
    var usize fit=16777216-project;let c=assembled(true,ref mut fit);assert(fit==16777216 && ir::count(ref c)==1);
}
@test
fn void rejected_reconstruction_reports_cleanup_and_state_work(){let reference=fixture_clean::build();var usize expected=0;assert(clean_proof::check_metered(ref reference,ref mut expected).code==0);
    var usize used=7;assert(switch(restored(false,true,false,ref mut used)){case .ok(_):{false}case .err(code):{code==1001}} && used==expected+7);
    used=7;assert(switch(restored(true,true,false,ref mut used)){case .ok(_):{false}case .err(code):{code==1001}} && used==expected+7);
    var v::Vec<state::Layout> layouts=v::empty<state::Layout>();assert(v::push<state::Layout>(ref mut layouts,state::Layout{.owner=0,.module_index=0,.declaration=1,.start=0,.end=1,.uid_start=0,.uid_end=1,.field_start=0,.field_count=0,.hash=[0;8]}).is_ok);
    let names=switch(bytes::copy(b"a")){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};
    let states=state::assemble(names,layouts,v::empty<state::Field>());var usize state_cost=0;assert(state::validate(ref states,ref mut state_cost)==9 && state_cost>0);
    used=7;assert(switch(ir::assemble_artifact(states,sdk::empty(),fixture_clean::build(),v::empty<ir::Body>(),v::empty<ir::Layout>(),v::empty<interfaces::Field>(),v::empty<interfaces::Callable>(),v::empty<usize>(),bytes::empty(),ref mut used)){case .ok(_):{false}case .err(e):{e.code==1001}} && used==state_cost+7);
}

fn state::Plan valid_state(){
    let names=switch(bytes::copy(b"a")){case .ok(p):{p}case .err(_):{assert(false);bytes::empty()}};
    var v::Vec<state::Field> fields=v::empty<state::Field>();var usize measured=0;
    let hash=switch(state::layout_digest(b"a",ref names,ref fields,0,0,ref mut measured)){case .ok(h):{h}case .err(_):{assert(false);[0;8]}};
    var v::Vec<state::Layout> layouts=v::empty<state::Layout>();assert(v::push<state::Layout>(ref mut layouts,state::Layout{.owner=0,.module_index=0,.declaration=1,.start=0,.end=1,.uid_start=0,.uid_end=1,.field_start=0,.field_count=0,.hash}).is_ok);
    return state::assemble(names,layouts,fields);
}
@test
fn void valid_state_verification_is_charged_once_across_checkpoints(){let states=valid_state();var usize state_cost=0;assert(state::validate(ref states,ref mut state_cost)==0 && state_cost>0);
    var usize ordinary=0;let plain=assembled(false,ref mut ordinary);
    var usize cost=0;let p=switch(restored(false,false,true,ref mut cost)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(cost==ordinary+state_cost && state::count(ir::state_layouts(ref p))==1);
    var usize project=0;let q=switch(restored(true,false,true,ref mut project)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(project==cost && proof::check(ref q).code==0);
}
