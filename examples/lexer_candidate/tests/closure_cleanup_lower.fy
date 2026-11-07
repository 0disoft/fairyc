module tests::closure_cleanup_lower;
import module_unit_cleanup_mir_model as ir;
import module_unit_cleanup_mir_lower as lower;
import module_unit_cleanup_mir_verify as proof;
import module_unit_cleanup_policy as policy;
import module_unit_checked_bodies as checking;
import module_unit_values as values;
import module_unit_value_model as model;
import module_unit_moves as moves;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import module_signatures as signatures;
import project_source_plan as planning;
import generic_aggregate_types as aggregate_types;
import module_catalog as catalog;
import module_public_type_text as coordinates;
import sha256_stream as sha;
import syntax as syntax;
import scanner as scanner;
import std::bytes as bytes;
import std::vec as v;
import std::io as io;

// Go passes original text only. This producer checks cleanup MIR facts, not
// execution of the source closure body, bodyIR, C, artifacts or cache restore.
pub struct Receipt {u32 phase;u32 code;usize start;usize end;usize leaves;usize frames;usize drops;usize work;bool closures;bool aggregates;}
fn Receipt failed(u32 phase,u32 code,usize start,usize end){return Receipt{.phase,.code,.start,.end,.leaves=0,.frames=0,.drops=0,.work=0,.closures=false,.aggregates=false};}
fn u32 positive(i32 value){if(value<0 || value>65536){return 1;}var i32 left=value;var u32 code=0;while(left!=0){left-=1;code+=1;}return code;}
fn syntax::Parsed parsed(ref u8[] text){
    let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(_):{assert(false);return parsed(b"module empty;");}};
    let p=switch(syntax::parse(stream,512)){case .ok(p):{p}case .err(_):{assert(false);return parsed(b"module empty;");}};assert(syntax::problem_count(ref p)==0);return p;
}
// The three written Box<T> controls need the same authenticated aggregate
// prerequisite as project_compiler_core. Select it from original declarations,
// never a case name, Go type graph or expected admission. The optional Vec SDK
// provider is not a user aggregate declaration in the root source.
fn bool aggregate_source(ref v::Vec<syntax::Parsed> sources){
    let root=v::at<syntax::Parsed>(sources,0);var usize i=1;
    while(i<syntax::declaration_count(root)){let d=syntax::declaration_at(root,i);if((d.kind==4 || d.kind==6) && d.generic_parameters!=0){return true;}i+=1;}return false;
}
fn Result<signatures::Program,Receipt> prepared_signatures(ref v::Vec<syntax::Parsed> sources){
    if(!aggregate_source(sources)){return switch(signatures::build(sources)){case .ok(p):{.ok(p)}case .err(e):{.err(failed(4,e.code,e.start,e.end))}};}
    // These unchanged aggregate inputs each own one module and no imports.
    // Refuse unsupported fixture plumbing rather than silently inventing it.
    if(v::len<syntax::Parsed>(sources)!=1){return .err(failed(4,9,0,0));}
    var owners=v::empty<usize>();if(!v::push<usize>(ref mut owners,0).is_ok){return .err(failed(4,11,0,0));}let imports=v::empty<catalog::Binding>();
    var names=bytes::empty();var digest=sha::empty();let root=v::at<syntax::Parsed>(sources,0);
    if(!sha::update(ref mut digest,syntax::source_view(root)) || !bytes::append(ref mut names,b"fypc1-sha256-").is_ok || !sha::append_hex(ref digest,ref mut names)){return .err(failed(4,11,0,0));}
    var rows=v::empty<coordinates::Coordinate>();if(!v::push<coordinates::Coordinate>(ref mut rows,coordinates::Coordinate{.owner=0,.start=0,.end=bytes::len(ref names)}).is_ok){return .err(failed(4,11,0,0));}let table=coordinates::assemble(names,rows);
    var usize source_work=0;
    let original=switch(planning::build_aggregate_sources(sources,ref owners,ref imports,ref table,ref mut source_work)){case .ok(p):{p}case .err(e):{return .err(failed(4,e.code,e.start,e.end));}};
    let prepared=switch(aggregate_types::build(original,sources,ref mut source_work)){case .ok(p):{p}case .err(e):{return .err(failed(4,e.code,e.start,e.end));}};
    assert(aggregate_types::count(ref prepared)==1);
    return switch(signatures::build_aggregates(sources,prepared,ref mut source_work)){case .ok(p):{assert(signatures::aggregate_types(ref p) && signatures::source_matches(ref p,0,root));.ok(p)}case .err(e):{.err(failed(4,e.code,e.start,e.end))}};
}
fn void same(ref ir::Program a,ref ir::Program b){
    assert(proof::check(b).code==0 && ir::closures(a)==ir::closures(b) && ir::aggregates(a)==ir::aggregates(b) && !ir::project(b));
    assert(ir::type_count(a)==ir::type_count(b) && ir::frame_count(a)==ir::frame_count(b) && ir::field_count(a)==ir::field_count(b) && ir::operand_count(a)==ir::operand_count(b) && ir::drop_count(a)==ir::drop_count(b) && ir::work(a)==ir::work(b));var usize i=1;
    while(i<=ir::type_count(a)){let x=ir::type_at(a,i);let y=ir::type_at(b,i);assert(x.kind==y.kind && x.scalar==y.scalar && x.module_index==y.module_index && x.declaration==y.declaration && x.first==y.first && x.second==y.second);let g=ir::glue_at(a,i);let h=ir::glue_at(b,i);assert(g.kind==h.kind && g.frame==h.frame && g.nonempty==h.nonempty && g.effects==h.effects && g.abilities==h.abilities && g.field_start==h.field_start && g.field_count==h.field_count);i+=1;}
    i=1;while(i<=ir::frame_count(a)){let x=ir::frame_at(a,i);let y=ir::frame_at(b,i);assert(x.kind==y.kind && x.context==y.context && x.module_index==y.module_index && x.declaration==y.declaration && x.result==y.result && x.effects==y.effects && x.receiver==y.receiver && x.receiver_type==y.receiver_type);i+=1;}
    i=0;while(i<ir::field_count(a)){let x=ir::field_at(a,i);let y=ir::field_at(b,i);assert(x.owner==y.owner && x.ordinal==y.ordinal && x.type==y.type);i+=1;}
    i=1;while(i<=ir::operand_count(a)){let x=ir::operand_at(a,i);let y=ir::operand_at(b,i);assert(x.frame==y.frame && x.kind==y.kind && x.id==y.id && x.type==y.type && x.start==y.start && x.end==y.end);i+=1;}
    i=0;while(i<ir::drop_count(a)){let x=ir::drop_at(a,i);let y=ir::drop_at(b,i);assert(x.operand==y.operand && x.exit==y.exit && x.start==y.start && x.end==y.end);i+=1;}
}
fn Receipt inspect(ref ir::Program p,ref values::Plan typed,ref moves::Plan moved,usize prior_work){
    assert(proof::check(p).code==0 && !ir::project(p) && ir::type_count(p)==values::type_count(typed) && ir::frame_count(p)==values::count(typed) && ir::drop_count(p)==moves::cleanup_count(moved));
    assert(ir::work(p)>=prior_work && ir::work(p)<=ir::work_limit(p));var usize leaves=0;var usize i=1;
    while(i<=ir::type_count(p)){let t=ir::type_at(p,i);let expected=values::type_at(typed,i);assert(t.kind==expected.kind && t.scalar==expected.scalar && t.module_index==expected.module_index && t.declaration==expected.declaration && t.first==expected.first && t.second==expected.second);
        if(t.kind==23){leaves+=1;assert(t.scalar==0 && t.module_index==0 && t.declaration==0 && t.first==0 && t.second==0);let g=ir::glue_at(p,i);assert(g.kind==0 && g.frame==0 && !g.nonempty && g.effects==0 && g.abilities==0 && g.field_start==0 && g.field_count==0);}i+=1;
    }
    assert(leaves<=1 && ir::closures(p)==(leaves==1));i=1;
    while(i<=ir::frame_count(p)){let f=ir::frame_at(p,i);let original=model::frame(values::at(typed,i-1));assert(f.kind==original.kind && f.context==original.context && f.module_index==original.module_index && f.declaration==original.declaration && f.result==original.result);i+=1;}
    i=0;while(i<ir::drop_count(p)){let d=ir::drop_at(p,i);let o=ir::operand_at(p,d.operand);let request=moves::cleanup_at(moved,i);assert(request.kind==5 && o.frame==request.frame && o.type==request.type && o.kind==if(request.binding!=0){1}else{2} && o.id==if(request.binding!=0){request.binding}else{request.expression});assert(d.exit==request.exit && d.start==request.start && d.end==request.end && ir::type_at(p,o.type).kind!=23);i+=1;}
    return Receipt{.phase=0,.code=0,.start=0,.end=0,.leaves,.frames=ir::frame_count(p),.drops=ir::drop_count(p),.work=ir::work(p),.closures=ir::closures(p),.aggregates=ir::aggregates(p)};
}
pub fn Receipt checked(ref u8[] text,ref u8[] provider){
    let input=switch(bytes::copy(text)){case .ok(b):{b}case .err(_):{return failed(1,11,0,text.len);}};
    let stream=switch(scanner::collect(input,false,65536,1048576,256,16777216)){case .ok(s):{s}case .err(code):{return failed(2,positive(code),0,text.len);}};
    let source=switch(syntax::parse(stream,512)){case .ok(s):{s}case .err(code):{return failed(3,positive(code),0,text.len);}};
    if(syntax::problem_count(ref source)!=0){let e=syntax::problem_at(ref source,0);return failed(3,e.code,e.start,e.end);}
    var sources=v::empty<syntax::Parsed>();if(!v::push<syntax::Parsed>(ref mut sources,source).is_ok){return failed(1,11,0,text.len);}
    if(provider.len!=0){assert(v::push<syntax::Parsed>(ref mut sources,parsed(provider)).is_ok);}
    let metadata=switch(prepared_signatures(ref sources)){case .ok(p):{p}case .err(e):{return e;}};
    assert(signatures::aggregate_types(ref metadata)==aggregate_source(ref sources));
    let bound=switch(binding::bind(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{return failed(5,e.code,e.start,e.end);}};
    let ready=switch(closure::build_unit(ref sources,ref metadata,ref bound)){case .ok(p):{p}case .err(e):{return failed(5,e.code,e.start,e.end);}};
    let typed=switch(values::build(ref sources,ref metadata,ref ready)){case .ok(p):{p}case .err(e):{return failed(6,e.code,e.start,e.end);}};
    let moved=switch(moves::build(ref sources,ref typed,0)){case .ok(p):{p}case .err(e):{return failed(7,e.code,e.start,e.end);}};
    if(moves::count(ref moved)!=0){let e=moves::at(ref moved,0);let refusal=lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved);assert(switch(refusal){case .ok(_):{false}case .err(p):{p.code==e.code && p.module_index==e.module_index && p.start==e.start && p.end==e.end}});return failed(7,e.code,e.start,e.end);}
    let authorized=switch(policy::build(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{return failed(8,e.code,e.start,e.end);}};
    if(policy::count(ref authorized)!=0){
        let e=policy::at(ref authorized,0);
        // Raw lower must rebuild/check policy, even when kind23 is present.
        assert(switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(_):{false}case .err(p):{p.code==e.code && p.module_index==e.module_index && p.start==e.start && p.end==e.end}});
        assert(switch(checking::build(ref sources,ref metadata,0,1)){case .ok(_):{false}case .err(p):{p.stage==6 && p.code==e.code && p.module_index==e.module_index && p.start==e.start && p.end==e.end}});
        return failed(8,e.code,e.start,e.end);
    }
    let p=switch(lower::lower(ref sources,ref typed,closure::imported(ref ready),ref moved)){case .ok(p):{p}case .err(e):{return failed(9,e.code,e.start,e.end);}};
    let result=inspect(ref p,ref typed,ref moved,policy::work(ref authorized));
    assert(result.aggregates==signatures::aggregate_types(ref metadata));
    let owner=switch(checking::build(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(e):{return failed(10,e.code,e.start,e.end);}};
    let sourcechecked=switch(lower::lower_checked(ref sources,ref owner)){case .ok(p):{p}case .err(e):{return failed(11,e.code,e.start,e.end);}};
    let catalogchecked=switch(lower::lower_checked_catalog(ref sources,ref owner,ref metadata)){case .ok(p):{p}case .err(e):{return failed(12,e.code,e.start,e.end);}};
    assert(policy::count(checking::cleanup_policy(ref owner))==0);same(ref p,ref sourcechecked);same(ref p,ref catalogchecked);return result;
}
@effects(console) pub fn void verify(ref u8[] name,ref u8[] text,ref u8[] provider,bool accept,bool closures,bool aggregates,usize drops,u32 policy_code){
    let r=checked(text,provider);let accepted=r.phase==0 && r.code==0;
    // Allocation, budget and stale-record errors cannot count as refusals.
    let valid=if(accept){accepted && r.closures==closures && r.aggregates==aggregates && r.drops==drops && r.frames!=0 && r.work!=0}else{!accepted && r.phase>=2 && r.phase<=9 && r.code!=0 && r.code!=10 && r.code!=11 && r.code!=12 && r.code!=1002 && r.code!=1003 && r.start<=r.end && r.end<=text.len && (policy_code==0 || r.phase==8 && r.code==policy_code)};
    if(!valid){discard io::write_stderr("closure-cleanup-lower case=");discard io::write_stderr_bytes(name);discard io::write_stderr(" phase=");discard io::write_stderr_u32(r.phase);discard io::write_stderr(" code=");discard io::write_stderr_u32(r.code);discard io::write_stderr(" source=");discard io::write_stderr_bytes(text);discard io::write_stderr("\n");assert(false);return;}
    discard io::write("closure-cleanup-lower case=");discard io::write_bytes(name);discard io::write(" phase=");discard io::write_u32(r.phase);discard io::write(" code=");discard io::write_u32(r.code);discard io::write(" leaves=");discard io::write_usize(r.leaves);discard io::write(" drops=");discard io::write_usize(r.drops);discard io::write(" work=");discard io::write_usize(r.work);discard io::write("\n");
}
@test fn void closure_cleanup_lower_checked_sources_stay_fresh(){
    let text=b"module main;fn void use(fn i32(i32) action){discard action(1);}";
    let changed=b"module main;fn void use(fn i32(i32) action){discard action(2);}";assert(text.len==changed.len);
    var sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);
    let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    let owner=switch(checking::build(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(lower::lower_checked(ref sources,ref owner).is_ok && lower::lower_checked_catalog(ref sources,ref owner,ref metadata).is_ok);
    var other=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut other,parsed(changed)).is_ok);
    let refreshed=switch(signatures::build(ref other)){case .ok(p):{p}case .err(_):{assert(false);return;}};
    assert(switch(lower::lower_checked(ref other,ref owner)){case .ok(_):{false}case .err(e):{e.code==12}});
    assert(switch(lower::lower_checked_catalog(ref other,ref owner,ref refreshed)){case .ok(_):{false}case .err(e):{e.code==12}});
    assert(switch(lower::lower_checked_catalog(ref sources,ref owner,ref refreshed)){case .ok(_):{false}case .err(e):{e.code==12}});
    let empty=v::empty<syntax::Parsed>();assert(switch(lower::lower_checked(ref empty,ref owner)){case .ok(_):{false}case .err(e):{e.code==12}});
}
