module module_unit_borrow_sites;
import module_unit_origin_returns as origins;
import module_unit_origin_inputs as inputs;
import module_unit_values as values;
import module_unit_value_model as m;
import syntax as syntax;
import project_path as budget;
import std::vec as v;

// Detached semantic observations, not syntactic borrow expressions or CFG loans.
// Retain multiplicity: different semantic origins may share one source span.
pub struct Site{usize start;usize end;bool mutable;bool escapes;}
pub struct Range{usize start;usize end;}
pub resource Collection{v::Vec<Site> sites;v::Vec<Range> frames;cleanup{}}
pub fn usize frame_count(ref Collection p){return v::len<Range>(ref p.frames);}
pub fn Range range(ref Collection p,usize index){return *v::at<Range>(ref p.frames,index);}
pub fn Site at(ref Collection p,usize index){return *v::at<Site>(ref p.sites,index);}
fn bool before(Site a,Site b){return a.start<b.start || a.start==b.start && (a.end<b.end || a.end==b.end && (a.mutable && !b.mutable || a.mutable==b.mutable && !a.escapes && b.escapes));}
fn m::Problem sorted(ref mut v::Vec<Site> rows,usize base,m::Frame frame,ref mut usize work){
    let n=v::len<Site>(rows)-base;var spare=v::empty<Site>();var usize i=0;
    while(i<n){if(!m::charge(work,1)){return m::fail(frame,10,0,0,0,0);}if(!v::push<Site>(ref mut spare,Site{.start=0,.end=0,.mutable=false,.escapes=false}).is_ok){return m::fail(frame,11,0,0,0,0);}i+=1;}
    var usize width=1;while(width<n){var usize start=0;while(start<n){let middle=if(start+width<n){start+width}else{n};let end=if(middle+width<n){middle+width}else{n};var usize left=start;var usize right=middle;var usize out=start;
        while(out<end){if(!m::charge(work,1)){return m::fail(frame,10,0,0,0,0);}var Site selected=Site{.start=0,.end=0,.mutable=false,.escapes=false};
            if(left<middle && (right==end || !before(*v::at<Site>(rows,base+right),*v::at<Site>(rows,base+left)))){selected=*v::at<Site>(rows,base+left);left+=1;}else{selected=*v::at<Site>(rows,base+right);right+=1;}
            {let slot=v::at_mut<Site>(ref mut spare,out);*slot=selected;}out+=1;
        }start=end;
    }i=0;while(i<n){if(!m::charge(work,1)){return m::fail(frame,10,0,0,0,0);}let row=*v::at<Site>(ref spare,i);{let slot=v::at_mut<Site>(rows,base+i);*slot=row;}i+=1;}width*=2;}
    return m::fail(frame,0,0,0,0,0);
}
// Primary contexts are the exact prefix of the origin plan. Provider-only
// inference frames do not become duplicate observations in the checked unit.
pub fn Result<Collection,m::Problem> capture(ref v::Vec<syntax::Parsed> sources,ref values::Plan primary,ref origins::Plan inferred){
    let input=origins::owned_inputs(inferred);var sites=v::empty<Site>();var frames=v::empty<Range>();var usize work=0;var usize i=0;
    while(i<values::count(primary)){let f=m::frame(values::at(primary,i));if(!m::charge(ref mut work,1)){return .err(m::fail(f,10,0,0,0,0));}
        if(i>=inputs::frame_count(input)){return .err(m::fail(f,12,0,0,0,0));}let owner=inputs::frame(input,i+1);
        if(owner.unit!=0 || owner.context!=i || owner.value.kind!=f.kind || owner.value.context!=f.context || owner.value.module_index!=f.module_index || owner.value.declaration!=f.declaration || f.module_index>=v::len<syntax::Parsed>(sources)){return .err(m::fail(f,12,0,0,0,0));}
        let source=v::at<syntax::Parsed>(sources,f.module_index);if(f.declaration==0 || f.declaration>syntax::declaration_count(source)){return .err(m::fail(f,12,0,0,0,0));}let declaration=syntax::declaration_at(source,f.declaration-1);let start=v::len<Site>(ref sites);var usize j=0;
        while(j<origins::borrow_fact_count(inferred,i+1)){let r=origins::borrow_fact_at(inferred,i+1,j);if(!m::charge(ref mut work,1) || v::len<Site>(ref sites)>=16384){return .err(m::fail(f,10,r.start,r.end,0,0));}
            if(r.frame!=i+1 || r.start<declaration.start || r.end>declaration.end || r.start>=r.end){return .err(m::fail(f,12,r.start,r.end,0,0));}
            if(!v::push<Site>(ref mut sites,Site{.start=r.start,.end=r.end,.mutable=r.mutable,.escapes=r.escapes}).is_ok){return .err(m::fail(f,11,r.start,r.end,0,0));}j+=1;
        }
        let ordered=sorted(ref mut sites,start,f,ref mut work);if(ordered.code!=0){return .err(ordered);}
        if(!v::push<Range>(ref mut frames,Range{.start,.end=v::len<Site>(ref sites)}).is_ok){return .err(m::fail(f,11,0,0,0,0));}i+=1;
    }return .ok(Collection{.sites,.frames});
}
pub fn Result<v::Vec<Site>,u32> collect(ref Collection p,usize frame,usize maximum,ref mut usize work){
    if(frame==0 || frame>frame_count(p) || maximum>65536){return .err(9);}if(!budget::charge(work,1)){return .err(3);}let r=range(p,frame-1);
    if(r.end-r.start>maximum){return .err(3);}var rows=v::empty<Site>();var usize i=r.start;
    while(i<r.end){if(!budget::charge(work,1)){return .err(3);}if(!v::push<Site>(ref mut rows,at(p,i)).is_ok){return .err(2);}i+=1;}return .ok(rows);
}
pub fn Result<Collection,u32> collect_all(ref Collection p,usize maximum,ref mut usize work){
    if(maximum>65536){return .err(9);}if(!budget::charge(work,1)){return .err(3);}var sites=v::empty<Site>();var frames=v::empty<Range>();var usize i=0;
    while(i<frame_count(p)){if(!budget::charge(work,1)){return .err(3);}let r=range(p,i);if(r.end-r.start>maximum){return .err(3);}let start=v::len<Site>(ref sites);var usize j=r.start;
        while(j<r.end){if(!budget::charge(work,1) || v::len<Site>(ref sites)>=65536){return .err(3);}if(!v::push<Site>(ref mut sites,at(p,j)).is_ok){return .err(2);}j+=1;}
        if(!v::push<Range>(ref mut frames,Range{.start,.end=v::len<Site>(ref sites)}).is_ok){return .err(2);}i+=1;
    }return .ok(Collection{.sites,.frames});
}
