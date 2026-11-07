module main;
import module_unit_bind as binding;
import module_unit_call_closure as closure;
import std::bytes as bytes;
import std::io as io;
import sha256 as sha;
import scanner as scanner;

// The Go test merges this fixture into a temporary copy of the private lowerer.
fn interfaces::Shape shape(usize type,usize ordinal){return interfaces::Shape{.type,.module_index=0,.declaration=type,.field_start=0,.field_count=0,.abilities=1,.repr_c=false,.start=ordinal,.end=ordinal+1,.cleanup_nonempty=true,.cleanup_effects=0};}
fn ir::Frame frame(usize type,u32 kind){return ir::Frame{.kind,.context=1,.module_index=0,.declaration=type,.result=1,.effects=0,.receiver=1,.receiver_type=type};}
fn usize scan_shape(ref v::Vec<interfaces::Shape> shapes,usize type,ref mut usize work){var usize found=0;var usize j=0;while(j<v::len<interfaces::Shape>(shapes)){assert(m::charge(work,1));if(v::at<interfaces::Shape>(shapes,j).type==type){found=j+1;}j+=1;}return found;}
fn Result<usize,ir::Problem> scan_frame(ref v::Vec<ir::Frame> frames,usize type,usize module_index,usize start,usize end,ref mut usize work){var usize found=0;var usize j=0;while(j<v::len<ir::Frame>(frames)){if(!m::charge(work,1)){return .err(fail(1002,module_index,start,end));}let f=*v::at<ir::Frame>(frames,j);if(f.kind==2 && f.receiver_type==type){if(found!=0){return .err(fail(1001,module_index,start,end));}found=j+1;}j+=1;}return if(found==0){.err(fail(1001,module_index,start,end))}else{.ok(found)};}
fn usize selected(Result<usize,ir::Problem> value){return switch(value){case .ok(n):{n}case .err(_):{assert(false);0}};}
fn ir::Problem problem(Result<usize,ir::Problem> value){return switch(value){case .ok(_):{assert(false);fail(0,0,0,0)}case .err(p):{p}};}
fn void same_problem(ir::Problem a,ir::Problem b){assert(a.code==b.code && a.module_index==b.module_index && a.start==b.start && a.end==b.end);}
@effects(console) fn void report(ref u8[] label,usize n,usize prior,usize baseline,usize changed){discard io::write_bytes(label);discard io::write_usize(n);discard io::write(" prior=");discard io::write_usize(prior);discard io::write(" baseline=");discard io::write_usize(baseline);discard io::write(" changed=");discard io::write_usize(changed);discard io::write(" saved=");discard io::write_usize(baseline-changed);discard io::write("\n");}
@effects(console) fn void indexed(usize count){
 var v::Vec<interfaces::Shape> shapes=v::empty<interfaces::Shape>();var v::Vec<ir::Frame> frames=v::empty<ir::Frame>();var usize i=1;
 while(i<=count){assert(v::push<interfaces::Shape>(ref mut shapes,shape(i,i)).is_ok && v::push<ir::Frame>(ref mut frames,frame(i,2)).is_ok && v::push<ir::Frame>(ref mut frames,frame(i,3)).is_ok);i+=1;}
 assert(v::push<interfaces::Shape>(ref mut shapes,shape(1,count+1)).is_ok);var usize changed=0;var usize baseline=0;
 let index=switch(index_recipes(count,ref shapes,ref frames,7,ref mut changed)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 i=1;while(i<=count){assert(m::charge(ref mut changed,1));let row=*v::at<Selection>(ref index.rows,i);assert(row.shape==scan_shape(ref shapes,i,ref mut baseline));assert(selected(recipe_frame(ref index,i,v::len<ir::Frame>(ref frames),7,31,41,ref mut changed))==selected(scan_frame(ref frames,i,7,31,41,ref mut baseline)));assert(row.shape==if(i==1){count+1}else{i});i+=1;}
 assert(baseline==count*(3*count+1) && changed==6*count+2 && changed<baseline);report(b"lookup resources=",count,0,baseline,changed);
 // Index construction ignores unselectable IDs and ordinary frames, as the scan
 // did. Repeated cleanup matches defer their error to the requesting recipe.
 assert(v::push<interfaces::Shape>(ref mut shapes,shape(0,51)).is_ok && v::push<interfaces::Shape>(ref mut shapes,shape(count+1,52)).is_ok);
 assert(v::push<ir::Frame>(ref mut frames,frame(1,2)).is_ok && v::push<ir::Frame>(ref mut frames,frame(1,2)).is_ok && v::push<ir::Frame>(ref mut frames,frame(count+1,2)).is_ok);
 var usize work=0;let duplicate=switch(index_recipes(count+1,ref shapes,ref frames,7,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 var usize scanned=0;same_problem(problem(recipe_frame(ref duplicate,1,v::len<ir::Frame>(ref frames),7,31,41,ref mut work)),problem(scan_frame(ref frames,1,7,31,41,ref mut scanned)));
 // Missing cleanup for a selected shape is still the same span-qualified 1001.
 var v::Vec<ir::Frame> empty=v::empty<ir::Frame>();let missing=switch(index_recipes(count+1,ref shapes,ref empty,7,ref mut work)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 same_problem(problem(recipe_frame(ref missing,count+1,0,9,52,53,ref mut work)),problem(scan_frame(ref empty,count+1,9,52,53,ref mut scanned)));
 assert(v::len<interfaces::Shape>(ref shapes)==count+3 && v::len<ir::Frame>(ref frames)==2*count+3 && v::at<interfaces::Shape>(ref shapes,count).start==count+1 && v::at<ir::Frame>(ref frames,0).receiver_type==1);
 work=1048576;let exhausted=problem(recipe_frame(ref index,1,2*count,7,31,41,ref mut work));assert(exhausted.code==1002 && exhausted.module_index==7 && exhausted.start==31 && exhausted.end==41 && work==1048576);
 switch(index_recipes(4097,ref shapes,ref frames,7,ref mut work)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1002);}};
 switch(index_recipes(count,ref shapes,ref frames,7,ref mut work)){case .ok(_):{assert(false);}case .err(p):{assert(p.code==1002 && work==1048576);}};
}
fn syntax::Parsed parsed(bytes::Buffer text){let tokens=switch(scanner::collect(text,false,65536,1048576,256,16777216)){case .ok(p):{p}case .err(_):{assert(false);return parsed(bytes::empty());}};return switch(syntax::parse(tokens,512)){case .ok(p):{assert(syntax::problem_count(ref p)==0);p}case .err(_):{assert(false);parsed(bytes::empty())}};}
fn void append(ref mut bytes::Buffer text,ref u8[] raw){assert(bytes::append(text,raw).is_ok);}
fn void number(ref mut bytes::Buffer text,usize n){assert(bytes::append_usize(text,n).is_ok);}
fn bool same_type(types::Type a,types::Type b){return a.kind==b.kind && a.scalar==b.scalar && a.module_index==b.module_index && a.declaration==b.declaration && a.first==b.first && a.second==b.second;}
fn void same_ir(ref ir::Program a,ref ir::Program b){
 assert(ir::type_count(a)==ir::type_count(b) && ir::frame_count(a)==ir::frame_count(b) && ir::glue_count(a)==ir::glue_count(b) && ir::field_count(a)==ir::field_count(b) && ir::operand_count(a)==ir::operand_count(b) && ir::drop_count(a)==ir::drop_count(b) && ir::aggregates(a)==ir::aggregates(b) && ir::project(a)==ir::project(b));var usize i=1;
 while(i<=ir::type_count(a)){assert(same_type(ir::type_at(a,i),ir::type_at(b,i)));let x=ir::glue_at(a,i);let y=ir::glue_at(b,i);assert(x.kind==y.kind && x.frame==y.frame && x.nonempty==y.nonempty && x.effects==y.effects && x.abilities==y.abilities && x.field_start==y.field_start && x.field_count==y.field_count);i+=1;}
 i=1;while(i<=ir::frame_count(a)){let x=ir::frame_at(a,i);let y=ir::frame_at(b,i);assert(x.kind==y.kind && x.context==y.context && x.module_index==y.module_index && x.declaration==y.declaration && x.result==y.result && x.effects==y.effects && x.receiver==y.receiver && x.receiver_type==y.receiver_type);i+=1;}
 i=0;while(i<ir::field_count(a)){let x=ir::field_at(a,i);let y=ir::field_at(b,i);assert(x.owner==y.owner && x.ordinal==y.ordinal && x.type==y.type);i+=1;}
 i=1;while(i<=ir::operand_count(a)){let x=ir::operand_at(a,i);let y=ir::operand_at(b,i);assert(x.frame==y.frame && x.kind==y.kind && x.id==y.id && x.type==y.type && x.start==y.start && x.end==y.end);i+=1;}
 i=0;while(i<ir::drop_count(a)){let x=ir::drop_at(a,i);let y=ir::drop_at(b,i);assert(x.operand==y.operand && x.exit==y.exit && x.start==y.start && x.end==y.end);i+=1;}
}
@effects(console) fn void real_unit(usize count){
 var bytes::Buffer text=bytes::empty();append(ref mut text,b"module root;");var usize i=0;
 while(i<count){append(ref mut text,b"resource R");number(ref mut text,i);append(ref mut text,b"{i32 n;cleanup{discard 1;}}fn void use");number(ref mut text,i);append(ref mut text,b"(R");number(ref mut text,i);append(ref mut text,b" value){}");i+=1;}
 var v::Vec<syntax::Parsed> sources=v::empty<syntax::Parsed>();assert(v::push<syntax::Parsed>(ref mut sources,parsed(text)).is_ok);let before=sha::digest(syntax::source_view(v::at<syntax::Parsed>(ref sources,0)));
 let metadata=switch(signatures::build(ref sources)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let checked=switch(checking::build(ref sources,ref metadata,0,1)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let typed=checking::checked_values(ref checked);let imported=checking::checked_interfaces(ref checked);let moved=checking::checked_moves(ref checked);let prior=policy::work(checking::cleanup_policy(ref checked));
 let a=switch(lower_policy_scan(ref sources,typed,imported,moved,prior)){case .ok(p):{p}case .err(_):{assert(false);return;}};
 let b=switch(lower_policy(ref sources,typed,imported,moved,prior)){case .ok(p):{p}case .err(_):{assert(false);return;}};same_ir(ref a,ref b);
 let expected_scans=count*interfaces::shape_count(imported)+count*ir::frame_count(ref a);let expected_index=values::type_count(typed)+1+interfaces::shape_count(imported)+ir::frame_count(ref a)+2*count;
 assert(ir::work(ref a)>ir::work(ref b) && ir::work(ref a)-ir::work(ref b)==expected_scans-expected_index);report(b"unit resources=",count,prior,ir::work(ref a),ir::work(ref b));
 let after=sha::digest(syntax::source_view(v::at<syntax::Parsed>(ref sources,0)));i=0;while(i<8){assert(before[i]==after[i]);i+=1;}
 // Same checked input is reused; neither lowering may consume/mutate it.
 let c=switch(lower_policy(ref sources,typed,imported,moved,prior)){case .ok(p):{p}case .err(_):{assert(false);return;}};same_ir(ref b,ref c);assert(ir::work(ref b)==ir::work(ref c));
}
@effects(console) fn i32 main(){indexed(32);indexed(256);indexed(512);real_unit(16);real_unit(64);discard io::write("selection=last frame=unique errors=ordered inputs=unchanged\n");return 0;}
