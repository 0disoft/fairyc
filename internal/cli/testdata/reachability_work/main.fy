module main;
import module_unit_c_reachability as reachable;
import module_unit_c_callable_index as contracts;
import module_unit_c_frame_index as targets;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_model as cfg;
import module_unit_value_model as values;
import module_unit_region_model as regions;
import module_unit_interfaces as interfaces;
import module_types as types;
import std::bytes as bytes;
import std::io as io;
import std::vec as v;

// Small traversal-unit IR, not a saved compiler corpus or a claim of whole-IR
// authentication. Every CFG goes through the real checked assembler, which
// must derive its own drop summary. No fixture writes Graph.drop_blocks.
resource Inputs {ir::Program program;v::Vec<cfg::Graph> graphs;cleanup{}}
fn types::Type type(u32 kind,u32 scalar,usize declaration,usize first,usize second){
  return types::Type{.kind,.scalar,.module_index=0,.declaration,.first,.second};
}
fn clean::Glue glue(u32 kind,usize frame,usize field_start,usize field_count){
  return clean::Glue{.kind,.frame,.nonempty=kind!=0,.effects=0,.abilities=0,.field_start,.field_count};
}
fn clean::Frame frame(usize id){
  let u32 kind=if(id==4 || id==6 || id==7 || id==8){2}else if(id>=9){1}else{3};
  let usize receiver=if(id==4){2}else if(id==6){3}else if(id==7){6}else if(id==8){7}else{0};
  return clean::Frame{.kind,.context=id,.module_index=0,.declaration=if(id>=9){9}else{id},.result=1,.effects=0,.receiver=0,.receiver_type=receiver};
}
fn interfaces::Callable callable(usize declaration,bool foreign,bool template){
  return interfaces::Callable{.module_index=0,.declaration,.template,.result=1,.parameter_start=0,.parameter_count=0,.effects=0,.must_use=false,.unsafe_call=foreign,.foreign,.link_start=0,.link_end=0};
}
fn values::Invocation call(usize declaration,usize instance){
  return values::Invocation{.kind=if(instance==0){1}else{2},.expression=0,.source_index=0,.target_module=0,.target_declaration=declaration,.instance,.argument_start=0,.argument_count=0,.start=0,.end=0,.checked=true};
}
fn ir::Body body(usize id,usize copies,bool foreign_only,u32 bad){
  var v::Vec<values::Invocation> calls=v::empty<values::Invocation>();var usize i=0;
  if(id==1){while(i<copies){
    if(!foreign_only){
      assert(v::push<values::Invocation>(ref mut calls,call(2,0)).is_ok);
      // Same declaration, distinct specialization contexts: a lookup cache
      // must not merge the two targets, even across duplicate requests.
      assert(v::push<values::Invocation>(ref mut calls,call(9,9)).is_ok);
      assert(v::push<values::Invocation>(ref mut calls,call(9,10)).is_ok);
    }
    assert(v::push<values::Invocation>(ref mut calls,call(if(bad==1){91}else{90},0)).is_ok);i+=1;
  }}else if(id==2){assert(v::push<values::Invocation>(ref mut calls,call(1,0)).is_ok);}
  else if(id==3){
    // A missing target in a dead frame is never a reachable rejection.
    assert(v::push<values::Invocation>(ref mut calls,call(91,0)).is_ok);
  }else if(id==4){assert(v::push<values::Invocation>(ref mut calls,call(5,0)).is_ok);}
  else if(id==5){assert(v::push<values::Invocation>(ref mut calls,call(1,0)).is_ok);}
  return ir::assemble_body(id,0,0,0,v::empty<ir::Node>(),v::empty<ir::Statement>(),v::empty<values::Local>(),calls,v::empty<values::Argument>(),v::empty<regions::Scope>(),v::empty<regions::Exit>(),bytes::empty());
}
fn cfg::Block block(u32 kind,usize next,usize drop){
  return cfg::Block{.kind,.source=0,.target=0,.first=next,.second=0,.drop,.start=0,.end=0};
}
fn cfg::Graph graph(ref ir::Program p,usize id,usize begin,usize count){
  var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();var usize i=0;
  // Non-drop padding separates drop sites. Keep table order different from
  // execution order, so the summary must retain original block IDs, not drop
  // table IDs, successor order, or a set of cleanup types.
  assert(v::push<cfg::Block>(ref mut blocks,block(7,if(count==0){2}else{3},0)).is_ok);
  assert(v::push<cfg::Block>(ref mut blocks,block(6,0,0)).is_ok);
  while(i<count){
    let usize next=if(i+1==count){2}else{5+2*i};
    assert(v::push<cfg::Block>(ref mut blocks,block(5,4+2*i,begin+count-i-1)).is_ok);
    assert(v::push<cfg::Block>(ref mut blocks,block(7,next,0)).is_ok);i+=1;
  }
  return switch(cfg::assemble_deferred(p,id,1,blocks,v::empty<cfg::Deferred>(),v::empty<cfg::Capture>(),0)){
    case .ok(g):{g}case .err(_):{assert(false);graph(p,id,0,0)}
  };
}
fn Inputs inputs(usize copies,bool foreign_only,u32 bad){
  var v::Vec<types::Type> ts=v::empty<types::Type>();
  assert(v::push<types::Type>(ref mut ts,type(1,7,0,0,0)).is_ok); // 1 void
  assert(v::push<types::Type>(ref mut ts,type(3,0,2,0,0)).is_ok); // 2 root
  assert(v::push<types::Type>(ref mut ts,type(3,0,3,0,0)).is_ok); // 3 leaf
  assert(v::push<types::Type>(ref mut ts,type(5,0,0,3,0)).is_ok); // 4 Option<leaf>
  assert(v::push<types::Type>(ref mut ts,type(6,0,0,4,6)).is_ok); // 5 Result<Option<leaf>,err>
  assert(v::push<types::Type>(ref mut ts,type(3,0,6,0,0)).is_ok); // 6 err
  assert(v::push<types::Type>(ref mut ts,type(3,0,7,0,0)).is_ok); // 7 dead resource
  assert(v::push<types::Type>(ref mut ts,type(1,1,0,0,0)).is_ok); // 8 plain field
  var v::Vec<clean::Glue> glues=v::empty<clean::Glue>();
  assert(v::push<clean::Glue>(ref mut glues,glue(0,0,0,0)).is_ok);
  assert(v::push<clean::Glue>(ref mut glues,glue(1,4,0,3)).is_ok);
  assert(v::push<clean::Glue>(ref mut glues,glue(1,6,0,0)).is_ok);
  assert(v::push<clean::Glue>(ref mut glues,glue(2,0,0,0)).is_ok);
  assert(v::push<clean::Glue>(ref mut glues,glue(3,0,0,0)).is_ok);
  assert(v::push<clean::Glue>(ref mut glues,glue(1,7,0,0)).is_ok);
  assert(v::push<clean::Glue>(ref mut glues,glue(1,8,0,0)).is_ok);
  assert(v::push<clean::Glue>(ref mut glues,glue(0,0,0,0)).is_ok);
  var v::Vec<clean::Field> fields=v::empty<clean::Field>();
  assert(v::push<clean::Field>(ref mut fields,clean::Field{.owner=2,.ordinal=0,.type=5}).is_ok);
  assert(v::push<clean::Field>(ref mut fields,clean::Field{.owner=2,.ordinal=1,.type=3}).is_ok);
  assert(v::push<clean::Field>(ref mut fields,clean::Field{.owner=2,.ordinal=2,.type=8}).is_ok);
  var v::Vec<clean::Frame> frames=v::empty<clean::Frame>();var v::Vec<ir::Body> bodies=v::empty<ir::Body>();var usize i=1;
  while(i<=10){assert(v::push<clean::Frame>(ref mut frames,frame(i)).is_ok);assert(v::push<ir::Body>(ref mut bodies,body(i,copies,foreign_only,bad)).is_ok);i+=1;}
  var v::Vec<clean::Operand> operands=v::empty<clean::Operand>();var v::Vec<clean::Drop> drops=v::empty<clean::Drop>();
  let usize total=if(foreign_only){0}else{copies+2};i=1;
  while(i<=total){
    let usize owner=if(i<=copies){1}else if(i==copies+1){4}else{5};
    let usize type=if(i<=copies && bad==3){1}else if(i==copies+1){6}else{2};
    assert(v::push<clean::Operand>(ref mut operands,clean::Operand{.frame=owner,.kind=1,.id=i,.type,.start=0,.end=0}).is_ok);
    assert(v::push<clean::Drop>(ref mut drops,clean::Drop{.operand=i,.exit=1,.start=0,.end=0}).is_ok);i+=1;
  }
  let cleanup_ir=clean::reachability_fixture(ts,frames,glues,fields,operands,drops);
  var v::Vec<interfaces::Callable> callables=v::empty<interfaces::Callable>();i=1;
  while(i<=5){if(i!=4){assert(v::push<interfaces::Callable>(ref mut callables,callable(i,false,false)).is_ok);}i+=1;}
  assert(v::push<interfaces::Callable>(ref mut callables,callable(9,false,true)).is_ok);
  assert(v::push<interfaces::Callable>(ref mut callables,callable(90,bad!=2,false)).is_ok);
  let program=ir::reachability_fixture(cleanup_ir,bodies,callables);
  var v::Vec<cfg::Graph> graphs=v::empty<cfg::Graph>();i=1;
  while(i<=10){
    let usize begin=if(i==1){1}else if(i==4){copies+1}else{copies+2};
    let usize count=if(foreign_only){0}else if(i==1){copies}else if(i==4 || i==5){1}else{0};
    assert(v::push<cfg::Graph>(ref mut graphs,graph(ref program,i,begin,count)).is_ok);i+=1;
  }
  return Inputs{.program,.graphs};
}
fn void summary(ref cfg::Graph g){
  var usize row=0;var usize id=1;
  while(id<=cfg::count(g)){
    let n=cfg::at(g,id);
    if(n.kind==5){assert(row<cfg::drop_count(g));assert(cfg::drop_block(g,row)==id);row+=1;}
    id+=1;
  }
  assert(row==cfg::drop_count(g));
}
fn void summaries(ref Inputs input,usize copies,bool foreign_only){
  assert(v::len<cfg::Graph>(ref input.graphs)==10);var usize i=0;
  while(i<10){let g=v::at<cfg::Graph>(ref input.graphs,i);summary(g);
    let usize expected=if(foreign_only){0}else if(i==0){copies}else if(i==3 || i==4){1}else{0};
    assert(cfg::drop_count(g)==expected);
    // Public check must accept an assembled graph without appending another
    // summary or changing the checked block IDs on repeated checks.
    assert(cfg::check(ref input.program,g).code==0);summary(g);assert(cfg::drop_count(g)==expected);i+=1;
  }
}
fn void exact(ref reachable::Plan plan,bool foreign_only){
  assert(reachable::count(plan)==if(foreign_only){1}else{8});
  assert(v::len<u8>(reachable::frames(plan))==10 && v::len<u8>(reachable::drops(plan))==8);var usize i=1;
  while(i<=10){let expected=if(foreign_only){i==1}else{i!=3 && i!=8};assert(reachable::frame(plan,i)==expected);i+=1;}
  i=1;while(i<=8){let marked=*v::at<u8>(reachable::drops(plan),i-1)!=0;assert(marked==(!foreign_only && i>=2 && i<=6));i+=1;}
}
fn u32 rejected(Result<reachable::Plan,u32> result){return switch(result){case .ok(_):{assert(false);0}case .err(e):{e}};}
fn reachable::Plan accepted(Result<reachable::Plan,u32> result){return switch(result){case .ok(p):{p}case .err(_):{assert(false);accepted(.err(1001))}};}
@effects(console) fn void exercise(usize copies,bool foreign_only){
  let input=inputs(copies,foreign_only,0);summaries(ref input,copies,foreign_only);var usize index_work=0;
  let callables=switch(contracts::build(ref input.program,ref mut index_work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  let indexed=switch(targets::build(ref input.program,ref callables,ref mut index_work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  var usize work=0;let plan=accepted(reachable::build(ref input.program,ref input.graphs,ref callables,ref indexed,1,ref mut work));exact(ref plan,foreign_only);
  let used=work;assert(used>0 && used<1048576);
  // Measure this implementation, then replay the same immutable IR/indexes.
  // This asserts the precise shared-work boundary without predicting savings.
  work=1048576-used;let boundary=accepted(reachable::build(ref input.program,ref input.graphs,ref callables,ref indexed,1,ref mut work));exact(ref boundary,foreign_only);assert(work==1048576);
  work=1048576-used+1;assert(rejected(reachable::build(ref input.program,ref input.graphs,ref callables,ref indexed,1,ref mut work))==1002 && work<=1048576);
  work=1048576;assert(rejected(reachable::build(ref input.program,ref input.graphs,ref callables,ref indexed,1,ref mut work))==1002 && work==1048576);
  work=0;assert(rejected(reachable::build(ref input.program,ref input.graphs,ref callables,ref indexed,0,ref mut work))==1001);
  work=0;assert(rejected(reachable::build(ref input.program,ref input.graphs,ref callables,ref indexed,11,ref mut work))==1001);
  work=0;let replay=accepted(reachable::build(ref input.program,ref input.graphs,ref callables,ref indexed,1,ref mut work));exact(ref replay,foreign_only);assert(work==used);summaries(ref input,copies,foreign_only);
  discard io::write(if(foreign_only){"foreign copies="}else{"graph copies="});discard io::write_usize(copies);discard io::write(" work=");discard io::write_usize(used);discard io::write("\n");
}
fn void bad_inputs(u32 bad){
  let input=inputs(1,false,bad);var usize work=0;
  let callables=switch(contracts::build(ref input.program,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  let indexed=switch(targets::build(ref input.program,ref callables,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
  work=0;assert(rejected(reachable::build(ref input.program,ref input.graphs,ref callables,ref indexed,1,ref mut work))==1001);
}
fn void bad_cfg(){
  let input=inputs(1,false,0);var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();
  assert(v::push<cfg::Block>(ref mut blocks,block(6,0,0)).is_ok);
  assert(v::push<cfg::Block>(ref mut blocks,block(5,1,2)).is_ok);
  var v::Vec<cfg::Capture> captures=v::empty<cfg::Capture>();
  assert(v::push<cfg::Capture>(ref mut captures,cfg::Capture{.deferred=0,.position=0,.source=0,.type=0}).is_ok);
  // Blocks and the drop site are valid. The orphan capture fails in the
  // existing deferred tail check, after summary collection. No Graph may be
  // published when that later validation rejects the candidate.
  switch(cfg::assemble_deferred(ref input.program,4,2,blocks,v::empty<cfg::Deferred>(),captures,0)){
    case .ok(_):{assert(false);}case .err(e):{assert(e.code==1001);}
  };
}
@effects(console) fn i32 main(){
  exercise(1,false);exercise(16,false);exercise(128,false);exercise(128,true);
  bad_inputs(1);bad_inputs(2);bad_inputs(3);bad_cfg();
  discard io::write("summary=ordered set=exact budget=boundary rejection=1001,1002 inputs=reused\n");return 0;
}
