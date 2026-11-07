module module_unit_cfg_model;
import module_unit_body_ir_model as bodies;
import module_unit_cleanup_mir_model as clean_ir;
import module_unit_value_model as budget;
import module_unit_cfg_defer as deferred;
import module_unit_cfg_region as lifecycle;
import module_unit_cfg_switches as switches;
import module_unit_cfg_iteration as iteration;
import std::bytes as bytes;
import std::vec as v;
import module_unit_cfg_drops as drops;

// One operation per block. source/target are body-table IDs, never parser tokens.
// 1 eval, 2 store, 3 bool branch, 4 Result branch, 5 drop, 6 return,
// 7 jump, 8 assert, 9 propagate error, 10 short-circuit copy, 11 branch/block copy,
// 12 unreachable completion trap, 13 frozen scalar capture, 14 deferred invocation,
// 15 Region checkpoint, 16 Region reset, 17 compound-assignment snapshot,
// 18 switch case (source=switch node, target=arm node), 19 active payload seed.
// 20 frozen byte-iteration view, 21 bounds test plus copied iteration binding.
pub struct Block {u32 kind;usize source;usize target;usize first;usize second;usize drop;usize start;usize end;}
pub struct Deferred {usize statement;usize invocation;usize capture_start;usize capture_count;}
pub struct Capture {usize deferred;usize position;usize source;usize type;}
pub resource Graph {usize frame;usize entry;v::Vec<Block> blocks;v::Vec<Deferred> deferred;v::Vec<Capture> captures;v::Vec<usize> drop_blocks;usize work;cleanup{}}
pub fn usize count(ref Graph p){return v::len<Block>(ref p.blocks);}
pub fn Block at(ref Graph p,usize id){assert(id!=0 && id<=count(p));return *v::at<Block>(ref p.blocks,id-1);}
pub fn usize frame(ref Graph p){return p.frame;}
pub fn usize entry(ref Graph p){return p.entry;}
pub fn usize work(ref Graph p){return p.work;}
// Derived while validating the immutable blocks, in original CFG order. This
// private projection grants no independent graph or cleanup proof authority.
pub fn usize drop_count(ref Graph p){return v::len<usize>(ref p.drop_blocks);}
pub fn usize drop_block(ref Graph p,usize row){return *v::at<usize>(ref p.drop_blocks,row);}
pub fn usize deferred_count(ref Graph p){return v::len<Deferred>(ref p.deferred);}
pub fn Deferred deferred_at(ref Graph p,usize id){return *v::at<Deferred>(ref p.deferred,id-1);}
pub fn usize capture_count(ref Graph p){return v::len<Capture>(ref p.captures);}
pub fn Capture capture_at(ref Graph p,usize id){return *v::at<Capture>(ref p.captures,id-1);}
fn clean_ir::Problem fail(u32 code,usize module_index,usize start,usize end){return clean_ir::Problem{.code,.module_index,.start,.end};}
fn clean_ir::Problem check_blocks(ref bodies::Program p,usize frame,usize entry,ref v::Vec<Block> blocks,ref v::Vec<Deferred> deferred,ref v::Vec<Capture> captures,bool record,ref mut v::Vec<usize> drop_blocks,ref mut usize used){
    if(frame==0 || frame>bodies::count(p) || entry==0 || entry>v::len<Block>(blocks)){return fail(1001,0,0,0);}
    let b=bodies::at(p,frame);let cleaned=bodies::cleanup_ir(p);let f=clean_ir::frame_at(cleaned,frame);
    if(v::len<Block>(blocks)>32768 || *used>1048576){return fail(1002,f.module_index,0,0);}
    let table=switch(drops::build(p,frame,used)){case .ok(t):{t}case .err(e):{return e;}};var v::Vec<u8> seen=v::empty<u8>();var usize i=0;while(i<drops::count(ref table)){if(!budget::charge(used,1)){return fail(1002,f.module_index,0,0);}if(!v::push<u8>(ref mut seen,0).is_ok){return fail(1003,f.module_index,0,0);}i+=1;}
    var v::Vec<u8> snapshots=v::empty<u8>();i=0;while(i<bodies::statement_count(b)){if(!budget::charge(used,1)){return fail(1002,f.module_index,0,0);}if(!v::push<u8>(ref mut snapshots,0).is_ok){return fail(1003,f.module_index,0,0);}i+=1;}
    i=1;while(i<=v::len<Block>(blocks)){
        if(!budget::charge(used,1)){return fail(1002,f.module_index,0,0);}let n=*v::at<Block>(blocks,i-1);
        let target_limit=if(n.kind==2 || n.kind==17 || n.kind==20 || n.kind==21){bodies::statement_count(b)}else if(n.kind==13){v::len<Capture>(captures)}else if(n.kind==14){v::len<Deferred>(deferred)}else if(n.kind==15){bodies::region_count(b)}else if(n.kind==16){bodies::reset_count(b)}else{bodies::node_count(b)};
        if(n.kind==0 || n.kind>21 || n.start>n.end || n.end>bodies::source_bytes(b) || n.source>bodies::node_count(b) || n.target>target_limit || (n.kind==2 || n.kind==13 || n.kind==14 || n.kind==15 || n.kind==16 || n.kind==17 || n.kind==18 || n.kind==19 || n.kind==20 || n.kind==21) && n.target==0 || n.first>v::len<Block>(blocks) || n.second>v::len<Block>(blocks)){return fail(1001,f.module_index,n.start,n.end);}
        let branch=n.kind==3 || n.kind==4 || n.kind==18 || n.kind==21;let terminal=n.kind==6 || n.kind==9 || n.kind==12;
        if(branch && (n.first==0 || n.second==0 || n.source==0) || !branch && n.second!=0 || terminal && n.first!=0 || !terminal && n.first==0 || (n.kind==1 || n.kind==2 || n.kind==8 || n.kind==10 || n.kind==11 || n.kind==13 || n.kind==14 || n.kind==15 || n.kind==17 || n.kind==19 || n.kind==20) && n.source==0){return fail(1001,f.module_index,n.start,n.end);}
        if((n.kind==5 || n.kind==7 || n.kind==12) && (n.source!=0 || n.target!=0)){return fail(1001,f.module_index,n.start,n.end);}
        if(n.source!=0){let e=bodies::node_at(b,n.source-1).value;if(e.state!=1 || e.type==0){return fail(1001,f.module_index,n.start,n.end);}let t=clean_ir::type_at(cleaned,e.type);
            if((n.kind==3 || n.kind==8) && (t.kind!=1 || t.scalar!=6)){return fail(1001,f.module_index,n.start,n.end);}
            if(n.kind==2){let s=bodies::statement_at(b,n.target-1).value;if((s.kind!=2 && s.kind!=4) || bodies::node(b,s.expression)!=n.source){return fail(1001,f.module_index,n.start,n.end);}}
            if(n.kind==17){let stmt=bodies::statement_at(b,n.target-1).value;if(stmt.kind!=4 || bodies::node(b,stmt.target)!=n.source || bytes::equal(bodies::assignment_operator(b,n.target-1),b"=") || t.kind!=1 || t.scalar<1 || t.scalar>5 || n.start!=stmt.start || n.end!=stmt.end || *v::at<u8>(ref snapshots,n.target-1)!=0){return fail(1001,f.module_index,n.start,n.end);}let slot=v::at_mut<u8>(ref mut snapshots,n.target-1);*slot=1;}
            if(n.kind==4 || n.kind==9){if(t.kind!=6 || n.target==0){return fail(1001,f.module_index,n.start,n.end);}let target=bodies::node_at(b,n.target-1).value;if(target.kind!=30 || bodies::node(b,target.left)!=n.source || target.type!=t.first){return fail(1001,f.module_index,n.start,n.end);}if(n.kind==9){let result=clean_ir::type_at(cleaned,f.result);if(result.kind!=6 || result.second!=t.second){return fail(1001,f.module_index,n.start,n.end);}}}
            if(n.kind==6 && e.type!=f.result){return fail(1001,f.module_index,n.start,n.end);}
            if(n.kind==10){let op=bodies::node_text(b,n.source-1);if(e.kind!=5 || (!bytes::equal(op,b"&&") && !bytes::equal(op,b"||")) || bodies::node(b,e.left)!=n.target){return fail(1001,f.module_index,n.start,n.end);}}
            if(n.kind==11){if(e.kind!=12 && e.kind!=13 && e.kind!=15 && e.kind!=31){return fail(1001,f.module_index,n.start,n.end);}if(n.target!=0 && bodies::node_at(b,n.target-1).value.type!=e.type){return fail(1001,f.module_index,n.start,n.end);}}
        }else if(n.kind==6){let result=clean_ir::type_at(cleaned,f.result);if(result.kind!=1 || result.scalar!=7){return fail(1001,f.module_index,n.start,n.end);}}
        if(n.kind==5){if(n.drop==0 || n.drop>clean_ir::drop_count(cleaned)){return fail(1001,f.module_index,n.start,n.end);}let d=clean_ir::drop_at(cleaned,n.drop-1);let o=clean_ir::operand_at(cleaned,d.operand);let row=drops::row(ref table,n.drop,used);if(row==0){return fail(if(*used>=1048576){1002}else{1001},f.module_index,n.start,n.end);}if(o.frame!=frame || n.start!=d.start || n.end!=d.end || *v::at<u8>(ref seen,row-1)!=0){return fail(1001,f.module_index,n.start,n.end);}// The ownership walk may revisit a loop condition while validating its
            // back edge. Those identical requests describe one runtime site.
            var usize alias=0;while(alias<drops::count(ref table)){if(!budget::charge(used,1)){return fail(1002,f.module_index,n.start,n.end);}let other=clean_ir::drop_at(cleaned,drops::at(ref table,alias)-1);if(other.operand==d.operand && other.exit==d.exit && other.start==d.start && other.end==d.end){if(*v::at<u8>(ref seen,alias)!=0){return fail(1001,f.module_index,n.start,n.end);}let slot=v::at_mut<u8>(ref mut seen,alias);*slot=1;}alias+=1;}
            if(record && !v::push<usize>(drop_blocks,i).is_ok){return fail(1003,f.module_index,n.start,n.end);}
        }else if(n.drop!=0){return fail(1001,f.module_index,n.start,n.end);}i+=1;
    }
    i=0;while(i<bodies::statement_count(b)){if(!budget::charge(used,1)){return fail(1002,f.module_index,0,0);}let stmt=bodies::statement_at(b,i).value;if(stmt.kind==4 && !bytes::equal(bodies::assignment_operator(b,i),b"=") && *v::at<u8>(ref snapshots,i)!=1){return fail(1001,f.module_index,stmt.start,stmt.end);}i+=1;}
    i=0;while(i<drops::count(ref table)){if(!budget::charge(used,1)){return fail(1002,f.module_index,0,0);}if(*v::at<u8>(ref seen,i)==0){return fail(1001,f.module_index,0,0);}i+=1;}
    return fail(0,0,0,0);
}
fn clean_ir::Problem check_tail(ref bodies::Program p,ref Graph g,ref mut usize used){
    let repeated=iteration::check(p,g,used);if(repeated.code!=0){return repeated;}let selected=switches::check(p,g,used);if(selected.code!=0){return selected;}let checked=deferred::check(p,g,used);if(checked.code!=0){return checked;}return lifecycle::check(p,g,used);
}
pub fn clean_ir::Problem check(ref bodies::Program p,ref Graph g){
    var v::Vec<usize> ignored=v::empty<usize>();var usize used=work(g);let checked=check_blocks(p,frame(g),entry(g),ref g.blocks,ref g.deferred,ref g.captures,false,ref mut ignored,ref mut used);if(checked.code!=0){return checked;}return check_tail(p,g,ref mut used);
}
pub fn Result<Graph,clean_ir::Problem> assemble_deferred(ref bodies::Program p,usize frame,usize entry,v::Vec<Block> blocks,v::Vec<Deferred> deferred,v::Vec<Capture> captures,usize work){
    // Collect from the same charged validation walk before moving the owning
    // vectors. Resource fields remain immutable after construction.
    var v::Vec<usize> drop_blocks=v::empty<usize>();var usize used=work;let checked=check_blocks(p,frame,entry,ref blocks,ref deferred,ref captures,true,ref mut drop_blocks,ref mut used);if(checked.code!=0){return .err(checked);}
    let g=Graph{.frame,.entry,.blocks,.deferred,.captures,.drop_blocks,.work};let tail=check_tail(p,ref g,ref mut used);if(tail.code!=0){return .err(tail);}return .ok(g);
}
pub fn Result<Graph,clean_ir::Problem> assemble(ref bodies::Program p,usize frame,usize entry,v::Vec<Block> blocks,usize work){return assemble_deferred(p,frame,entry,blocks,v::empty<Deferred>(),v::empty<Capture>(),work);}
