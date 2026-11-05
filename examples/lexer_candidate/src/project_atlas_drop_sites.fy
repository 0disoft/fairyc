module project_atlas_drop_sites;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_lower as lower;
import module_unit_cfg_model as cfg;
import project_path as work;
import project_atlas_trap_kinds as traps;
import std::bytes as bytes;
import std::vec as v;

// Source-relative executable cleanup observations from one verified frame.
// Identity and current-source authority remain with the Complete-bound caller.
pub struct Site{usize start;usize end;}
pub struct Range{usize start;usize end;}
pub resource Collection{v::Vec<Site> sites;v::Vec<Range> frames;v::Vec<u32> traps;cleanup{}}
pub fn usize frame_count(ref Collection p){return v::len<Range>(ref p.frames);}
pub fn Range range(ref Collection p,usize index){return *v::at<Range>(ref p.frames,index);}
pub fn Site at(ref Collection p,usize index){return *v::at<Site>(ref p.sites,index);}
// Present only in a collection from collect_observed_all.
pub fn u32 trap_at(ref Collection p,usize index){return *v::at<u32>(ref p.traps,index);}
fn bool before(Site a,Site b){return a.start<b.start || a.start==b.start && a.end<b.end;}
fn Result<v::Vec<Site>,u32> canonical(v::Vec<Site> input,usize maximum,ref mut usize meter){
    var rows=input;let n=v::len<Site>(ref rows);var spare=v::empty<Site>();var usize i=0;
    while(i<n){if(!work::charge(meter,1)){return .err(3);}if(!v::push<Site>(ref mut spare,Site{.start=0,.end=0}).is_ok){return .err(2);}i+=1;}
    var usize width=1;while(width<n){var usize start=0;while(start<n){let middle=if(start+width<n){start+width}else{n};let end=if(middle+width<n){middle+width}else{n};var usize left=start;var usize right=middle;var usize out=start;
        while(out<end){if(!work::charge(meter,1)){return .err(3);}var Site selected=Site{.start=0,.end=0};
            if(left<middle && (right==end || !before(*v::at<Site>(ref rows,right),*v::at<Site>(ref rows,left)))){selected=*v::at<Site>(ref rows,left);left+=1;}else{selected=*v::at<Site>(ref rows,right);right+=1;}
            {let slot=v::at_mut<Site>(ref mut spare,out);*slot=selected;}out+=1;
        }start=end;
    }i=0;while(i<n){if(!work::charge(meter,1)){return .err(3);}let value=*v::at<Site>(ref spare,i);{let slot=v::at_mut<Site>(ref mut rows,i);*slot=value;}i+=1;}width*=2;}
    var result=v::empty<Site>();i=0;while(i<n){if(!work::charge(meter,1)){return .err(3);}let value=*v::at<Site>(ref rows,i);i+=1;
        if(i>1){let previous=*v::at<Site>(ref rows,i-2);if(value.start==previous.start && value.end==previous.end){continue;}}
        if(v::len<Site>(ref result)>=maximum){return .err(3);}if(!v::push<Site>(ref mut result,value).is_ok){return .err(2);}
    }return .ok(result);
}
pub fn Result<v::Vec<Site>,u32> collect(ref ir::Program program,usize frame,usize maximum,ref mut usize meter){
    if(frame==0 || frame>ir::count(program) || maximum>65536){return .err(9);}
    if(!work::charge(meter,1)){return .err(3);}
    let graph=switch(lower::lower(program,frame)){case .ok(p):{p}case .err(e):{return .err(if(e.code==1002){3}else if(e.code==1003){2}else{9});}};
    var rows=v::empty<Site>();discard visit(program,frame,ref graph,true,false,0,ref mut rows,meter)?;return canonical(rows,maximum,meter);
}
pub fn Result<u32,u32> collect_traps(ref ir::Program program,usize frame,usize maximum,ref mut usize meter){
    if(frame==0 || frame>ir::count(program) || maximum>6){return .err(9);}if(!work::charge(meter,1)){return .err(3);}
    let graph=switch(lower::lower(program,frame)){case .ok(p):{p}case .err(e):{return .err(if(e.code==1002){3}else if(e.code==1003){2}else{9});}};
    var rows=v::empty<Site>();return visit(program,frame,ref graph,false,true,maximum,ref mut rows,meter);
}
pub fn Result<Collection,u32> collect_all(ref ir::Program program,usize maximum,ref mut usize meter){
    var sites=v::empty<Site>();var frames=v::empty<Range>();var masks=v::empty<u32>();discard batch(program,maximum,0,true,false,ref mut sites,ref mut frames,ref mut masks,meter)?;return .ok(Collection{.sites,.frames,.traps=masks});
}
pub fn Result<v::Vec<u32>,u32> collect_all_traps(ref ir::Program program,usize maximum,ref mut usize meter){
    var sites=v::empty<Site>();var frames=v::empty<Range>();var masks=v::empty<u32>();discard batch(program,0,maximum,false,true,ref mut sites,ref mut frames,ref mut masks,meter)?;return .ok(masks);
}
pub fn Result<Collection,u32> collect_observed_all(ref ir::Program program,usize maximum_drops,usize maximum_traps,ref mut usize meter){
    var sites=v::empty<Site>();var frames=v::empty<Range>();var masks=v::empty<u32>();discard batch(program,maximum_drops,maximum_traps,true,true,ref mut sites,ref mut frames,ref mut masks,meter)?;return .ok(Collection{.sites,.frames,.traps=masks});
}
// Only locally constructed graphs reach visit. One whole-IR proof and one
// traversal per frame serve both families; each single-family port stays isolated.
fn Result<bool,u32> batch(ref ir::Program program,usize maximum_drops,usize maximum_traps,bool with_drops,bool with_traps,ref mut v::Vec<Site> sites,ref mut v::Vec<Range> frames,ref mut v::Vec<u32> masks,ref mut usize meter){
    if(maximum_drops>65536 || maximum_traps>6){return .err(9);}if(!work::charge(meter,1)){return .err(3);}
    let graphs=switch(lower::lower_all(program)){case .ok(p):{p}case .err(e):{return .err(if(e.code==1002){3}else if(e.code==1003){2}else{9});}};
    if(v::len<cfg::Graph>(ref graphs)!=ir::count(program)){return .err(9);}var usize i=0;
    while(i<v::len<cfg::Graph>(ref graphs)){
        var raw=v::empty<Site>();let mask=visit(program,i+1,v::at<cfg::Graph>(ref graphs,i),with_drops,with_traps,maximum_traps,ref mut raw,meter)?;
        let rows=canonical(raw,maximum_drops,meter)?;
        if(with_drops){let start=v::len<Site>(sites);var usize j=0;
            while(j<v::len<Site>(ref rows)){if(!work::charge(meter,1) || v::len<Site>(sites)>=65536){return .err(3);}if(!v::push<Site>(sites,*v::at<Site>(ref rows,j)).is_ok){return .err(2);}j+=1;}
            if(!work::charge(meter,1)){return .err(3);}if(!v::push<Range>(frames,Range{.start,.end=v::len<Site>(sites)}).is_ok){return .err(2);}
        }
        if(with_traps){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u32>(masks,mask).is_ok){return .err(2);}}i+=1;
    }return .ok(true);
}
fn Result<u32,u32> visit(ref ir::Program program,usize frame,ref cfg::Graph graph,bool with_drops,bool with_traps,usize maximum_traps,ref mut v::Vec<Site> rows,ref mut usize meter){
    if(!work::charge(meter,cfg::work(graph))){return .err(3);}
    let plan=ir::cleanup_ir(program);let body=ir::at(program,frame);var seen=v::empty<u8>();var usize i=0;
    while(i<=cfg::count(graph)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(2);}i+=1;}
    var pending=v::empty<usize>();discard schedule(ref mut pending,ref mut seen,cfg::entry(graph),meter)?;i=0;var u32 mask=0;
    while(i<v::len<usize>(ref pending)){
        if(!work::charge(meter,1)){return .err(3);}let n=cfg::at(graph,*v::at<usize>(ref pending,i));i+=1;
        // Match Go MIR's explicit assertion-trap terminal. Other conditional
        // paths stay represented even when their operands happen to be literals.
        if(n.kind==8){if(with_traps){mask|=1;}if(ir::node_at(body,n.source-1).value.kind==1 && bytes::equal(ir::node_text(body,n.source-1),b"false")){continue;}}
        if(with_traps){
            if(n.kind==21){mask|=10;}
            if(n.kind==1){let e=ir::node_at(body,n.source-1).value;
                if(e.kind==9 || e.kind==26){mask|=2;}
                else if(e.kind==4 && !e.minimum || e.kind==5){let t=clean::type_at(plan,e.type);let op=ir::node_text(body,n.source-1);if(!work::charge(meter,op.len)){return .err(3);}mask|=traps::arithmetic(op,if(t.kind==1){t.scalar}else{0},e.kind==4);}
            }else if(n.kind==2){let statement=ir::statement_at(body,n.target-1).value;if(statement.kind==4){let op=ir::assignment_operator(body,n.target-1);if(!work::charge(meter,op.len)){return .err(3);}if(op.len>1){let target=ir::node_at(body,ir::node(body,statement.target)-1).value;let t=clean::type_at(plan,target.type);mask|=traps::arithmetic(op[..op.len-1],if(t.kind==1){t.scalar}else{0},false);}}}
        }
        if(n.first!=0){discard schedule(ref mut pending,ref mut seen,n.first,meter)?;}if(n.second!=0){discard schedule(ref mut pending,ref mut seen,n.second,meter)?;}
        if(!with_drops || n.kind!=5){continue;}let drop=clean::drop_at(plan,n.drop-1);let operand=clean::operand_at(plan,drop.operand);if(operand.frame!=frame){return .err(9);}
        // Drop.start/end identify the exit that schedules cleanup. Atlas owns
        // the consumed binding/temporary span, not that shared exit location.
        if(v::len<Site>(rows)>=65536){return .err(3);}if(!v::push<Site>(rows,Site{.start=operand.start,.end=operand.end}).is_ok){return .err(2);}
    }
    if(with_traps){var usize count=0;i=0;while(i<6){if(!work::charge(meter,1)){return .err(3);}if(traps::contains(mask,i)){count+=1;}i+=1;}if(count>maximum_traps){return .err(3);}}return .ok(mask);
}
fn Result<bool,u32> schedule(ref mut v::Vec<usize> pending,ref mut v::Vec<u8> seen,usize id,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}if(id==0 || id>=v::len<u8>(seen)){return .err(9);}if(*v::at<u8>(seen,id)!=0){return .ok(false);}
    if(!v::push<usize>(pending,id).is_ok){return .err(2);}{let slot=v::at_mut<u8>(seen,id);*slot=1;}return .ok(true);
}
