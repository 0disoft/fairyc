module module_unit_cfg_lower;
import module_unit_cfg_model as cfg;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_lookup as lookup;
import module_unit_body_ir_verify as proof;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_value_model as m;
import std::bytes as bytes;
import std::vec as v;

struct Context {usize frame;usize module_index;usize work;}
struct Loop {usize break_to;usize continue_to;}
fn clean_ir::Problem fail(ref Context c,u32 code,usize start,usize end){return clean_ir::Problem{.code,.module_index=c.module_index,.start,.end};}
fn Result<usize,clean_ir::Problem> emit(ref mut v::Vec<cfg::Block> blocks,u32 kind,usize source,usize target,usize first,usize second,usize drop,usize start,usize end,ref mut Context c){
    if(v::len<cfg::Block>(blocks)>=32768 || !m::charge(ref mut c.work,1)){return .err(fail(c,1002,start,end));}
    return switch(v::push<cfg::Block>(blocks,cfg::Block{.kind,.source,.target,.first,.second,.drop,.start,.end})){case .ok(n):{.ok(n)}case .err(_):{.err(fail(c,1003,start,end))}};
}
fn Result<usize,clean_ir::Problem> drops(ref ir::Program p,u32 reason,usize start,usize end,usize next,ref mut v::Vec<cfg::Block> blocks,ref mut Context c){
    let cleaned=ir::cleanup_ir(p);var v::Vec<usize> ordered=v::empty<usize>();var usize i=0;
    while(i<clean_ir::drop_count(cleaned)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,1002,start,end));}let d=clean_ir::drop_at(cleaned,i);let o=clean_ir::operand_at(cleaned,d.operand);
        if(o.frame==c.frame && d.exit==reason && d.start==start && d.end==end){var bool duplicate=false;var usize j=0;while(j<v::len<usize>(ref ordered)){if(!m::charge(ref mut c.work,1)){return .err(fail(c,1002,start,end));}if(clean_ir::drop_at(cleaned,*v::at<usize>(ref ordered,j)-1).operand==d.operand){duplicate=true;}j+=1;}if(!duplicate && !v::push<usize>(ref mut ordered,i+1).is_ok){return .err(fail(c,1003,start,end));}}i+=1;
    }
    // Construct a chain in ascending construction order: execution is reversed.
    // Earlier materialized operands are evaluated before later ones; temporaries
    // unwind before lexical owners when an expression exits abruptly.
    i=1;while(i<v::len<usize>(ref ordered)){var usize j=i;while(j>0){if(!m::charge(ref mut c.work,1)){return .err(fail(c,1002,start,end));}let a=clean_ir::operand_at(cleaned,clean_ir::drop_at(cleaned,*v::at<usize>(ref ordered,j-1)-1).operand);let b=clean_ir::operand_at(cleaned,clean_ir::drop_at(cleaned,*v::at<usize>(ref ordered,j)-1).operand);
        if(a.kind<b.kind || a.kind==b.kind && a.start<=b.start){break;}let old=*v::at<usize>(ref ordered,j-1);let here=*v::at<usize>(ref ordered,j);{let slot=v::at_mut<usize>(ref mut ordered,j-1);*slot=here;}{let slot=v::at_mut<usize>(ref mut ordered,j);*slot=old;}j-=1;}i+=1;}
    var usize entry=next;i=0;while(i<v::len<usize>(ref ordered)){entry=switch(emit(blocks,5,0,0,entry,0,*v::at<usize>(ref ordered,i),start,end,c)){case .ok(n):{n}case .err(e):{return .err(e);}};i+=1;}return .ok(entry);
}
fn Result<usize,clean_ir::Problem> expression(ref ir::Program p,usize original,usize next,Loop loop,usize depth,ref mut v::Vec<cfg::Block> blocks,ref mut Context c){
    if(original==0){return .ok(next);}if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(fail(c,1002,0,0));}let b=ir::at(p,c.frame);let id=lookup::node(b,original);if(id==0){return .err(fail(c,1001,0,0));}let e=ir::node_at(b,id-1).value;
    if(e.kind==15 || e.kind==26 || e.kind==32 || e.kind==33){return .err(fail(c,1004,e.start,e.end));}
    if(e.state==2 && (e.kind==2 || e.kind==17 || e.kind==20 || e.kind==16 || e.kind==28)){return .ok(next);}
    if(e.kind==12 || e.kind==31){let root=ir::statement_at(b,lookup::statement(b,e.left)-1).value;let copied=switch(emit(blocks,11,id,lookup::node(b,root.expression),next,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return block(p,e.left,copied,loop,depth+1,blocks,c);}
    if(e.kind==13){let pair=ir::node_at(b,lookup::node(b,e.right)-1).value;let yes_root=ir::statement_at(b,lookup::statement(b,pair.left)-1).value;
        let yes_copy=switch(emit(blocks,11,id,lookup::node(b,yes_root.expression),next,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let yes=switch(block(p,pair.left,yes_copy,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let no_copy=switch(emit(blocks,11,id,lookup::node(b,pair.right),next,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let no=switch(expression(p,pair.right,no_copy,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let branch=switch(emit(blocks,3,lookup::node(b,e.left),0,yes,no,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return expression(p,e.left,branch,loop,depth+1,blocks,c);
    }
    var usize after=switch(drops(p,7,e.start,e.end,next,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    let evaluated=switch(emit(blocks,1,id,0,after,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    if(e.kind==30){let returned=switch(emit(blocks,9,lookup::node(b,e.left),id,0,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let error=switch(drops(p,6,e.start,e.end,returned,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let branch=switch(emit(blocks,4,lookup::node(b,e.left),id,evaluated,error,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return expression(p,e.left,branch,loop,depth+1,blocks,c);}
    let op=ir::node_text(b,id-1);if(e.kind==5 && (bytes::equal(op,b"&&") || bytes::equal(op,b"||"))){
        let right=switch(expression(p,e.right,evaluated,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let skipped=switch(emit(blocks,10,id,lookup::node(b,e.left),next,0,0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let branch=switch(emit(blocks,3,lookup::node(b,e.left),0,if(bytes::equal(op,b"&&")){right}else{skipped},if(bytes::equal(op,b"&&")){skipped}else{right},0,e.start,e.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};return expression(p,e.left,branch,loop,depth+1,blocks,c);
    }
    after=evaluated;
    if(e.kind==6 || e.kind==10 || e.kind==24){var usize link=e.right;while(link!=0){if(!m::charge(ref mut c.work,1)){return .err(fail(c,1002,e.start,e.end));}let pair=ir::node_at(b,lookup::node(b,link)-1).value;after=switch(expression(p,pair.left,after,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};link=pair.right;}return expression(p,e.left,after,loop,depth+1,blocks,c);}
    if(e.kind!=20 && e.kind!=23){after=switch(expression(p,e.right,after,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}return expression(p,e.left,after,loop,depth+1,blocks,c);
}
fn Result<usize,clean_ir::Problem> statement(ref ir::Program p,usize original,usize next,Loop loop,usize depth,ref mut v::Vec<cfg::Block> blocks,ref mut Context c){
    if(original==0){return .ok(next);}if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(fail(c,1002,0,0));}let b=ir::at(p,c.frame);let id=lookup::statement(b,original);if(id==0){return .err(fail(c,1001,0,0));}let s=ir::statement_at(b,id-1).value;
    if(s.kind==1){return block(p,original,next,loop,depth+1,blocks,c);}if(s.kind==13 || s.kind==17 || s.kind==18){return .err(fail(c,1004,s.start,s.end));}
    if(s.kind==9 || s.kind==10){let to=if(s.kind==9){loop.break_to}else{loop.continue_to};if(to==0){return .err(fail(c,1001,s.start,s.end));}return drops(p,if(s.kind==9){3}else{4},s.start,s.end,to,blocks,c);}
    if(s.kind==8 || s.kind==14){
        let anchor=switch(emit(blocks,7,0,0,next,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        var usize initial=0;var usize post=0;if(s.kind==14){let h=ir::statement_at(b,lookup::statement(b,s.second)-1).value;initial=h.first;post=h.second;}
        let after=switch(statement(p,post,anchor,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let inside=switch(block(p,s.first,after,Loop{.break_to=next,.continue_to=after},depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
        let branch=if(s.expression==0){inside}else{switch(emit(blocks,3,lookup::node(b,s.expression),0,inside,next,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}}};
        let condition=switch(expression(p,s.expression,branch,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};{let slot=v::at_mut<cfg::Block>(blocks,anchor-1);slot.first=condition;}
        return statement(p,initial,condition,loop,depth+1,blocks,c);
    }
    var usize after=next;
    if(s.kind==3){after=switch(emit(blocks,6,lookup::node(b,s.expression),0,0,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};after=switch(drops(p,2,s.start,s.end,after,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}
    else if(s.kind==2 || s.kind==4){after=switch(emit(blocks,2,lookup::node(b,s.expression),id,next,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};if(s.kind==4){let target=ir::node_at(b,lookup::node(b,s.target)-1).value;after=switch(drops(p,5,target.start,target.end,after,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}}
    else if(s.kind==11){after=switch(emit(blocks,8,lookup::node(b,s.expression),0,next,0,0,s.start,s.end,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}
    else if(s.kind!=6 && s.kind!=12 && s.kind!=16){return .err(fail(c,1004,s.start,s.end));}
    if(s.kind==12 && s.expression!=0){let e=ir::node_at(b,lookup::node(b,s.expression)-1).value;after=switch(drops(p,8,e.start,e.end,after,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};}
    let entry=switch(expression(p,s.expression,after,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    // A root assignment selects a storage slot; it must not load a moved-out
    // owner before evaluating its replacement. Indexed/projected targets keep
    // their checked receiver/index evaluation before the RHS.
    if(s.kind==4 && ir::node_at(b,lookup::node(b,s.target)-1).value.kind!=2){return expression(p,s.target,entry,loop,depth+1,blocks,c);}return .ok(entry);
}
fn Result<usize,clean_ir::Problem> block(ref ir::Program p,usize original,usize next,Loop loop,usize depth,ref mut v::Vec<cfg::Block> blocks,ref mut Context c){
    if(depth>=64 || !m::charge(ref mut c.work,1)){return .err(fail(c,1002,0,0));}let b=ir::at(p,c.frame);let index=lookup::statement(b,original);if(index==0){return .err(fail(c,1001,0,0));}let root=ir::statement_at(b,index-1).value;if(root.kind!=1){return .err(fail(c,1001,root.start,root.end));}
    var usize entry=switch(drops(p,1,root.start,root.end,next,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    entry=switch(expression(p,root.expression,entry,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    var usize link=root.first;while(link!=0){if(!m::charge(ref mut c.work,1)){return .err(fail(c,1002,root.start,root.end));}let pair=ir::statement_at(b,lookup::statement(b,link)-1).value;entry=switch(statement(p,pair.first,entry,loop,depth+1,blocks,c)){case .ok(n):{n}case .err(p):{return .err(p);}};link=pair.second;}return .ok(entry);
}
pub fn Result<cfg::Graph,clean_ir::Problem> lower(ref ir::Program p,usize frame){
    let checked=proof::check(p);if(checked.code!=0){return .err(checked);}if(frame==0 || frame>ir::count(p)){return .err(clean_ir::Problem{.code=1001,.module_index=0,.start=0,.end=0});}
    let b=ir::at(p,frame);var Context c=Context{.frame,.module_index=clean_ir::frame_at(ir::cleanup_ir(p),frame).module_index,.work=0};
    if(ir::region_count(b)!=0){return .err(fail(ref c,1004,ir::start(b),ir::end(b)));}
    var v::Vec<cfg::Block> blocks=v::empty<cfg::Block>();let result=clean_ir::type_at(ir::cleanup_ir(p),clean_ir::frame_at(ir::cleanup_ir(p),frame).result);let returned=switch(emit(ref mut blocks,if(result.kind==1 && result.scalar==7){6}else{12},0,0,0,0,0,ir::start(b),ir::end(b),ref mut c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    let entry=switch(block(p,ir::body(b),returned,Loop{.break_to=0,.continue_to=0},0,ref mut blocks,ref mut c)){case .ok(n):{n}case .err(p):{return .err(p);}};
    return cfg::assemble(p,frame,entry,blocks,c.work);
}
