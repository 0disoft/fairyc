module tests::module_unit_cleanup_mir;
import module_unit_cleanup_mir_model as ir;
import module_unit_cleanup_mir_lower as lower;
import module_unit_cleanup_mir_verify as proof;
import module_unit_moves as moves;
import module_unit_values as values;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import module_types as types;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;

fn syntax::Parsed parsed(ref u8[] text){let b=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};let s=switch(scanner::collect(b,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};let p=switch(syntax::parse(s,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;}
fn ir::Program retained(ref u8[] root,ref u8[] provider){
    var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(root)).is_ok);if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    let bound=switch(binding::bind(ref sources,ref metadata,0,v::len<syntax::Parsed>(ref sources))){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    let ready=switch(closure::build(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    let moved=switch(moves::build(ref sources,ref typed,values::work(ref typed))){case .ok(p):{p}case .err(_):{assert(false);return retained(root,provider);}};
    return switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(_):{assert(false);retained(root,provider)}};
}
fn ir::Program nested(){return retained(b"module main;import provider as p;@effects(console)resource R{i32 n;cleanup{discard 1;}}resource Box{Result<p::R,i32> value;R other;cleanup{}}fn R make(){return R{.n=1};}@effects(console)fn void f(){let Box box=Box{.value=.ok(p::make()),.other=make()};}",b"module provider;@effects(console)pub resource R{i32 n;cleanup{discard 1;}}pub fn R make(){return R{.n=1};}");}
@test fn void cleanup_mir_owns_provider_identity_and_ordered_fields(){
    let p=nested();assert(proof::check(ref p).code==0 && ir::drop_count(ref p)==1 && ir::operand_count(ref p)==1);let d=ir::drop_at(ref p,0);let value=ir::operand_at(ref p,d.operand);assert(value.kind==1 && d.exit==1 && ir::frame_at(ref p,value.frame).module_index==0);
    let box=ir::glue_at(ref p,value.type);assert(box.kind==1 && !box.nonempty && box.frame==0 && box.field_count==2);let first=ir::field_at(ref p,box.field_start);let second=ir::field_at(ref p,box.field_start+1);assert(first.ordinal==0 && second.ordinal==1 && first.owner==value.type && second.owner==value.type);
    let tagged=ir::type_at(ref p,first.type);let root=ir::type_at(ref p,second.type);assert(tagged.kind==6 && root.kind==3 && root.module_index==0);let provider=ir::type_at(ref p,tagged.first);assert(provider.kind==3 && provider.module_index==1 && tagged.first!=second.type);
    let a=ir::glue_at(ref p,tagged.first);let b=ir::glue_at(ref p,second.type);assert(a.nonempty && b.nonempty && a.frame!=b.frame && ir::frame_at(ref p,a.frame).module_index==1 && ir::frame_at(ref p,b.frame).module_index==0);
}
@test fn void cleanup_mir_owns_distinct_partial_operand_values(){
    let p=retained(b"module main;resource R{i32 n;cleanup{}}resource Pair{R a;R b;i32 n;cleanup{}}fn R make(){return R{.n=1};}fn Result<i32,i32> number(){return .err(1);}fn Result<Pair,i32> f(){return .ok(Pair{.a=make(),.b=make(),.n=number()?});}",b"");
    assert(proof::check(ref p).code==0 && ir::drop_count(ref p)==2 && ir::operand_count(ref p)==2);let a=ir::operand_at(ref p,ir::drop_at(ref p,0).operand);let b=ir::operand_at(ref p,ir::drop_at(ref p,1).operand);assert(a.kind==2 && b.kind==2 && a.frame==b.frame && a.id!=b.id && a.type==b.type && a.start!=b.start);assert(ir::drop_at(ref p,0).exit==6 && ir::drop_at(ref p,1).exit==6);
}
@test fn void cleanup_receiver_exemption_does_not_exempt_same_type_locals(){
    let p=retained(b"module main;@effects(console)resource R{i32 n;cleanup{let R local=make();}}fn R make(){return R{.n=1};}",b"");assert(proof::check(ref p).code==0 && ir::drop_count(ref p)==1);let o=ir::operand_at(ref p,ir::drop_at(ref p,0).operand);let f=ir::frame_at(ref p,o.frame);assert(f.kind==2 && o.kind==1 && o.type==f.receiver_type && o.id!=f.receiver);
}

fn u32 mutated(ref ir::Program p,u32 mode){
    var v::Vec<types::Type> graph=v::empty<types::Type>();var v::Vec<ir::Frame> frames=v::empty<ir::Frame>();var v::Vec<ir::Glue> glues=v::empty<ir::Glue>();var v::Vec<ir::Field> fields=v::empty<ir::Field>();var v::Vec<ir::Operand> operands=v::empty<ir::Operand>();var v::Vec<ir::Drop> drops=v::empty<ir::Drop>();
    let drop=ir::drop_at(p,0);let operand=ir::operand_at(p,drop.operand);let root_glue=ir::glue_at(p,operand.type);let child=ir::type_at(p,ir::field_at(p,root_glue.field_start).type).first;var usize i=1;
    var usize scalar=0;while(i<=ir::type_count(p)){var t=ir::type_at(p,i);var g=ir::glue_at(p,i);if(t.kind==1 && t.scalar==1){scalar=i;}if(mode==4 && i==child){g.frame=0;}if(mode==7 && i==child){g.abilities=2;}if(mode==10 && t.kind==6){t.second=ir::type_count(p)+1;}if((mode==13 || mode==17) && t.kind==1 && t.scalar==3){t.scalar=1;}assert(v::push<types::Type>(ref mut graph,t).is_ok && v::push<ir::Glue>(ref mut glues,g).is_ok);i+=1;}
    i=1;while(i<=ir::frame_count(p)){var f=ir::frame_at(p,i);if(mode==3 && i==operand.frame){f.effects=0;}if(mode==5 && i==ir::glue_at(p,child).frame){f.module_index=0;}if(mode==12 && i==ir::glue_at(p,child).frame){f.result=scalar;}assert(v::push<ir::Frame>(ref mut frames,f).is_ok);i+=1;}
    i=0;while(i<ir::field_count(p)){var f=ir::field_at(p,i);if(mode==6 && f.owner==operand.type && f.ordinal==0){f.ordinal=1;}if(mode==9 && f.owner==operand.type && f.ordinal==0){f.type=operand.type;}assert(v::push<ir::Field>(ref mut fields,f).is_ok);i+=1;}
    i=1;while(i<=ir::operand_count(p)){var o=ir::operand_at(p,i);if(mode==2){o.type=ir::type_count(p)+1;}assert(v::push<ir::Operand>(ref mut operands,o).is_ok);i+=1;}if(mode==8){assert(v::push<ir::Operand>(ref mut operands,operand).is_ok);}
    i=0;while(i<ir::drop_count(p)){var d=ir::drop_at(p,i);if(mode==1){d.operand=ir::operand_count(p)+1;}assert(v::push<ir::Drop>(ref mut drops,d).is_ok);i+=1;}
    if(mode>=14){return switch(ir::assemble_project(graph,frames,glues,fields,operands,drops,if(mode==14){1048577}else if(mode==16){4194305}else{4194304})){case .ok(project):{assert(ir::project(ref project) && ir::work_limit(ref project)==4194304 && proof::check(ref project).code==0);0}case .err(e):{e.code}};}
    return switch(ir::assemble(graph,frames,glues,fields,operands,drops,if(mode==11){1048577}else{ir::work(p)})){case .ok(unit):{assert(!ir::project(ref unit) && ir::work_limit(ref unit)==1048576);0}case .err(e):{e.code}};
}
@test fn void cleanup_mir_rejects_invalid_handoffs_and_missing_authority(){let p=nested();assert(mutated(ref p,0)==0);var u32 mode=1;while(mode<=10){assert(mutated(ref p,mode)==1001);mode+=1;}assert(mutated(ref p,11)==1002 && mutated(ref p,12)==1001 && mutated(ref p,13)==1001);assert(mutated(ref p,14)==0 && mutated(ref p,15)==0 && mutated(ref p,16)==1002 && mutated(ref p,17)==1001);}
