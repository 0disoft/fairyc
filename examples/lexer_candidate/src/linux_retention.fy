module linux_retention;
import linux_rooted_file as rooted;
import linux_directory as directory;
import linux_command as command;
import std::bytes as bytes;
import std::vec as v;

@effects(filesystem) @link_name("fairyc_private_linux_root_file_erase_if_v1") unsafe extern C fn u32 native_erase(u32 root,ptr u8 leaf,usize size,ptr void expected,ptr mut u32 removed,ptr mut u32 close_code);
pub struct Limits { usize entries; usize bytes; usize record; }
pub struct Stats { usize entries; usize bytes; usize evicted; }
// 1 input, 2 scan, 3 erase/close, 4 stale snapshot, 5 work, 6 allocation, 7 protected oversize.
pub struct Problem { u32 kind; u32 os_code; u32 close_code; usize evicted; }
// 1 input, 2 allocation, 3 native, 4 close, 5 ABI. removed preserves applied mutation.
pub struct EraseProblem { u32 kind; u32 os_code; u32 close_code; bool removed; }
fn Problem fail(u32 kind,usize evicted) { return Problem{.kind,.os_code=0,.close_code=0,.evicted}; }
fn EraseProblem erase_fail(u32 kind,u32 os_code,u32 close_code,bool removed) { return EraseProblem{.kind,.os_code,.close_code,.removed}; }
pub fn bool valid_limits(Limits p) { return p.entries>0 && p.entries<=4096 && p.bytes>=8 && p.bytes<=1073741824 && p.record>=8 && p.record<=268435456 && p.record<=p.bytes; }
pub fn bool valid_generation(ref u8[] leaf) {
    if(leaf.len!=144 || !bytes::equal(leaf[..11],b"fyproject0-") || leaf[75]!=45 || !bytes::equal(leaf[140..],b".fyc")) { return false; }
    var usize at=11;while(at<140) { if(at!=75 && !(leaf[at]>=48 && leaf[at]<=57 || leaf[at]>=97 && leaf[at]<=102)) { return false; } at+=1; }return true;
}
fn bool charge(ref mut usize work,usize cost) { if(*work>directory::work_limit() || cost>directory::work_limit()-*work) { return false; };*work+=cost;return true; }

@effects(filesystem) pub fn Result<bool,EraseProblem> erase_if(ref rooted::Directory root,ref u8[] leaf,rooted::Metadata expected) {
    if(!valid_generation(leaf) || !rooted::regular(expected) || expected.identity==0 || expected.modified_nanos>=1000000000) { return .err(erase_fail(1,0,0,false)); }
    let encoded=switch(command::path(leaf)) { case .ok(p): { p } case .err(e): { return .err(erase_fail(if(e.kind==3){2}else{1},0,0,false)); } };
    var u32 removed=0;var u32 closed=0;
    let code=unsafe { native_erase(rooted::directory_handle(root),bytes::view(ref encoded) as ptr u8,bytes::len(ref encoded),ref expected as ptr void,ref mut removed as ptr mut u32,ref mut closed as ptr mut u32) };
    if(removed>1) { return .err(erase_fail(5,code,closed,false)); }
    if(code!=0 || closed!=0) { return .err(erase_fail(if(code!=0){3}else{4},code,closed,removed==1)); }return .ok(removed==1);
}
fn void set(ref mut v::Vec<usize> order,usize at,usize value) { let slot=v::at_mut<usize>(order,at);*slot=value; }
fn void swap(ref mut v::Vec<usize> order,usize a,usize b) { let x=*v::at<usize>(order,a);let y=*v::at<usize>(order,b);set(order,a,y);set(order,b,x); }
fn Result<bool,Problem> before(ref directory::Entries entries,usize a,usize b,ref mut usize work) {
    if(!charge(work,1)) { return .err(fail(5,0)); }
    let x=directory::metadata(entries,a);let y=directory::metadata(entries,b);
    if(x.modified_seconds!=y.modified_seconds) { return .ok(x.modified_seconds<y.modified_seconds); }
    if(x.modified_nanos!=y.modified_nanos) { return .ok(x.modified_nanos<y.modified_nanos); }
    let left=directory::name(entries,a);let right=directory::name(entries,b);var usize at=0;
    while(at<left.len && at<right.len) { if(!charge(work,1)) { return .err(fail(5,0)); }if(left[at]!=right[at]) { return .ok(left[at]<right[at]); }at+=1; }
    return .ok(if(left.len!=right.len){left.len<right.len}else{a<b});
}
fn Result<bool,Problem> sift(ref directory::Entries entries,ref mut v::Vec<usize> order,usize start,usize length,ref mut usize work) {
    var usize at=start;while(at<length/2) {
        var usize child=at*2+1;
        if(child+1<length) { let right=switch(before(entries,*v::at<usize>(order,child),*v::at<usize>(order,child+1),work)) { case .ok(p): { p } case .err(e): { return .err(e); } };if(right) { child+=1; } }
        let earlier=switch(before(entries,*v::at<usize>(order,at),*v::at<usize>(order,child),work)) { case .ok(p): { p } case .err(e): { return .err(e); } };
        if(!earlier) { return .ok(true); }swap(order,at,child);at=child;
    }return .ok(true);
}
fn Result<bool,Problem> sort(ref directory::Entries entries,ref mut v::Vec<usize> order,ref mut usize work) {
    let n=v::len<usize>(order);var usize at=n/2;
    while(at>0) { at-=1;switch(sift(entries,order,at,n,work)) { case .ok(_): {} case .err(e): { return .err(e); } }; }
    var usize end=n;while(end>1) { end-=1;swap(order,0,end);switch(sift(entries,order,0,end,work)) { case .ok(_): {} case .err(e): { return .err(e); } }; }return .ok(true);
}
// Results describe this observed inventory only. Concurrent publication can add entries.
@effects(filesystem) pub fn Result<Stats,Problem> sweep(ref rooted::Directory root,Limits limits,ref u8[] preserve,ref mut usize work) {
    if(!valid_limits(limits) || preserve.len!=0 && !valid_generation(preserve)) { return .err(fail(1,0)); }
    let entries=switch(directory::scan(root,4096,work)) { case .ok(p): { p } case .err(e): { return .err(Problem{.kind=2,.os_code=e.os_code,.close_code=e.close_code,.evicted=0}); } };
    var v::Vec<usize> order=v::empty<usize>();var usize at=0;var usize retained=0;var usize size=0;
    while(at<directory::count(ref entries)) {
        let leaf=directory::name(ref entries,at);if(!charge(work,leaf.len+1)) { return .err(fail(5,0)); }
        let info=directory::metadata(ref entries,at);
        if(rooted::regular(info) && valid_generation(leaf)) {
            if(info.size>limits.record && bytes::equal(leaf,preserve)) { return .err(fail(7,0)); }
            if(!v::push<usize>(ref mut order,at).is_ok) { return .err(fail(6,0)); }
            if(info.size<=limits.record) { retained+=1;size+=info.size; }
        }at+=1;
    }
    switch(sort(ref entries,ref mut order,work)) { case .ok(_): {} case .err(e): { return .err(e); } };
    var usize evicted=0;at=0;
    while(at<v::len<usize>(ref order)) {
        let index=*v::at<usize>(ref order,at);at+=1;let info=directory::metadata(ref entries,index);let leaf=directory::name(ref entries,index);
        let oversize=info.size>limits.record;
        if((oversize || retained>limits.entries || size>limits.bytes) && !bytes::equal(leaf,preserve)) {
            if(!charge(work,leaf.len+41)) { return .err(fail(5,evicted)); }
            switch(erase_if(root,leaf,info)) {
                case .ok(removed): { if(!removed) { return .err(fail(4,evicted)); }evicted+=1;if(!oversize) { retained-=1;size-=info.size; } }
                case .err(e): { let usize applied=if(e.removed){1}else{0};return .err(Problem{.kind=3,.os_code=e.os_code,.close_code=e.close_code,.evicted=evicted+applied}); }
            };
        }
    }if(retained>limits.entries || size>limits.bytes) { return .err(fail(7,evicted)); }return .ok(Stats{.entries=retained,.bytes=size,.evicted});
}
