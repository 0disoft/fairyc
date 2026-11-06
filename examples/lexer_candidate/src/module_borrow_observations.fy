module module_borrow_observations;
import module_unit_borrow_sites as sites;
import candidate_limits as limits;
import project_path as budget;
import std::vec as v;

// A missing table differs from an observed frame containing zero borrows.
// Builders own separate local arrays until all frame observations are sealed.
pub resource Table{v::Vec<sites::Site> sites;v::Vec<sites::Range> frames;bool ready;cleanup{}}
pub fn Table absent(){return Table{.sites=v::empty<sites::Site>(),.frames=v::empty<sites::Range>(),.ready=false};}
pub fn bool ready(ref Table p){return p.ready;}
pub fn usize count(ref Table p){return v::len<sites::Range>(ref p.frames);}
pub fn usize site_count(ref Table p,usize frame){assert(p.ready && frame>0 && frame<=count(p));let r=*v::at<sites::Range>(ref p.frames,frame-1);return r.end-r.start;}
pub fn sites::Site at(ref Table p,usize frame,usize index){assert(p.ready && frame>0 && frame<=count(p));let r=*v::at<sites::Range>(ref p.frames,frame-1);assert(index<r.end-r.start);return *v::at<sites::Site>(ref p.sites,r.start+index);}
pub fn Result<bool,u32> initialize(usize frames,ref mut v::Vec<sites::Range> ranges,ref mut v::Vec<u8> seen,ref mut usize work){
    if(v::len<sites::Range>(ranges)!=0 || v::len<u8>(seen)!=0){return .err(9);}if(frames>limits::project_frames() || !budget::charge(work,1)){return .err(3);}var usize i=0;
    while(i<frames){if(!budget::charge(work,1)){return .err(3);}if(!v::push<sites::Range>(ranges,sites::Range{.start=0,.end=0}).is_ok || !v::push<u8>(seen,0).is_ok){return .err(2);}i+=1;}return .ok(true);
}
fn bool same(sites::Site a,sites::Site b){return a.start==b.start && a.end==b.end && a.mutable==b.mutable && a.escapes==b.escapes;}
fn Result<bool,u32> row(ref mut v::Vec<sites::Site> rows,ref mut v::Vec<sites::Range> ranges,ref mut v::Vec<u8> seen,usize frame,ref v::Vec<sites::Site> input,ref mut usize work){
    if(frame==0 || frame>v::len<sites::Range>(ranges) || v::len<sites::Range>(ranges)!=v::len<u8>(seen)){return .err(9);}let old=*v::at<sites::Range>(ranges,frame-1);let used=*v::at<u8>(seen,frame-1)!=0;let n=v::len<sites::Site>(input);
    if(used && old.end-old.start!=n){return .err(9);}let start=v::len<sites::Site>(rows);var usize i=0;
    while(i<n){if(!budget::charge(work,1)){return .err(3);}let value=*v::at<sites::Site>(input,i);
        if(used){if(!same(value,*v::at<sites::Site>(rows,old.start+i))){return .err(9);}}else{if(!v::push<sites::Site>(rows,value).is_ok){return .err(2);}}i+=1;
    }if(!used){let target=v::at_mut<sites::Range>(ranges,frame-1);*target=sites::Range{.start,.end=v::len<sites::Site>(rows)};let mark=v::at_mut<u8>(seen,frame-1);*mark=1;}return .ok(true);
}
pub fn Result<bool,u32> add_sites(ref mut v::Vec<sites::Site> rows,ref mut v::Vec<sites::Range> ranges,ref mut v::Vec<u8> seen,usize frame,ref sites::Collection source,usize source_frame,ref mut usize work){
    let values=sites::collect(source,source_frame,65536,work)?;return row(rows,ranges,seen,frame,ref values,work);
}
pub fn Result<bool,u32> add_table(ref mut v::Vec<sites::Site> rows,ref mut v::Vec<sites::Range> ranges,ref mut v::Vec<u8> seen,usize frame,ref Table source,usize source_frame,ref mut usize work){
    if(!source.ready || source_frame==0 || source_frame>count(source)){return .err(9);}if(!budget::charge(work,1)){return .err(3);}var values=v::empty<sites::Site>();var usize i=0;
    while(i<site_count(source,source_frame)){if(!budget::charge(work,1)){return .err(3);}if(!v::push<sites::Site>(ref mut values,at(source,source_frame,i)).is_ok){return .err(2);}i+=1;}return row(rows,ranges,seen,frame,ref values,work);
}
pub fn Result<Table,u32> seal(v::Vec<sites::Site> rows,v::Vec<sites::Range> ranges,ref v::Vec<u8> seen,ref mut usize work){
    if(v::len<sites::Range>(ref ranges)!=v::len<u8>(seen)){return .err(9);}var usize i=0;
    while(i<v::len<sites::Range>(ref ranges)){if(!budget::charge(work,1)){return .err(3);}let r=*v::at<sites::Range>(ref ranges,i);if(*v::at<u8>(seen,i)==0 || r.start>r.end || r.end>v::len<sites::Site>(ref rows)){return .err(9);}i+=1;}
    return .ok(Table{.sites=rows,.frames=ranges,.ready=true});
}
pub fn Result<Table,u32> copy(ref Table p,ref mut usize work){
    if(!p.ready){return .err(9);}var rows=v::empty<sites::Site>();var ranges=v::empty<sites::Range>();var seen=v::empty<u8>();discard initialize(count(p),ref mut ranges,ref mut seen,work)?;var usize i=1;
    while(i<=count(p)){discard add_table(ref mut rows,ref mut ranges,ref mut seen,i,p,i,work)?;i+=1;}return seal(rows,ranges,ref seen,work);
}
pub fn Result<Table,u32> from_sites(ref sites::Collection p,ref mut usize work){
    var rows=v::empty<sites::Site>();var ranges=v::empty<sites::Range>();var seen=v::empty<u8>();discard initialize(sites::frame_count(p),ref mut ranges,ref mut seen,work)?;var usize i=1;
    while(i<=sites::frame_count(p)){discard add_sites(ref mut rows,ref mut ranges,ref mut seen,i,p,i,work)?;i+=1;}return seal(rows,ranges,ref seen,work);
}
