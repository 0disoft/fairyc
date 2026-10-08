module hir_borrow_cache;
import hir_model as h;
import hir_borrow_paths as paths;
import std::vec as v;

// Owned, analysis-local facts indexed by actual expression/statement IDs.
// Reachability is immutable. Origin results are cached only after return
// summaries have converged; callers retain the existing work/depth limits.
pub struct Slot{u8 expression_reach;u8 statement_reach;u8 origin_state;usize origin;usize origin_depth;}
struct Span { usize start; usize end; }
fn bool charge(ref mut usize work){*work=*work+1;return *work<=1048576;}
fn bool sift(ref mut v::Vec<Span> spans,usize root,usize count,ref mut usize work){
    var usize at=root;
    while(at<count/2){
        if(!charge(work)){return false;}
        var usize child=at*2+1;
        if(child+1<count && v::at<Span>(spans,child).start<v::at<Span>(spans,child+1).start){child+=1;}
        let old=*v::at<Span>(spans,at);let next=*v::at<Span>(spans,child);
        if(old.start>=next.start){return true;}
        {let ref mut Span slot=v::at_mut<Span>(spans,at);*slot=next;}
        {let ref mut Span slot=v::at_mut<Span>(spans,child);*slot=old;}at=child;
    }return true;
}
fn bool sort(ref mut v::Vec<Span> spans,ref mut usize work){
    let usize count=v::len<Span>(spans);var usize at=count/2;
    while(at>0){at-=1;if(!sift(spans,at,count,work)){return false;}}
    at=count;
    while(at>1){
        if(!charge(work)){return false;}at-=1;
        let first=*v::at<Span>(spans,0);let last=*v::at<Span>(spans,at);
        {let ref mut Span slot=v::at_mut<Span>(spans,0);*slot=last;}
        {let ref mut Span slot=v::at_mut<Span>(spans,at);*slot=first;}
        if(!sift(spans,0,at,work)){return false;}
    }
    var usize end=0;at=0;
    while(at<count){if(!charge(work)){return false;}let ref mut Span span=v::at_mut<Span>(spans,at);if(span.end>end){end=span.end;}span.end=end;at+=1;}
    return true;
}
fn bool live(ref v::Vec<Span> spans,usize start,usize end,ref mut usize work){
    var usize low=0;var usize high=v::len<Span>(spans);
    while(low<high){if(!charge(work)){return false;}let usize mid=low+(high-low)/2;if(v::at<Span>(spans,mid).start<=start){low=mid+1;}else{high=mid;}}
    return low==0 || v::at<Span>(spans,low-1).end<end;
}
pub fn Result<v::Vec<Slot>,u32> create(ref h::Program p,ref mut usize work){
    // Index definite-exit suffixes by start and prefix maximum end. A binary
    // lookup answers containment without scanning every block for every node.
    var v::Vec<Span> spans=v::empty<Span>();var usize block=1;
    while(block<=h::block_count(p)){
        if(!charge(work)){return .err(2);}let b=h::block_at(p,block);let usize start=paths::unreachable_start(p,block,work);
        if(*work>1048576){return .err(2);}
        if(start<=b.end && !v::push<Span>(ref mut spans,Span{.start,.end=b.end}).is_ok){return .err(3);}block+=1;
    }
    if(!sort(ref mut spans,work)){return .err(2);}
    let usize count=if(h::expression_count(p)>h::statement_count(p)){h::expression_count(p)}else{h::statement_count(p)};
    var v::Vec<Slot> result=v::empty<Slot>();var usize i=0;
    while(i<=count){
        if(!charge(work)){return .err(2);}var u8 expression_reach=1;var u8 statement_reach=1;
        if(i>0 && i<=h::expression_count(p)){let e=h::expression_at(p,i);expression_reach=if(live(ref spans,e.start,e.end,work)){1}else{2};}
        if(i>0 && i<=h::statement_count(p)){let s=h::statement_at(p,i);statement_reach=if(live(ref spans,s.start,s.end,work)){1}else{2};}
        if(*work>1048576){return .err(2);}
        if(!v::push<Slot>(ref mut result,Slot{.expression_reach,.statement_reach,.origin_state=0,.origin=0,.origin_depth=0}).is_ok){return .err(3);}i+=1;
    }return .ok(result);
}
pub fn bool reachable(ref v::Vec<Slot> cache,usize id,bool statement,ref mut usize work){
    if(!charge(work)){return false;}if(id==0){return true;}
    let old=*v::at<Slot>(cache,id);return if(statement){old.statement_reach==1}else{old.expression_reach==1};
}
