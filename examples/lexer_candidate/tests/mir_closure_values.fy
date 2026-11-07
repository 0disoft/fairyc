module tests::mir_closure_values;
import mir_model as m;
import mir_state as state;
import mir_verify as verify;
import hir_shapes as shapes;
import std::bytes as bytes;
import std::vec as v;

fn m::Target target(usize id,usize start,usize count){return m::Target{.foreign=false,.link_start=0,.link_end=0,.unsafe_call=false,.effects=0,.binding=id,.module_index=0,.declaration=id,.result=1,.signature_start=start,.signature_count=count,.start=0,.end=1,.external=false};}
fn m::Local local(usize function,u32 type,bool parameter){return m::Local{.function,.binding=0,.type,.parameter,.mutable=false,.start=0,.end=1};}
fn shapes::Table nominal(u32 mutation){
    if(mutation!=27 && mutation!=28){return shapes::empty();}
    var records=v::empty<shapes::Shape>();var fields=v::empty<shapes::Field>();
    assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=256,.binding=1,.field_start=0,.field_count=1,.public=false}).is_ok);
    assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=256,.type=13,.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);
    if(mutation==27){assert(v::push<shapes::Shape>(ref mut records,shapes::Shape{.type=257,.binding=2,.field_start=1,.field_count=1,.public=false}).is_ok);assert(v::push<shapes::Field>(ref mut fields,shapes::Field{.owner=257,.type=256,.start=0,.end=1,.payload_start=0,.payload_count=0}).is_ok);}
    return shapes::assemble(records,fields);
}
// Every seed is hand-built independently of source preparation/lowering. The
// helper returns its argument; captures are retained but do not run the body.
fn u32 probe(usize count,u32 mutation){
    var source=bytes::empty();assert(bytes::append(ref mut source,b"fg").is_ok);
    var targets=v::empty<m::Target>();assert(v::push<m::Target>(ref mut targets,target(1,0,1)).is_ok);
    var helper=target(2,1,count+1);if(mutation==8){helper.unsafe_call=true;}if(mutation==9){helper.effects=1;}if(mutation==10){helper.foreign=true;helper.external=true;helper.link_end=1;helper.unsafe_call=true;}
    assert(v::push<m::Target>(ref mut targets,helper).is_ok);
    var signatures=v::empty<m::Signature>();var usize i=0;while(i<count+2){assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=if(mutation==11 && i==count+1){6}else{1}}).is_ok);i+=1;}
    var functions=v::empty<m::Function>();assert(v::push<m::Function>(ref mut functions,m::Function{.target=1,.entry=1,.local_start=0,.local_count=4,.block_start=0,.block_count=2,.parameter_start=0,.parameter_count=1,.start=0,.end=1,.public=false}).is_ok);
    if(mutation!=10){assert(v::push<m::Function>(ref mut functions,m::Function{.target=2,.entry=3,.local_start=4,.local_count=count+if(mutation==25){2}else{1},.block_start=2,.block_count=if(mutation==25){3}else if(mutation==18 || mutation==19 || mutation==20){2}else{1},.parameter_start=1,.parameter_count=count+1,.start=0,.end=1,.public=mutation==12}).is_ok);}
    var locals=v::empty<m::Local>();assert(v::push<m::Local>(ref mut locals,local(1,1,true)).is_ok);assert(v::push<m::Local>(ref mut locals,local(1,13,false)).is_ok);assert(v::push<m::Local>(ref mut locals,local(1,if(mutation==2){13}else{1},false)).is_ok);assert(v::push<m::Local>(ref mut locals,local(1,1,false)).is_ok);
    if(mutation!=10){i=0;while(i<count+1){var l=local(2,if(mutation==11 && i==count){6}else{1},true);if(mutation==13){l.mutable=true;}assert(v::push<m::Local>(ref mut locals,l).is_ok);i+=1;}}
    if(mutation==25){assert(v::push<m::Local>(ref mut locals,local(2,1,false)).is_ok);}
    var edges=v::empty<usize>();assert(v::push<usize>(ref mut edges,1).is_ok);if(mutation!=10){i=0;while(i<count+1){assert(v::push<usize>(ref mut edges,5+i).is_ok);i+=1;}}
    let usize capture_start=if(count==0){0}else{v::len<usize>(ref edges)};i=0;while(i<count){assert(v::push<usize>(ref mut edges,if(mutation==4){3}else if(mutation==5){2}else{1}).is_ok);i+=1;}
    var create=state::operation(40,2,13,13,if(mutation==6){1}else{2},0,0,0,1);create.function=1;create.block=1;create.argument_start=capture_start;create.argument_count=count;if(mutation==7){create.boolean=true;}
    var operations=v::empty<m::Operation>();if(mutation!=1 && mutation!=26){assert(v::push<m::Operation>(ref mut operations,create).is_ok);}
    if(mutation==25){var check=state::operation(5,0,1,1,6,5,6,0,1);check.function=2;check.block=3;assert(v::push<m::Operation>(ref mut operations,check).is_ok);var divide=state::operation(4,7,1,1,6,5,6,0,1);divide.function=2;divide.block=4;assert(v::push<m::Operation>(ref mut operations,divide).is_ok);}
    var invoke=state::block(1,0,1);invoke.kind=8;invoke.value=if(mutation==2){3}else{2};invoke.left=if(mutation==3){3}else{1};invoke.destination=4;invoke.type=1;invoke.target=if(mutation==14){3}else{2};if(mutation==15){invoke.argument_count=1;}if(mutation==16){invoke.unsafe_call=true;}if(mutation==17){invoke.right=1;}
    if(mutation!=1 && mutation!=26){invoke.first=1;invoke.last=1;}
    if(mutation==26){invoke.kind=1;invoke.value=1;invoke.left=0;invoke.destination=0;invoke.type=0;invoke.target=0;}
    var done=state::block(1,0,1);done.kind=1;done.value=if(mutation==26){1}else{4};
    var blocks=v::empty<m::Block>();assert(v::push<m::Block>(ref mut blocks,invoke).is_ok);assert(v::push<m::Block>(ref mut blocks,done).is_ok);
    if(mutation!=10){var returned=state::block(2,0,1);returned.kind=if(mutation==20){2}else{1};returned.value=if(mutation==20){0}else{5+count};if(mutation==20){returned.target=3;}if(mutation==25){returned.kind=4;returned.value=0;returned.first=2;returned.last=2;returned.check=2;returned.left=5;returned.right=6;returned.operator=6;returned.type=1;returned.target=4;returned.alternative=5;returned.trap_mask=6;}assert(v::push<m::Block>(ref mut blocks,returned).is_ok);}
    if(mutation==25){var normal=state::block(2,0,1);normal.kind=1;normal.value=7;normal.first=3;normal.last=3;var trap=state::block(2,0,1);trap.kind=6;trap.trap_mask=6;assert(v::push<m::Block>(ref mut blocks,normal).is_ok);assert(v::push<m::Block>(ref mut blocks,trap).is_ok);}
    if(mutation==18 || mutation==19 || mutation==20){var dead=state::block(2,0,1);dead.kind=1;dead.value=5+count;
        if(mutation==18){dead.kind=5;dead.value=0;dead.callee=2;dead.destination=5;dead.argument_start=v::len<usize>(ref edges);dead.argument_count=count+1;dead.target=3;i=0;while(i<count+1){assert(v::push<usize>(ref mut edges,5+i).is_ok);i+=1;}}
        if(mutation==19){var overwrite=state::operation(2,5,1,1,0,5,0,0,1);overwrite.function=2;overwrite.block=4;assert(v::push<m::Operation>(ref mut operations,overwrite).is_ok);dead.first=2;dead.last=2;}
        assert(v::push<m::Block>(ref mut blocks,dead).is_ok);
    }
    var closures=v::empty<m::ClosureTarget>();if(mutation!=21){assert(v::push<m::ClosureTarget>(ref mut closures,m::ClosureTarget{.target=2,.captures=if(mutation==22){count+1}else{count}}).is_ok);}if(mutation==23){assert(v::push<m::ClosureTarget>(ref mut closures,m::ClosureTarget{.target=2,.captures=count}).is_ok);}
    if(mutation==24 || mutation==26){return switch(m::assemble_resources(source,bytes::empty(),targets,signatures,functions,locals,blocks,operations,edges,0,nominal(mutation),v::empty<m::Drop>())){case .err(problem):{problem.code}case .ok(p):{verify::check(ref p).code}};}
    return switch(m::assemble_closures(source,bytes::empty(),targets,signatures,functions,locals,blocks,operations,edges,0,nominal(mutation),v::empty<m::Drop>(),closures)){case .err(problem):{problem.code}case .ok(p):{verify::check(ref p).code}};
}
// An unused identity function carries the public signature without creating a
// closure, binding an external helper or importing a native pointer.
fn u32 signature_only(bool old){
    var source=bytes::empty();assert(bytes::append(ref mut source,b"fg").is_ok);var payload=bytes::empty();assert(bytes::append(ref mut payload,b"42").is_ok);
    var targets=v::empty<m::Target>();assert(v::push<m::Target>(ref mut targets,target(1,0,0)).is_ok);var identity=target(2,0,1);identity.result=13;assert(v::push<m::Target>(ref mut targets,identity).is_ok);
    var signatures=v::empty<m::Signature>();assert(v::push<m::Signature>(ref mut signatures,m::Signature{.type=13}).is_ok);
    var functions=v::empty<m::Function>();assert(v::push<m::Function>(ref mut functions,m::Function{.target=1,.entry=1,.local_start=0,.local_count=1,.block_start=0,.block_count=1,.parameter_start=0,.parameter_count=0,.start=0,.end=1,.public=false}).is_ok);assert(v::push<m::Function>(ref mut functions,m::Function{.target=2,.entry=2,.local_start=1,.local_count=1,.block_start=1,.block_count=1,.parameter_start=0,.parameter_count=1,.start=0,.end=1,.public=false}).is_ok);
    var locals=v::empty<m::Local>();assert(v::push<m::Local>(ref mut locals,local(1,1,false)).is_ok);assert(v::push<m::Local>(ref mut locals,local(2,13,true)).is_ok);
    var blocks=v::empty<m::Block>();var main=state::block(1,0,1);main.kind=1;main.value=1;main.first=1;main.last=1;assert(v::push<m::Block>(ref mut blocks,main).is_ok);var returned=state::block(2,0,1);returned.kind=1;returned.value=2;assert(v::push<m::Block>(ref mut blocks,returned).is_ok);
    var operations=v::empty<m::Operation>();var constant=state::operation(1,1,1,1,0,0,0,0,1);constant.function=1;constant.block=1;constant.payload_end=2;assert(v::push<m::Operation>(ref mut operations,constant).is_ok);
    var edges=v::empty<usize>();assert(v::push<usize>(ref mut edges,2).is_ok);
    if(old){return switch(m::assemble_resources(source,payload,targets,signatures,functions,locals,blocks,operations,edges,0,shapes::empty(),v::empty<m::Drop>())){case .err(e):{e.code}case .ok(p):{verify::check(ref p).code}};}
    return switch(m::assemble_closures(source,payload,targets,signatures,functions,locals,blocks,operations,edges,0,shapes::empty(),v::empty<m::Drop>(),v::empty<m::ClosureTarget>())){case .err(e):{e.code}case .ok(p):{verify::check(ref p).code}};
}
@test
pub fn void closure_mir_checks_targets_captures_and_initialization(){
    var usize count=0;while(count<=4){assert(probe(count,0)==0);count+=1;}
    assert(probe(5,0)!=0);
    assert(probe(1,25)==0);
    assert(probe(1,24)==4 && probe(1,26)==4 && probe(1,27)!=0 && probe(1,28)==0);
    assert(probe(1,2)==1002 && probe(1,3)==1002 && probe(1,4)==1002);
    assert(probe(1,21)==4);
    assert(signature_only(false)==0 && signature_only(true)==4);
    var u32 mutation=1;while(mutation<=24){assert(probe(1,mutation)!=0);mutation+=1;}
}
