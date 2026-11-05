module project_atlas_drop_sites;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_lower as lower;
import module_unit_cfg_model as cfg;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Source-relative executable cleanup observations from one verified frame.
// Identity and current-source authority remain with the Complete-bound caller.
pub struct Site{usize start;usize end;}
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
    if(!work::charge(meter,cfg::work(ref graph))){return .err(3);}
    let plan=ir::cleanup_ir(program);let body=ir::at(program,frame);var rows=v::empty<Site>();var seen=v::empty<u8>();var usize i=0;
    while(i<=cfg::count(ref graph)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(2);}i+=1;}
    var pending=v::empty<usize>();discard schedule(ref mut pending,ref mut seen,cfg::entry(ref graph),meter)?;i=0;
    while(i<v::len<usize>(ref pending)){
        if(!work::charge(meter,1)){return .err(3);}let n=cfg::at(ref graph,*v::at<usize>(ref pending,i));i+=1;
        // Match Go MIR's explicit assertion-trap terminal. Other conditional
        // paths stay represented even when their operands happen to be literals.
        if(n.kind==8 && ir::node_at(body,n.source-1).value.kind==1 && bytes::equal(ir::node_text(body,n.source-1),b"false")){continue;}
        if(n.first!=0){discard schedule(ref mut pending,ref mut seen,n.first,meter)?;}if(n.second!=0){discard schedule(ref mut pending,ref mut seen,n.second,meter)?;}
        if(n.kind!=5){continue;}let drop=clean::drop_at(plan,n.drop-1);let operand=clean::operand_at(plan,drop.operand);if(operand.frame!=frame){return .err(9);}
        // Drop.start/end identify the exit that schedules cleanup. Atlas owns
        // the consumed binding/temporary span, not that shared exit location.
        if(v::len<Site>(ref rows)>=65536){return .err(3);}if(!v::push<Site>(ref mut rows,Site{.start=operand.start,.end=operand.end}).is_ok){return .err(2);}
    }return canonical(rows,maximum,meter);
}
fn Result<bool,u32> schedule(ref mut v::Vec<usize> pending,ref mut v::Vec<u8> seen,usize id,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}if(id==0 || id>=v::len<u8>(seen)){return .err(9);}if(*v::at<u8>(seen,id)!=0){return .ok(false);}
    if(!v::push<usize>(pending,id).is_ok){return .err(2);}{let slot=v::at_mut<u8>(seen,id);*slot=1;}return .ok(true);
}
