module project_atlas_unsafe_sites;
import module_unit_body_ir_model as ir;
import module_unit_cleanup_mir_model as clean;
import module_unit_cfg_lower as lower;
import module_unit_cfg_model as cfg;
import module_unit_sdk_sources as sdk;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

// Source-relative observations. Complete-bound publication owns package/source
// identity. No public factory accepts a graph, permission flag or site list.
pub struct Site{usize start;usize end;}
pub struct Range{usize start;usize end;}
struct Leaf{usize module_index;Site site;}
pub resource Collection{v::Vec<Site> sites;v::Vec<Range> frames;v::Vec<Leaf> leaves;cleanup{}}
pub fn usize frame_count(ref Collection p){return v::len<Range>(ref p.frames);}
pub fn Range range(ref Collection p,usize index){return *v::at<Range>(ref p.frames,index);}
pub fn Site at(ref Collection p,usize index){return *v::at<Site>(ref p.sites,index);}
// A witnessed folded helper has no frame. Its original unsafe operation belongs
// to the helper's source, not to the safe call in the receiver cleanup.
pub fn Result<Site,u32> leaf(ref Collection p,usize module_index,ref mut usize meter){var usize i=0;while(i<v::len<Leaf>(ref p.leaves)){if(!work::charge(meter,1)){return .err(3);}let row=*v::at<Leaf>(ref p.leaves,i);if(row.module_index==module_index){return .ok(row.site);}i+=1;}return .err(9);}
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
// Sorted module/declaration pairs are derived from verified callable contracts.
// Merely entering an unsafe block does not make an ordinary call unsafe.
fn Result<v::Vec<Site>,u32> foreign_targets(ref ir::Program program,ref mut usize meter){
    var raw=v::empty<Site>();var usize i=0;
    while(i<ir::callable_count(program)){if(!work::charge(meter,1)){return .err(3);}let c=ir::callable_at(program,i);i+=1;
        if(c.foreign && !v::push<Site>(ref mut raw,Site{.start=c.module_index,.end=c.declaration}).is_ok){return .err(2);}
    }return canonical(raw,65536,meter);
}
fn Result<v::Vec<u8>,u32> foreign_nodes(ref ir::Body body,ref v::Vec<Site> targets,ref mut usize meter){
    var flags=v::empty<u8>();var usize i=0;while(i<ir::node_count(body)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut flags,0).is_ok){return .err(2);}i+=1;}
    i=0;while(i<ir::call_count(body)){if(!work::charge(meter,1)){return .err(3);}let c=ir::call_at(body,i);i+=1;let key=Site{.start=c.target_module,.end=c.target_declaration};var usize low=0;var usize high=v::len<Site>(targets);
        while(low<high){if(!work::charge(meter,1)){return .err(3);}let middle=low+(high-low)/2;if(before(*v::at<Site>(targets,middle),key)){low=middle+1;}else{high=middle;}}
        if(low<v::len<Site>(targets)){let value=*v::at<Site>(targets,low);if(value.start==key.start && value.end==key.end){if(c.expression==0 || c.expression>v::len<u8>(ref flags)){return .err(9);}let slot=v::at_mut<u8>(ref mut flags,c.expression-1);*slot=1;}}
    }return .ok(flags);
}
fn bool folded(ref ir::Body body,usize index){let e=ir::node_at(body,index).value;if(e.kind!=6 || e.builtin.kind!=103){return false;}let generic=ir::node(body,e.left);if(generic==0){return false;}let base=ir::node(body,ir::node_at(body,generic-1).value.left);return base!=0 && bytes::equal(ir::node_text(body,base-1),b"take_slot");}
fn Result<v::Vec<Leaf>,u32> inline_leaves(ref ir::Program program,usize maximum,ref mut usize meter){
    var rows=v::empty<Leaf>();var usize frame=1;
    while(frame<=ir::count(program)){if(!work::charge(meter,1)){return .err(3);}let f=clean::frame_at(ir::cleanup_ir(program),frame);let body=ir::at(program,frame);frame+=1;if(f.kind!=2){continue;}var bool witnessed=false;var usize i=0;
        while(i<ir::node_count(body)){if(!work::charge(meter,1)){return .err(3);}if(folded(body,i)){witnessed=true;}i+=1;}if(!witnessed){continue;}
        var bool present=false;i=0;while(i<v::len<Leaf>(ref rows)){if(!work::charge(meter,1)){return .err(3);}if(v::at<Leaf>(ref rows,i).module_index==f.module_index){present=true;}i+=1;}if(present){continue;}
        let unit=sdk::find(ir::sdk_units(program),f.module_index);if(unit==0 || sdk::at(ir::sdk_units(program),unit-1).sdk_id!=8){return .err(9);}let raw=sdk::source_at(ir::sdk_units(program),unit-1);let needle=b"storage_take<T>(slot)";var usize found=0;var Site site=Site{.start=0,.end=0};i=0;
        while(i<=raw.len && needle.len<=raw.len-i){if(!work::charge(meter,needle.len+1)){return .err(3);}if(bytes::equal(raw[i..i+needle.len],needle)){site=Site{.start=i,.end=i+needle.len};found+=1;}i+=1;}
        if(found!=1){return .err(9);}if(maximum==0){return .err(3);}if(!v::push<Leaf>(ref mut rows,Leaf{.module_index=f.module_index,.site}).is_ok){return .err(2);}
    }return .ok(rows);
}
pub fn Result<v::Vec<Site>,u32> collect(ref ir::Program program,usize frame,usize maximum,ref mut usize meter){
    if(frame==0 || frame>ir::count(program) || maximum>65536){return .err(9);}if(!work::charge(meter,1)){return .err(3);}
    let graph=switch(lower::lower(program,frame)){case .ok(p):{p}case .err(e):{return .err(failure(e));}};
    let targets=foreign_targets(program,meter)?;return visit(program,frame,ref graph,ref targets,maximum,meter);
}
pub fn Result<Collection,u32> collect_all(ref ir::Program program,usize maximum,ref mut usize meter){
    if(maximum>65536){return .err(9);}if(!work::charge(meter,1)){return .err(3);}
    let graphs=switch(lower::lower_all(program)){case .ok(p):{p}case .err(e):{return .err(failure(e));}};
    if(v::len<cfg::Graph>(ref graphs)!=ir::count(program)){return .err(9);}let targets=foreign_targets(program,meter)?;
    let leaves=inline_leaves(program,maximum,meter)?;
    var sites=v::empty<Site>();var frames=v::empty<Range>();var usize i=0;
    while(i<v::len<cfg::Graph>(ref graphs)){
        let rows=visit(program,i+1,v::at<cfg::Graph>(ref graphs,i),ref targets,maximum,meter)?;let start=v::len<Site>(ref sites);var usize j=0;
        while(j<v::len<Site>(ref rows)){if(!work::charge(meter,1) || v::len<Site>(ref sites)+v::len<Leaf>(ref leaves)>=65536){return .err(3);}if(!v::push<Site>(ref mut sites,*v::at<Site>(ref rows,j)).is_ok){return .err(2);}j+=1;}
        if(!work::charge(meter,1)){return .err(3);}if(!v::push<Range>(ref mut frames,Range{.start,.end=v::len<Site>(ref sites)}).is_ok){return .err(2);}i+=1;
    }return .ok(Collection{.sites,.frames,.leaves});
}
fn Result<v::Vec<Site>,u32> visit(ref ir::Program program,usize frame,ref cfg::Graph graph,ref v::Vec<Site> targets,usize maximum,ref mut usize meter){
    if(!work::charge(meter,cfg::work(graph))){return .err(3);}let body=ir::at(program,frame);let flags=foreign_nodes(body,targets,meter)?;
    var seen=v::empty<u8>();var usize i=0;while(i<=cfg::count(graph)){if(!work::charge(meter,1)){return .err(3);}if(!v::push<u8>(ref mut seen,0).is_ok){return .err(2);}i+=1;}
    var pending=v::empty<usize>();discard schedule(ref mut pending,ref mut seen,cfg::entry(graph),meter)?;i=0;var rows=v::empty<Site>();
    while(i<v::len<usize>(ref pending)){
        if(!work::charge(meter,1)){return .err(3);}let n=cfg::at(graph,*v::at<usize>(ref pending,i));i+=1;
        if(n.kind==8){let e=ir::node_at(body,n.source-1).value;if(e.kind==1 && bytes::equal(ir::node_text(body,n.source-1),b"false")){continue;}}
        if(n.kind==1){let e=ir::node_at(body,n.source-1).value;var bool observed=e.kind==23 && clean::type_at(ir::cleanup_ir(program),e.type).kind==12;
            if(e.kind==6){let code=e.builtin.kind;observed=code>=9 && code<=12 || code>=103 && code<=107 && !folded(body,n.source-1) || code==0 && *v::at<u8>(ref flags,n.source-1)!=0;}
            if(observed){if(v::len<Site>(ref rows)>=65536){return .err(3);}if(!v::push<Site>(ref mut rows,Site{.start=e.start,.end=e.end}).is_ok){return .err(2);}}
        }
        if(n.first!=0){discard schedule(ref mut pending,ref mut seen,n.first,meter)?;}if(n.second!=0){discard schedule(ref mut pending,ref mut seen,n.second,meter)?;}
    }return canonical(rows,maximum,meter);
}
fn Result<bool,u32> schedule(ref mut v::Vec<usize> pending,ref mut v::Vec<u8> seen,usize id,ref mut usize meter){
    if(!work::charge(meter,1)){return .err(3);}if(id==0 || id>=v::len<u8>(seen)){return .err(9);}if(*v::at<u8>(seen,id)!=0){return .ok(false);}
    if(!v::push<usize>(pending,id).is_ok){return .err(2);}{let slot=v::at_mut<u8>(seen,id);*slot=1;}return .ok(true);
}
