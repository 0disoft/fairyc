module project_atlas_return_error_sites;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_lower as lower;
import module_unit_cfg_model as cfg;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Source-relative executable observations. The Complete-bound producer retains
// package/source authority; callers cannot supply a graph or observed site list.
pub struct Site{usize start;usize end;}
pub struct Range{usize start;usize end;}
pub resource Collection{v::Vec<Site> sites;v::Vec<Range> frames;cleanup{}}
pub fn usize frame_count(ref Collection p){return v::len<Range>(ref p.frames);}
pub fn Range range(ref Collection p,usize index){return *v::at<Range>(ref p.frames,index);}
pub fn Site at(ref Collection p,usize index){return *v::at<Site>(ref p.sites,index);}
fn u32 failure(clean::Problem e){return if(e.code==1002){3}else if(e.code==1003){2}else{9};}
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

// Observe the reached Result-tag branch after the operand completes.
// The source fact is the written question token, not its implicit error return.
fn Result<bool,u32> schedule(ref mut v::Vec<usize> pending,ref mut v::Vec<u8> seen,usize id,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}if(id==0 || id>=v::len<u8>(seen)){return .err(9);}if(*v::at<u8>(seen,id)!=0){return .ok(false);}
    if(!v::push<usize>(pending,id).is_ok){return .err(2);}{let slot=v::at_mut<u8>(seen,id);*slot=1;}return .ok(true);
}
fn Result<v::Vec<Site>,u32> visit(ref ir::Program p,usize frame,ref cfg::Graph graph,usize maximum,ref mut usize meter){
    if(!work::charge(meter,cfg::work(graph))){return .err(3);}let body=ir::at(p,frame);
    var seen=v::empty<u8>();var usize i=0;while(i<=cfg::count(graph)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(2);}i+=1;}
    var pending=v::empty<usize>();discard schedule(ref mut pending,ref mut seen,cfg::entry(graph),meter)?;var rows=v::empty<Site>();i=0;
    while(i<v::len<usize>(ref pending)){if(!work::charge(meter,1)){return .err(3);}let b=cfg::at(graph,*v::at<usize>(ref pending,i));i+=1;
        if(b.kind==8){let e=ir::node_at(body,b.source-1).value;if(e.kind==1 && bytes::equal(ir::node_text(body,b.source-1),b"false")){continue;}}
        if(b.kind==4){
            if(b.target==0 || b.target>ir::node_count(body)){return .err(9);}let e=ir::node_at(body,b.target-1).value;
            if(e.kind!=30 || e.end<=e.start || b.source!=ir::node(body,e.left) || b.start!=e.start || b.end!=e.end){return .err(9);}
            if(v::len<Site>(ref rows)>=65536){return .err(3);}if(!v::push<Site>(ref mut rows,Site{.start=e.end-1,.end=e.end}).is_ok){return .err(2);}
        }
        if(b.first!=0){discard schedule(ref mut pending,ref mut seen,b.first,meter)?;}if(b.second!=0){discard schedule(ref mut pending,ref mut seen,b.second,meter)?;}
    }return canonical(rows,maximum,meter);
}
pub fn Result<v::Vec<Site>,u32> collect(ref ir::Program p,usize frame,usize maximum,ref mut usize meter){
    if(frame==0 || frame>ir::count(p) || maximum>65536){return .err(9);}if(!work::charge(meter,1)){return .err(3);}
    let graph=switch(lower::lower(p,frame)){case .ok(p):{p}case .err(e):{return .err(failure(e));}};return visit(p,frame,ref graph,maximum,meter);
}
pub fn Result<Collection,u32> collect_all(ref ir::Program p,usize maximum,ref mut usize meter){
    if(maximum>65536){return .err(9);}if(!work::charge(meter,1)){return .err(3);}let graphs=switch(lower::lower_all(p)){case .ok(p):{p}case .err(e):{return .err(failure(e));}};
    if(v::len<cfg::Graph>(ref graphs)!=ir::count(p)){return .err(9);}var sites=v::empty<Site>();var frames=v::empty<Range>();var usize i=0;
    while(i<v::len<cfg::Graph>(ref graphs)){let rows=visit(p,i+1,v::at<cfg::Graph>(ref graphs,i),maximum,meter)?;let start=v::len<Site>(ref sites);var usize j=0;
        while(j<v::len<Site>(ref rows)){if(!work::charge(meter,1) || v::len<Site>(ref sites)>=65536){return .err(3);}if(!v::push<Site>(ref mut sites,*v::at<Site>(ref rows,j)).is_ok){return .err(2);}j+=1;}
        if(!work::charge(meter,1)){return .err(3);}if(!v::push<Range>(ref mut frames,Range{.start,.end=v::len<Site>(ref sites)}).is_ok){return .err(2);}i+=1;
    }return .ok(Collection{.sites,.frames});
}
