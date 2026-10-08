module hir_borrow_access;
import hir_model as h;
import borrow_origins as owners;
import std::vec as v;

struct Link { usize expression; usize next; }
pub resource Index { v::Vec<usize> heads; v::Vec<Link> links; cleanup { } }
fn bool charge(ref mut usize work){*work=*work+1;return *work<=1048576;}
fn u32 insert(ref mut v::Vec<usize> heads,ref mut v::Vec<Link> links,usize owner,usize expression,ref mut usize work){
    if(owner==0 || owner>1152){return 4;}if(!charge(work) || v::len<Link>(links)>32768){return 2;}
    let usize at=v::len<Link>(links);
    if(!v::push<Link>(links,Link{.expression,.next=*v::at<usize>(heads,owner)}).is_ok){return 3;}
    {let ref mut usize head=v::at_mut<usize>(heads,owner);*head=at;}return 0;
}
pub fn Result<Index,u32> create(ref h::Program p,ref v::Vec<usize> origins,ref v::Vec<owners::Entry> sets,ref v::Vec<usize> members,ref mut usize work){
    var v::Vec<usize> heads=v::empty<usize>();var usize head=0;while(head<1153){if(!charge(work)){return .err(2);}if(!v::push<usize>(ref mut heads,0).is_ok){return .err(3);}head+=1;}var v::Vec<Link> links=v::empty<Link>();
    if(!v::push<Link>(ref mut links,Link{.expression=0,.next=0}).is_ok){return .err(3);}
    // Descending insertion leaves each owner's expression IDs ascending. Index
    // both whole-place references and every owner of mixed descriptors; Static
    // and Unknown contribute no descriptor owner.
    var usize id=h::expression_count(p);
    while(id>0){
        if(!charge(work)){return .err(2);}let e=h::expression_at(p,id);
        let usize binding=if(h::expression_code(e.kind)==3){e.binding}else{0};
        if(binding!=0){let u32 code=insert(ref mut heads,ref mut links,binding,id,work);if(code!=0){return .err(code);}}
        let usize origin=*v::at<usize>(origins,id);let usize count=owners::count(sets,origin);var usize i=0;
        while(i<count){let usize owner=owners::at(sets,members,origin,i);if(owner!=binding){let u32 code=insert(ref mut heads,ref mut links,owner,id,work);if(code!=0){return .err(code);}}i+=1;}
        id-=1;
    }return .ok(Index{.heads,.links});
}
pub fn u32 begin(ref Index index,ref v::Vec<owners::Entry> sets,ref v::Vec<usize> members,usize origin,ref mut v::Vec<usize> cursor,ref mut usize work){
    discard v::clear<usize>(cursor);let usize count=owners::count(sets,origin);var usize i=0;
    while(i<count){if(!charge(work)){return 2;}let usize head=*v::at<usize>(ref index.heads,owners::at(sets,members,origin,i));if(!v::push<usize>(cursor,head).is_ok){return 3;}i+=1;}return 0;
}
pub fn usize next(ref Index index,ref mut v::Vec<usize> cursor,ref mut usize work){
    var usize result=0;var usize i=0;
    while(i<v::len<usize>(cursor)){
        if(!charge(work)){return 0;}let usize link=*v::at<usize>(cursor,i);
        if(link!=0){let usize id=v::at<Link>(ref index.links,link).expression;if(result==0 || id<result){result=id;}}i+=1;
    }
    if(result==0){return 0;}i=0;
    // Merge owner lists without duplicate checks or changing first-error order.
    while(i<v::len<usize>(cursor)){
        if(!charge(work)){return 0;}let usize link=*v::at<usize>(cursor,i);
        if(link!=0){let item=*v::at<Link>(ref index.links,link);if(item.expression==result){let ref mut usize slot=v::at_mut<usize>(cursor,i);*slot=item.next;}}i+=1;
    }return result;
}
