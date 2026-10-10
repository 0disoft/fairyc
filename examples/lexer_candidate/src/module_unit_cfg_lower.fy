module module_unit_cfg_lower;
import module_unit_cfg_model as cfg;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_value_model as m;
import module_unit_cfg_defer as deferred;
import module_unit_cfg_region as lifecycle;
import std::bytes as bytes;
import std::vec as v;

struct Context {usize frame;usize module_index;usize work;usize active_count;usize[256] active;}
struct Loop {usize break_to;usize continue_to;usize defer_base;}
fn clean_ir::Problem fail(ref Context c,u32 code,usize start,usize end){return clean_ir::Problem{.code,.module_index=c.module_index,.start,.end};}
fn Result<usize,clean_ir::Problem> emit(ref mut v::Vec<cfg::Block> blocks,ref mut v::Vec<cfg::Deferred> sites,ref mut v::Vec<cfg::Capture> captures,u32 kind,usize source,usize target,usize first,usize second,usize drop,usize start,usize end,ref mut Context c){
    if(v::len<cfg::Block>(blocks)>=32768 || !m::charge(ref mut c.work,1)){return .err(fail(c,1002,start,end));}
    return switch(v::push<cfg::Block>(blocks,cfg::Block{.kind,.source,.target,.first,.second,.drop,.start,.end})){case .ok(n):{.ok(n)}case .err(_):{.err(fail(c,1003,start,end))}};
}
fn Result<usize,clean_ir::Problem> drops(ref ir::Program p,u32 reason,usize start,usize end,usize next,ref mut v::Vec<cfg::Block> blocks,ref mut v::Vec<cfg::Deferred> sites,ref mut v::Vec<cfg::Capture> captures,ref mut Context c){
    return lifecycle::exit(p,c.frame,reason,start,end,0,0,0,next,blocks,ref mut c.work);
}
fn Result<usize,clean_ir::Problem> invoke(ref ir::Program p,usize from,usize next,ref mut v::Vec<cfg::Block> blocks,ref mut v::Vec<cfg::Deferred> sites,ref mut v::Vec<cfg::Capture> captures,ref mut Context c){
    if(from>c.active_count){return .err(fail(c,1001,0,0));}var usize entry=next;var usize i=from;while(i<c.active_count){let id=c.active[i];let d=*v::at<cfg::Deferred>(sites,id-1);let b=ir::at(p,c.frame);let call=ir::call_at(b,d.invocation-1);entry=switch(emit(blocks,sites,captures,14,call.expression,id,entry,0,0,call.start,call.end,c)){case .ok(n):{n}case .err(e):{return .err(e);}};i+=1;}return .ok(entry);
}
fn Result<usize,clean_ir::Problem> expression(ref ir::Program p,usize original,usize next,Loop loop,usize depth,ref mut v::Vec<cfg::Block> blocks,ref mut v::Vec<cfg::Deferred> sites,ref mut v::Vec<cfg::Capture> captures,ref mut Context c){
    if(original==0){return .ok(next);}if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(fail(c,1002,0,0));}let b=ir::at(p,c.frame);let id=lookup::node(b,original);if(id==0){return .err(fail(c,1001,0,0));}let e=ir::node_at(b,id-1).value;
    if(e.kind==15 || e.kind==32 || e.kind==33){return .err(fail(c,1004,e.start,e.end));}
    if(e.state==2 && e.kind==8){return expression(p,e.left,next,loop,depth+1,blocks,sites,captures,c);}
    if(e.state==2 && (e.kind==2 || e.kind==17 || e.kind==20 || e.kind==16 || e.kind==28)){return .ok(next);}
    if(e.kind==12 || e.kind==31){let root=ir::statement_at(b,lookup::statement(b,e.left)-1).value;let copied=switch(emit(blocks,sites,captures,11,id,lookup::node(b,root.expression),next,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return block(p,e.left,copied,loop,depth+1,blocks,sites,captures,c);}
    if(e.kind==13){let pair=ir::node_at(b,lookup::node(b,e.right)-1).value;let yes_root=ir::statement_at(b,lookup::statement(b,pair.left)-1).value;
        let yes_copy=switch(emit(blocks,sites,captures,11,id,lookup::node(b,yes_root.expression),next,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let yes=switch(block(p,pair.left,yes_copy,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let no_copy=switch(emit(blocks,sites,captures,11,id,lookup::node(b,pair.right),next,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let no=switch(expression(p,pair.right,no_copy,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let branch=switch(emit(blocks,sites,captures,3,lookup::node(b,e.left),0,yes,no,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return expression(p,e.left,branch,loop,depth+1,blocks,sites,captures,c);
    }
    var usize after=switch(drops(p,7,e.start,e.end,next,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    let evaluated=switch(emit(blocks,sites,captures,1,id,0,after,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    if(e.kind==26){let bound_id=lookup::node(b,e.right);if(bound_id==0){return .err(fail(c,1001,e.start,e.end));}let pair=ir::node_at(b,bound_id-1).value;if(pair.kind!=27){return .err(fail(c,1001,e.start,e.end));}after=switch(expression(p,pair.right,evaluated,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};after=switch(expression(p,pair.left,after,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return expression(p,e.left,after,loop,depth+1,blocks,sites,captures,c);}
    if(e.kind==30){let returned=switch(emit(blocks,sites,captures,9,lookup::node(b,e.left),id,0,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let dropped=switch(lifecycle::exit(p,c.frame,6,e.start,e.end,3,e.start,e.end,returned,blocks,ref mut c.work)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let error=switch(invoke(p,0,dropped,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};let branch=switch(emit(blocks,sites,captures,4,lookup::node(b,e.left),id,evaluated,error,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return expression(p,e.left,branch,loop,depth+1,blocks,sites,captures,c);}
    let op=ir::node_text(b,id-1);if(e.kind==5 && (bytes::equal(op,b"&&") || bytes::equal(op,b"||"))){
        let right=switch(expression(p,e.right,evaluated,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let skipped=switch(emit(blocks,sites,captures,10,id,lookup::node(b,e.left),next,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let branch=switch(emit(blocks,sites,captures,3,lookup::node(b,e.left),0,if(bytes::equal(op,b"&&")){right}else{skipped},if(bytes::equal(op,b"&&")){skipped}else{right},0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return expression(p,e.left,branch,loop,depth+1,blocks,sites,captures,c);
    }
    after=evaluated;
    if(e.kind==6 || e.kind==10 || e.kind==24){var usize link=e.right;while(link!=0){if(!m::charge(ref mut c.work,1)){return .err(fail(c,1002,e.start,e.end));}let pair=ir::node_at(b,lookup::node(b,link)-1).value;after=switch(expression(p,pair.left,after,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};link=pair.right;}let receiver=if(e.builtin.kind==300){ir::node_at(b,lookup::node(b,e.left)-1).value.left}else{e.left};return expression(p,receiver,after,loop,depth+1,blocks,sites,captures,c);}
    if(e.kind!=20 && e.kind!=23){after=switch(expression(p,e.right,after,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}return expression(p,e.left,after,loop,depth+1,blocks,sites,captures,c);
}
fn Result<usize,clean_ir::Problem> statement(ref ir::Program p,usize original,usize next,Loop loop,usize depth,ref mut v::Vec<cfg::Block> blocks,ref mut v::Vec<cfg::Deferred> sites,ref mut v::Vec<cfg::Capture> captures,ref mut Context c){
    if(original==0){return .ok(next);}if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(fail(c,1002,0,0));}let b=ir::at(p,c.frame);let id=lookup::statement(b,original);if(id==0){return .err(fail(c,1001,0,0));}let s=ir::statement_at(b,id-1).value;
    if(s.kind==1){return block(p,original,next,loop,depth+1,blocks,sites,captures,c);}if(s.kind==18){return .err(fail(c,1004,s.start,s.end));}
    if(s.kind==17){let region_id=lifecycle::find_region(b,original);if(region_id==0){return .err(fail(c,1001,s.start,s.end));}let reset=switch(lifecycle::exit(p,c.frame,0,0,0,1,s.start,s.end,next,blocks,ref mut c.work)){case .ok(n):{n}case .err(p):{return .err(p);}};let inside=switch(block(p,s.first,reset,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};let mark=switch(emit(blocks,sites,captures,15,lookup::node(b,s.expression),region_id,inside,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return expression(p,s.expression,mark,loop,depth+1,blocks,sites,captures,c);}
    if(s.kind==9 || s.kind==10){let to=if(s.kind==9){loop.break_to}else{loop.continue_to};if(to==0){return .err(fail(c,1001,s.start,s.end));}let dropped=switch(lifecycle::exit(p,c.frame,if(s.kind==9){3}else{4},s.start,s.end,if(s.kind==9){4}else{5},s.start,s.end,to,blocks,ref mut c.work)){case .ok(n):{n}case .err(e):{return .err(e);}};return invoke(p,loop.defer_base,dropped,blocks,sites,captures,c);}
    if(s.kind==8 || s.kind==14){
        let anchor=switch(emit(blocks,sites,captures,7,0,0,next,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        var usize initial=0;var usize post=0;if(s.kind==14){let h=ir::statement_at(b,lookup::statement(b,s.second)-1).value;initial=h.first;post=h.second;}
        let after=switch(statement(p,post,anchor,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let inside=switch(block(p,s.first,after,Loop{.break_to=next,.continue_to=after,.defer_base=c.active_count},depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let branch=if(s.expression==0){inside}else{switch(emit(blocks,sites,captures,3,lookup::node(b,s.expression),0,inside,next,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}}};
        let condition=switch(expression(p,s.expression,branch,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};{let slot=v::at_mut<cfg::Block>(blocks,anchor-1);slot.first=condition;}
        return statement(p,initial,condition,loop,depth+1,blocks,sites,captures,c);
    }
    if(s.kind==13){let site=deferred::find(sites,id);if(site==0){return .err(fail(c,1001,s.start,s.end));}let d=*v::at<cfg::Deferred>(sites,site-1);var usize entry=next;var usize i=d.capture_count;while(i>0){i-=1;let capture=d.capture_start+i+1;let item=*v::at<cfg::Capture>(captures,capture-1);let e=ir::node_at(b,item.source-1).value;let frozen=switch(emit(blocks,sites,captures,13,item.source,capture,entry,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};entry=switch(expression(p,e.source_index,frozen,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}return .ok(entry);}
    var usize after=next;
    if(s.kind==3){after=switch(emit(blocks,sites,captures,6,lookup::node(b,s.expression),0,0,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};let result_source=lookup::node(b,s.expression);let reset_start=if(result_source==0){s.start}else{ir::node_at(b,result_source-1).value.start};let reset_end=if(result_source==0){s.end}else{ir::node_at(b,result_source-1).value.end};after=switch(lifecycle::exit(p,c.frame,2,s.start,s.end,2,reset_start,reset_end,after,blocks,ref mut c.work)){case .ok(n):{n}case .err(p):{return .err(p);}};after=switch(invoke(p,0,after,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}
    else if(s.kind==2 || s.kind==4){after=switch(emit(blocks,sites,captures,2,lookup::node(b,s.expression),id,next,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};if(s.kind==4){let target=ir::node_at(b,lookup::node(b,s.target)-1).value;after=switch(drops(p,5,target.start,target.end,after,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}}
    else if(s.kind==11){after=switch(emit(blocks,sites,captures,8,lookup::node(b,s.expression),0,next,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}
    else if(s.kind!=6 && s.kind!=12 && s.kind!=16){return .err(fail(c,1004,s.start,s.end));}
    if(s.kind==12 && s.expression!=0){let e=ir::node_at(b,lookup::node(b,s.expression)-1).value;after=switch(drops(p,8,e.start,e.end,after,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}
    var usize entry=switch(expression(p,s.expression,after,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    if(s.kind==4 && !bytes::equal(ir::assignment_operator(b,id-1),b"=")){entry=switch(emit(blocks,sites,captures,17,lookup::node(b,s.target),id,entry,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}
    // A root assignment selects a storage slot; it must not load a moved-out
    // owner before evaluating its replacement. Indexed/projected targets keep
    // their checked receiver/index evaluation before the RHS.
    if(s.kind==4 && ir::node_at(b,lookup::node(b,s.target)-1).value.kind!=2){return expression(p,s.target,entry,loop,depth+1,blocks,sites,captures,c);}return .ok(entry);
}
fn Result<usize,clean_ir::Problem> block(ref ir::Program p,usize original,usize next,Loop loop,usize depth,ref mut v::Vec<cfg::Block> blocks,ref mut v::Vec<cfg::Deferred> sites,ref mut v::Vec<cfg::Capture> captures,ref mut Context c){
    if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(fail(c,1002,0,0));}let b=ir::at(p,c.frame);let index=lookup::statement(b,original);if(index==0){return .err(fail(c,1001,0,0));}let root=ir::statement_at(b,index-1).value;if(root.kind!=1){return .err(fail(c,1001,root.start,root.end));}
    let base=c.active_count;var v::Vec<usize> order=v::empty<usize>();var v::Vec<usize> prefixes=v::empty<usize>();var usize link=root.first;
    while(link!=0){if(!m::charge(ref mut c.work,1)){return .err(fail(c,1002,root.start,root.end));}let pair=ir::statement_at(b,lookup::statement(b,link)-1).value;if(!v::push<usize>(ref mut order,pair.first).is_ok){return .err(fail(c,1003,root.start,root.end));}link=pair.second;}
    var usize i=v::len<usize>(ref order);while(i>0){i-=1;let stmt=lookup::statement(b,*v::at<usize>(ref order,i));let s=ir::statement_at(b,stmt-1).value;if(!v::push<usize>(ref mut prefixes,c.active_count).is_ok){return .err(fail(c,1003,s.start,s.end));}if(s.kind==13){if(c.active_count>=256){return .err(fail(c,1002,s.start,s.end));}let site=switch(deferred::prepare(p,c.frame,stmt,sites,captures,ref mut c.work)){case .ok(n):{n}case .err(e):{return .err(e);}};let slot=c.active_count;c.active[slot]=site;c.active_count=slot+1;}}
    var usize entry=switch(drops(p,1,root.start,root.end,next,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};entry=switch(invoke(p,base,entry,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    entry=switch(expression(p,root.expression,entry,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    i=0;while(i<v::len<usize>(ref order)){c.active_count=*v::at<usize>(ref prefixes,v::len<usize>(ref order)-i-1);entry=switch(statement(p,*v::at<usize>(ref order,i),entry,loop,depth+1,blocks,sites,captures,c)){case .ok(n):{n}case .err(p):{return .err(p);}};i+=1;}c.active_count=base;return .ok(entry);
}
pub fn Result<cfg::Graph,clean_ir::Problem> lower(ref ir::Program p,usize frame){
    let checked=proof::check(p);if(checked.code!=0){return .err(checked);}if(frame==0 || frame>ir::count(p)){return .err(clean_ir::Problem{.code=1001,.module_index=0,.start=0,.end=0});}
    let b=ir::at(p,frame);var Context c=Context{.frame,.module_index=clean_ir::frame_at(ir::cleanup_ir(p),frame).module_index,.work=0,.active_count=0,.active=[0;256]};
    var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();var v::Vec<cfg::Deferred> sites=v::empty<cfg::Deferred>();var v::Vec<cfg::Capture> captures=v::empty<cfg::Capture>();let result=clean_ir::type_at(ir::cleanup_ir(p),clean_ir::frame_at(ir::cleanup_ir(p),frame).result);let returned=switch(emit(ref mut blocks,ref mut sites,ref mut captures,if(result.kind==1 && result.scalar==7){6}else{12},0,0,0,0,0,ir::start(b),ir::end(b),ref mut c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    let entry=switch(block(p,ir::body(b),returned,Loop{.break_to=0,.continue_to=0,.defer_base=0},0,ref mut blocks,ref mut sites,ref mut captures,ref mut c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    return cfg::assemble_deferred(p,frame,entry,blocks,sites,captures,c.work);
}
