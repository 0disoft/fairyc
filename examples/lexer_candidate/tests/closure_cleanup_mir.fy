module tests::closure_cleanup_mir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cleanup_mir_verify as proof;
import module_cleanup_artifact_decode as decode;
import candidate_artifact_wire as wire;
import candidate_limits as limits;
import module_types as types;
import std::vec as v;

fn types::Type node(u32 kind,u32 scalar,usize module_index,usize declaration,usize first,usize second){return types::Type{.kind,.scalar,.module_index,.declaration,.first,.second};}
fn types::Type leaf(){return node(23,0,0,0,0,0);}
fn clean::Glue trivial(){return clean::Glue{.kind=0,.frame=0,.nonempty=false,.effects=0,.abilities=0,.field_start=0,.field_count=0};}
fn v::Vec<types::Type> graph(types::Type t){var out=v::empty<types::Type>();assert(v::push<types::Type>(ref mut out,t).is_ok);return out;}
fn v::Vec<clean::Glue> glue(clean::Glue g){var out=v::empty<clean::Glue>();assert(v::push<clean::Glue>(ref mut out,g).is_ok);return out;}
fn u32 code(Result<clean::Program,clean::Problem> result){return switch(result){case .ok(_):{0}case .err(e):{e.code}};}
fn Result<clean::Program,clean::Problem> assemble(v::Vec<types::Type> records,v::Vec<clean::Glue> glues,u32 mode,bool project,bool aggregates,usize work){
    let frames=v::empty<clean::Frame>();let fields=v::empty<clean::Field>();let operands=v::empty<clean::Operand>();let drops=v::empty<clean::Drop>();
    if(mode==1){return clean::assemble(records,frames,glues,fields,operands,drops,work);}
    if(mode==2){return clean::assemble_project(records,frames,glues,fields,operands,drops,work);}
    if(mode==3){return clean::assemble_aggregates(records,frames,glues,fields,operands,drops,work);}
    if(mode==4){return clean::assemble_project_aggregates(records,frames,glues,fields,operands,drops,work);}
    var usize measured=0;
    if(mode==5){let result=clean::assemble_artifact(records,frames,glues,fields,operands,drops,project,ref mut measured);assert(measured>=8193);return result;}
    if(mode==6){let result=clean::assemble_aggregate_artifact(records,frames,glues,fields,operands,drops,project,ref mut measured);assert(measured>=8193);return result;}
    return clean::assemble_closures(records,frames,glues,fields,operands,drops,work,project,aggregates);
}

@test fn void closure_cleanup_leaf_is_explicit_trivial_and_modes_are_independent(){
    var u32 mode=0;while(mode<=6){let scalar=switch(assemble(graph(node(1,1,0,0,0,0)),glue(trivial()),mode,false,false,0)){case .ok(p):{p}case .err(_):{assert(false);return;}};assert(clean::closures(ref scalar)==(mode==0) && clean::project(ref scalar)==(mode==2 || mode==4) && clean::aggregates(ref scalar)==(mode==3 || mode==4 || mode==6));assert(code(assemble(graph(leaf()),glue(trivial()),mode,false,false,0))==if(mode==0){0}else{1001});mode+=1;}
    var usize flags=0;while(flags<4){let project=(flags&1)!=0;let aggregates=(flags&2)!=0;let p=switch(assemble(graph(leaf()),glue(trivial()),0,project,aggregates,0)){case .ok(p):{p}case .err(_):{assert(false);return;}};
        assert(clean::closures(ref p) && clean::project(ref p)==project && clean::aggregates(ref p)==aggregates && clean::type_count(ref p)==1 && clean::glue_count(ref p)==1 && clean::drop_count(ref p)==0 && clean::frame_count(ref p)==0 && clean::operand_count(ref p)==0);
        let g=clean::glue_at(ref p,1);assert(g.kind==0 && g.frame==0 && !g.nonempty && g.effects==0 && g.abilities==0 && g.field_start==0 && g.field_count==0);
        var usize measured=0;assert(proof::check_metered(ref p,ref mut measured).code==0 && measured==40962);flags+=1;
    }
    // A scalar tuple and aggregate keep their existing admission bit.
    flags=0;while(flags<4){var records=graph(node(1,1,0,0,0,0));assert(v::push<types::Type>(ref mut records,node(22,0,0,0,1,0)).is_ok && v::push<types::Type>(ref mut records,node(20,0,0,1,2,0)).is_ok && v::push<types::Type>(ref mut records,leaf()).is_ok);var glues=glue(trivial());var usize i=1;while(i<4){assert(v::push<clean::Glue>(ref mut glues,trivial()).is_ok);i+=1;}assert(code(assemble(records,glues,0,(flags&1)!=0,(flags&2)!=0,0))==if((flags&2)!=0){0}else{1001});flags+=1;}
}

@test fn void closure_cleanup_rejects_payloads_duplicates_and_wrappers(){
    var usize i=0;while(i<5){var t=leaf();if(i==0){t.scalar=1;}else if(i==1){t.module_index=1;}else if(i==2){t.declaration=1;}else if(i==3){t.first=1;}else{t.second=1;}assert(code(assemble(graph(t),glue(trivial()),0,false,true,0))==1001);i+=1;}
    var records=graph(leaf());assert(v::push<types::Type>(ref mut records,leaf()).is_ok);var glues=glue(trivial());assert(v::push<clean::Glue>(ref mut glues,trivial()).is_ok);assert(code(assemble(records,glues,0,false,true,0))==1001);
    var u32 kind=5;while(kind<=22){if(kind<=8 || kind==12 || kind==13 || kind==16 || kind==17 || kind==18 || kind==20 || kind==21 || kind==22){var wrapped=graph(leaf());let t=node(kind,0,0,if(kind==13 || kind==20 || kind==21){1}else{0},1,if(kind==6 || kind==16){1}else{0});assert(v::push<types::Type>(ref mut wrapped,t).is_ok);var gs=glue(trivial());var g=trivial();g.kind=if(kind==5){2}else if(kind==6){3}else if(kind==13 || kind==21){1}else{0};assert(v::push<clean::Glue>(ref mut gs,g).is_ok);assert(code(assemble(wrapped,gs,0,false,true,0))==1001);}kind+=1;}
    // The Result error alternative cannot smuggle the leaf either.
    var result=graph(node(1,1,0,0,0,0));assert(v::push<types::Type>(ref mut result,leaf()).is_ok && v::push<types::Type>(ref mut result,node(6,0,0,0,1,2)).is_ok);var result_glues=glue(trivial());assert(v::push<clean::Glue>(ref mut result_glues,trivial()).is_ok);var tagged=trivial();tagged.kind=3;assert(v::push<clean::Glue>(ref mut result_glues,tagged).is_ok);assert(code(assemble(result,result_glues,0,false,true,0))==1001);
    var tuple=graph(node(1,1,0,0,0,0));assert(v::push<types::Type>(ref mut tuple,leaf()).is_ok && v::push<types::Type>(ref mut tuple,node(22,0,0,0,1,2)).is_ok);var tuple_glues=glue(trivial());assert(v::push<clean::Glue>(ref mut tuple_glues,trivial()).is_ok && v::push<clean::Glue>(ref mut tuple_glues,trivial()).is_ok);assert(code(assemble(tuple,tuple_glues,0,false,true,0))==1001);
    // Invalid links, unsupported future records, and noncanonical payload words.
    assert(code(assemble(graph(node(23,4294967295,0,0,0,0)),glue(trivial()),0,false,false,0))==1001);
    assert(code(assemble(graph(node(23,0,0,0,4097,0)),glue(trivial()),0,false,false,0))==1001);
    assert(code(assemble(graph(node(24,0,0,0,0,0)),glue(trivial()),0,false,false,0))==1001);
}

fn u32 value_sites(bool drop,u32 operand_kind,bool receiver){
    var records=graph(leaf());assert(v::push<types::Type>(ref mut records,node(1,7,0,0,0,0)).is_ok);var gs=glue(trivial());assert(v::push<clean::Glue>(ref mut gs,trivial()).is_ok);
    var frames=v::empty<clean::Frame>();assert(v::push<clean::Frame>(ref mut frames,clean::Frame{.kind=if(receiver){2}else{1},.context=1,.module_index=0,.declaration=1,.result=if(receiver){2}else{1},.effects=0,.receiver=if(receiver){1}else{0},.receiver_type=if(receiver){1}else{0}}).is_ok);
    var operands=v::empty<clean::Operand>();assert(v::push<clean::Operand>(ref mut operands,clean::Operand{.frame=1,.kind=operand_kind,.id=1,.type=1,.start=1,.end=2}).is_ok);var drops=v::empty<clean::Drop>();if(drop){assert(v::push<clean::Drop>(ref mut drops,clean::Drop{.operand=1,.exit=1,.start=2,.end=3}).is_ok);}
    return code(clean::assemble_closures(records,frames,gs,v::empty<clean::Field>(),operands,drops,0,false,false));
}
fn u32 resource_field(bool closure){
    var records=graph(node(1,1,0,0,0,0));assert(v::push<types::Type>(ref mut records,leaf()).is_ok && v::push<types::Type>(ref mut records,node(3,0,0,1,0,0)).is_ok);var gs=glue(trivial());assert(v::push<clean::Glue>(ref mut gs,trivial()).is_ok);var owned=trivial();owned.kind=1;owned.abilities=1;owned.field_count=1;assert(v::push<clean::Glue>(ref mut gs,owned).is_ok);var fields=v::empty<clean::Field>();assert(v::push<clean::Field>(ref mut fields,clean::Field{.owner=3,.ordinal=0,.type=if(closure){2}else{1}}).is_ok);
    return code(clean::assemble_closures(records,v::empty<clean::Frame>(),gs,fields,v::empty<clean::Operand>(),v::empty<clean::Drop>(),0,false,false));
}
@test fn void closure_cleanup_has_no_drop_or_resource_obligation(){
    var u32 kind=1;while(kind<=2){assert(value_sites(false,kind,false)==0 && value_sites(true,kind,false)==1001 && value_sites(false,kind,true)==1001);kind+=1;}
    assert(resource_field(false)==0 && resource_field(true)==1001);
    var usize i=0;while(i<7){var g=trivial();if(i==0){g.kind=1;}else if(i==1){g.frame=1;}else if(i==2){g.nonempty=true;}else if(i==3){g.effects=1;}else if(i==4){g.abilities=1;}else if(i==5){g.field_start=1;}else{g.field_count=1;}assert(code(assemble(graph(leaf()),glue(g),0,false,false,0))==1001);i+=1;}
}

fn Result<wire::Payload,u32> old_wire(bool aggregates,bool project,usize kind,bool extra){
    var usize work=0;var status=wire::Status{.code=0};var out=wire::writer(if(aggregates){b"FYCLEAN1"}else{b"FYCLEAN0"},ref mut status,ref mut work);
    wire::number(ref mut out,if(project){1}else{0},ref mut status,ref mut work);wire::number(ref mut out,1,ref mut status,ref mut work);wire::number(ref mut out,kind,ref mut status,ref mut work);wire::number(ref mut out,if(kind==1){1}else{0},ref mut status,ref mut work);
    var usize i=0;while(i<15){wire::number(ref mut out,0,ref mut status,ref mut work);i+=1;}if(extra){wire::number(ref mut out,23,ref mut status,ref mut work);}
    return wire::finish(out,status);
}
@test fn void closure_cleanup_keeps_old_artifact_decoders_closed(){
    var usize flags=0;while(flags<4){let aggregates=(flags&2)!=0;let project=(flags&1)!=0;var usize i=0;while(i<4){let usize kind=if(i==0 || i==3){1}else if(i==1){23}else{24};let data=switch(old_wire(aggregates,project,kind,i==3)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=0;
        let result=if(aggregates){decode::decode_aggregates(wire::content(ref data),ref mut work)}else{decode::decode(wire::content(ref data),ref mut work)};assert(switch(result){case .ok(p):{i==0 && !clean::closures(ref p) && clean::project(ref p)==project && clean::aggregates(ref p)==aggregates}case .err(e):{i!=0 && e==1001}} && work!=0);i+=1;
    }flags+=1;}
}

@test fn void closure_cleanup_preserves_work_and_record_bounds(){
    assert(code(assemble(graph(leaf()),glue(trivial()),0,false,false,1048576))==0 && code(assemble(graph(leaf()),glue(trivial()),0,false,false,1048577))==1002);
    assert(code(assemble(graph(leaf()),glue(trivial()),0,true,true,limits::project_work()))==0 && code(assemble(graph(leaf()),glue(trivial()),0,true,true,limits::project_work()+1))==1002);
    let p=switch(assemble(graph(leaf()),glue(trivial()),0,false,false,0)){case .ok(p):{p}case .err(_):{assert(false);return;}};var usize work=1048576-40962;assert(proof::check_metered(ref p,ref mut work).code==0 && work==1048576);work=1048576-40961;assert(proof::check_metered(ref p,ref mut work).code==1002 && work<=1048576);work=1048577;assert(proof::check_metered(ref p,ref mut work).code==1002 && work==1048577);
    assert(code(assemble(graph(leaf()),v::empty<clean::Glue>(),0,false,false,0))==1001 && code(assemble(v::empty<types::Type>(),v::empty<clean::Glue>(),0,false,false,0))==1002);
    // Invalid synthetic records only exercise count preflight, never publication.
    var records=v::empty<types::Type>();var gs=v::empty<clean::Glue>();var usize i=0;while(i<4097){assert(v::push<types::Type>(ref mut records,leaf()).is_ok && v::push<clean::Glue>(ref mut gs,trivial()).is_ok);i+=1;}assert(code(assemble(records,gs,0,false,false,0))==1002);
    var operands=v::empty<clean::Operand>();i=0;while(i<16385){assert(v::push<clean::Operand>(ref mut operands,clean::Operand{.frame=1,.kind=1,.id=1,.type=1,.start=1,.end=2}).is_ok);i+=1;}assert(code(clean::assemble_closures(graph(leaf()),v::empty<clean::Frame>(),glue(trivial()),v::empty<clean::Field>(),operands,v::empty<clean::Drop>(),0,false,false))==1002);
    var usize project=0;while(project<2){var drops=v::empty<clean::Drop>();i=0;while(i<=clean::drop_limit(project==1)){assert(v::push<clean::Drop>(ref mut drops,clean::Drop{.operand=1,.exit=1,.start=1,.end=2}).is_ok);i+=1;}assert(code(clean::assemble_closures(graph(leaf()),v::empty<clean::Frame>(),glue(trivial()),v::empty<clean::Field>(),v::empty<clean::Operand>(),drops,0,project==1,false))==1002);project+=1;}
}
