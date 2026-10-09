module module_unit_origin_provenance;
import module_unit_value_model as m;
import std::vec as v;

// Sites annotate origin leaves; they never become semantic origins. Mixed
// summaries retain each leaf's site because caller Region inputs may alias.
// The standalone Vec ledger avoids mutable resource-field borrows. Map IDs
// name kind-1 headers followed by sorted kind-2 owner/site entries.
pub struct Row {bool expired;u32 kind;usize start;usize count;usize owner;usize allocation;}
pub fn v::Vec<Row> empty(){return v::empty<Row>();}
fn Row member(usize owner,usize allocation){return Row{.expired=false,.kind=2,.start=0,.count=0,.owner,.allocation};}
fn bool valid(ref v::Vec<Row> p,usize map){if(map==0 || map>v::len<Row>(p)){return false;}let r=*v::at<Row>(p,map-1);return r.kind==1 && r.start==map && r.start<=v::len<Row>(p) && r.count<=v::len<Row>(p)-r.start;}
fn Result<usize,u32> intern(ref mut v::Vec<Row> p,ref v::Vec<Row> values,ref mut usize work){
    let n=v::len<Row>(values);if(n==0){return .ok(0);}var usize i=0;var usize maps=0;while(i<v::len<Row>(p)){if(!m::charge(work,1)){return .err(10);}let map=*v::at<Row>(p,i);if(!valid(p,i+1)){return .err(12);}maps+=1;if(map.count==n){var usize j=0;var bool equal=true;while(j<n && equal){if(!m::charge(work,1)){return .err(10);}let a=*v::at<Row>(p,map.start+j);let b=*v::at<Row>(values,j);equal=a.owner==b.owner && a.allocation==b.allocation && a.expired==b.expired;j+=1;}if(equal){return .ok(i+1);}}i=map.start+map.count;}
    if(maps>=4096 || n+1>36864-v::len<Row>(p) || !m::charge(work,n+1)){return .err(10);}let start=v::len<Row>(p)+1;if(!v::push<Row>(p,Row{.expired=false,.kind=1,.start,.count=n,.owner=0,.allocation=0}).is_ok){return .err(11);}i=0;while(i<n){if(!v::push<Row>(p,*v::at<Row>(values,i)).is_ok){return .err(11);}i+=1;}return .ok(start);
}
pub fn Result<usize,u32> single(ref mut v::Vec<Row> p,usize owner,usize allocation,ref mut usize work){
    if(owner<3 || owner>16386 || allocation==0 || allocation>16384){return .err(12);}var v::Vec<Row> values=v::empty<Row>();if(!m::charge(work,1)){return .err(10);}if(!v::push<Row>(ref mut values,member(owner,allocation)).is_ok){return .err(11);}return intern(p,ref values,work);
}
pub fn Result<usize,u32> join(ref mut v::Vec<Row> p,usize a,usize b,ref mut usize work){
    if(!m::charge(work,1)){return .err(10);}if(a!=0 && !valid(p,a) || b!=0 && !valid(p,b)){return .err(12);}if(a==b || b==0){return .ok(a);}if(a==0){return .ok(b);}let left=*v::at<Row>(p,a-1);let right=*v::at<Row>(p,b-1);var usize i=0;var usize j=0;var v::Vec<Row> merged=v::empty<Row>();
    while(i<left.count || j<right.count){if(!m::charge(work,1)){return .err(10);}let x=if(i<left.count){*v::at<Row>(p,left.start+i)}else{member(0,0)};let y=if(j<right.count){*v::at<Row>(p,right.start+j)}else{member(0,0)};var next=if(i==left.count){y}else if(j==right.count || x.owner<=y.owner){x}else{y};if(i<left.count && j<right.count && x.owner==y.owner){if(!x.expired && y.expired){next=y;}next.expired=x.expired || y.expired;}if(i<left.count && x.owner==next.owner){i+=1;}if(j<right.count && y.owner==next.owner){j+=1;}if(!v::push<Row>(ref mut merged,next).is_ok){return .err(11);}}
    return intern(p,ref merged,work);
}
pub fn Result<usize,u32> lookup(ref v::Vec<Row> p,usize map,usize owner,ref mut usize work){
    if(!m::charge(work,1)){return .err(10);}if(map==0){return .ok(0);}if(!valid(p,map)){return .err(12);}let entry=*v::at<Row>(p,map-1);var usize low=0;var usize high=entry.count;while(low<high){if(!m::charge(work,1)){return .err(10);}let mid=low+(high-low)/2;let row=*v::at<Row>(p,entry.start+mid);if(row.owner<owner){low=mid+1;}else{high=mid;}}return .ok(if(low<entry.count && v::at<Row>(p,entry.start+low).owner==owner){v::at<Row>(p,entry.start+low).allocation}else{0});
}

// Expiration is path-local immutable provenance, not an origin identity. It
// survives loop re-entry into the same lexical Region and ORs across joins.
pub fn Result<usize,u32> expire(ref mut v::Vec<Row> p,usize map,usize owner,ref mut usize work){
    if(!m::charge(work,1)){return .err(10);}if(map==0){return .ok(0);}if(!valid(p,map)){return .err(12);}let header=*v::at<Row>(p,map-1);var bool changed=false;var v::Vec<Row> rows=v::empty<Row>();var usize i=0;while(i<header.count){if(!m::charge(work,1)){return .err(10);}var row=*v::at<Row>(p,header.start+i);if(row.owner==owner && !row.expired){row.expired=true;changed=true;}if(!v::push<Row>(ref mut rows,row).is_ok){return .err(11);}i+=1;}return if(changed){intern(p,ref rows,work)}else{.ok(map)};
}
pub fn Result<bool,u32> expired(ref v::Vec<Row> p,usize map,usize owner,ref mut usize work){
    if(!m::charge(work,1)){return .err(10);}if(map==0){return .ok(false);}if(!valid(p,map)){return .err(12);}let header=*v::at<Row>(p,map-1);var usize i=0;while(i<header.count){if(!m::charge(work,1)){return .err(10);}let row=*v::at<Row>(p,header.start+i);if(row.owner==owner){return .ok(row.expired);}if(row.owner>owner){return .ok(false);}i+=1;}return .ok(false);
}
