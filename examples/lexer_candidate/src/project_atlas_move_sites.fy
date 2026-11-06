module project_atlas_move_sites;
import module_unit_body_ir_model as ir;
import module_unit_body_ir_receiver as receiver;
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

fn Result<bool,u32> owned(ref clean::Program p,usize type,usize depth,ref mut usize meter){
    if(type==0){return .ok(false);}if(depth>=64 || !work::charge(meter,1)){return .err(3);}let t=clean::type_at(p,type);
    if(t.kind==3 || t.kind==13 || t.kind==21){return .ok(true);}if(t.kind!=5 && t.kind!=6){return .ok(false);}
    if(owned(p,t.first,depth+1,meter)?){return .ok(true);}return if(t.kind==5){.ok(false)}else{owned(p,t.second,depth+1,meter)};
}
// Peeling preserves the original complete-root path span. Projections, fresh
// temporaries, Result branches and conditional joins are not root paths.
fn Result<usize,u32> root(ref ir::Body body,usize id,usize depth,ref mut usize meter){
    if(id==0){return .ok(0);}if(depth>=64 || !work::charge(meter,1)){return .err(3);}if(id>ir::node_count(body)){return .err(9);}let e=ir::node_at(body,id-1).value;
    if(e.kind==2){return .ok(if(e.binding!=0){id}else{0});}
    if(e.kind==3){return root(body,ir::node(body,e.left),depth+1,meter);}
    if(e.kind==12 || e.kind==31){let statement=ir::statement(body,e.left);if(statement==0){return .err(9);}let b=ir::statement_at(body,statement-1).value;return root(body,ir::node(body,b.expression),depth+1,meter);}
    return .ok(0);
}
fn Result<bool,u32> mark(ref ir::Program p,ref ir::Body body,usize id,ref mut v::Vec<u8> flags,ref mut usize meter){
    let path=root(body,id,0,meter)?;if(path==0){return .ok(false);}let e=ir::node_at(body,path-1).value;
    if(!owned(ir::cleanup_ir(p),e.type,0,meter)?){return .ok(false);}if(e.binding>ir::local_count(body)){return .err(9);}
    {let slot=v::at_mut<u8>(flags,path-1);*slot=1;}return .ok(true);
}
fn Result<bool,u32> ignored(ref ir::Program p,ref ir::Body body,usize id,ref mut v::Vec<u8> flags,ref mut usize meter){
    var usize current=id;var usize depth=0;while(current!=0){if(depth>=64 || !work::charge(meter,1)){return .err(3);}depth+=1;let e=ir::node_at(body,current-1).value;
        if(e.kind==2){return .ok(false);}if(e.kind!=3){break;}current=ir::node(body,e.left);
    }return mark(p,body,id,flags,meter);
}
fn Result<v::Vec<Site>,u32> scopes(ref ir::Program p,ref cfg::Graph graph,ref mut usize meter){
    var rows=v::empty<Site>();var usize i=1;while(i<=cfg::count(graph)){if(!work::charge(meter,1)){return .err(3);}let b=cfg::at(graph,i);i+=1;if(b.kind!=5){continue;}let d=clean::drop_at(ir::cleanup_ir(p),b.drop-1);
        if(d.exit==1 && !v::push<Site>(ref mut rows,Site{.start=d.start,.end=d.end}).is_ok){return .err(2);}
    }return canonical(rows,65536,meter);
}
fn Result<bool,u32> scope_contains(ref v::Vec<Site> scopes,Site key,ref mut usize meter){
    var usize low=0;var usize high=v::len<Site>(scopes);while(low<high){if(!work::charge(meter,1)){return .err(3);}let middle=low+(high-low)/2;if(before(*v::at<Site>(scopes,middle),key)){low=middle+1;}else{high=middle;}}
    return .ok(low<v::len<Site>(scopes) && v::at<Site>(scopes,low).start==key.start && v::at<Site>(scopes,low).end==key.end);
}
fn Result<bool,u32> block_result(ref ir::Program p,ref ir::Body body,usize statement,ref v::Vec<Site> scopes,ref mut v::Vec<u8> flags,ref mut usize meter){
    if(statement==0){return .err(9);}let b=ir::statement_at(body,statement-1).value;let id=root(body,ir::node(body,b.expression),0,meter)?;if(id==0){return .ok(false);}let e=ir::node_at(body,id-1).value;
    if(e.binding==0 || e.binding>ir::local_count(body)){return .err(9);}let local=ir::local_at(body,e.binding);
    // Before capture, a live tail-local itself is an in-scope obligation even
    // though its post-transfer cleanup is absent from the verified drop plan.
    let local_scope=local.start>=b.start && local.end<=b.end;
    if(local_scope || scope_contains(scopes,Site{.start=b.start,.end=b.end},meter)?){discard mark(p,body,id,flags,meter)?;}return .ok(true);
}
fn Result<v::Vec<u8>,u32> consumers(ref ir::Program p,usize frame,ref cfg::Graph graph,ref mut usize meter){
    let body=ir::at(p,frame);let scopes=scopes(p,graph,meter)?;var flags=v::empty<u8>();var usize i=0;
    while(i<ir::node_count(body)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut flags,0).is_ok){return .err(2);}i+=1;}
    // Only consuming dot-call receivers move; implicit shared/exclusive
    // receivers retain their owner. Tagged
    // constructors also retain original argument wrappers in the node table.
    i=0;while(i<ir::argument_count(body)){if(!work::charge(meter,1)){return .err(3);}let a=ir::argument_at(body,i);if(!receiver::implicit(ir::cleanup_ir(p),body,a)){discard mark(p,body,a.expression,ref mut flags,meter)?;}i+=1;}
    i=0;while(i<ir::statement_count(body)){if(!work::charge(meter,1)){return .err(3);}let s=ir::statement_at(body,i).value;i+=1;
        if(s.kind==2 || s.kind==3 || s.kind==4){discard mark(p,body,ir::node(body,s.expression),ref mut flags,meter)?;}
        else if(s.kind==6 || s.kind==12){discard ignored(p,body,ir::node(body,s.expression),ref mut flags,meter)?;}
    }
    let entry=ir::statement_at(body,ir::statement(body,ir::body(body))-1).value;
    if(owned(ir::cleanup_ir(p),clean::frame_at(ir::cleanup_ir(p),frame).result,0,meter)?){discard mark(p,body,ir::node(body,entry.expression),ref mut flags,meter)?;}
    i=0;while(i<ir::node_count(body)){if(!work::charge(meter,1)){return .err(3);}let e=ir::node_at(body,i).value;i+=1;
        if(e.kind==7){discard mark(p,body,ir::node(body,e.left),ref mut flags,meter)?;}
        if(e.kind==11 && e.field.owner!=0 && owned(ir::cleanup_ir(p),e.field.owner,0,meter)?){discard mark(p,body,ir::node(body,e.left),ref mut flags,meter)?;}
        if(e.kind==12 || e.kind==31){discard block_result(p,body,ir::statement(body,e.left),ref scopes,ref mut flags,meter)?;}
        // The then arm is a statement block, unlike the else block expression.
        // Use the same scoped obligation rule; a value type alone is not a move.
        if(e.kind==13){let pair=ir::node_at(body,ir::node(body,e.right)-1).value;discard block_result(p,body,ir::statement(body,pair.left),ref scopes,ref mut flags,meter)?;}
        if(e.kind!=15){continue;}discard mark(p,body,ir::node(body,e.left),ref mut flags,meter)?;
        if(!owned(ir::cleanup_ir(p),e.type,0,meter)?){continue;}var usize link=e.right;
        while(link!=0){if(!work::charge(meter,1)){return .err(3);}let pair=ir::node_at(body,ir::node(body,link)-1).value;let arm=ir::node_at(body,ir::node(body,pair.left)-1).value;let b=ir::statement_at(body,ir::statement(body,arm.right)-1).value;
            discard mark(p,body,ir::node(body,b.expression),ref mut flags,meter)?;link=pair.right;
        }
    }return .ok(flags);
}
fn Result<bool,u32> schedule(ref mut v::Vec<usize> pending,ref mut v::Vec<u8> seen,usize id,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}if(id==0 || id>=v::len<u8>(seen)){return .err(9);}if(*v::at<u8>(seen,id)!=0){return .ok(false);}
    if(!v::push<usize>(pending,id).is_ok){return .err(2);}{let slot=v::at_mut<u8>(seen,id);*slot=1;}return .ok(true);
}
fn Result<v::Vec<Site>,u32> visit(ref ir::Program p,usize frame,ref cfg::Graph graph,usize maximum,ref mut usize meter){
    if(!work::charge(meter,cfg::work(graph))){return .err(3);}let body=ir::at(p,frame);let flags=consumers(p,frame,graph,meter)?;
    var seen=v::empty<u8>();var usize i=0;while(i<=cfg::count(graph)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(2);}i+=1;}
    var pending=v::empty<usize>();discard schedule(ref mut pending,ref mut seen,cfg::entry(graph),meter)?;var rows=v::empty<Site>();i=0;
    while(i<v::len<usize>(ref pending)){if(!work::charge(meter,1)){return .err(3);}let b=cfg::at(graph,*v::at<usize>(ref pending,i));i+=1;
        if(b.kind==8){let e=ir::node_at(body,b.source-1).value;if(e.kind==1 && bytes::equal(ir::node_text(body,b.source-1),b"false")){continue;}}
        if(b.kind==1 && *v::at<u8>(ref flags,b.source-1)!=0){let e=ir::node_at(body,b.source-1).value;if(v::len<Site>(ref rows)>=65536){return .err(3);}if(!v::push<Site>(ref mut rows,Site{.start=e.start,.end=e.end}).is_ok){return .err(2);}}
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
