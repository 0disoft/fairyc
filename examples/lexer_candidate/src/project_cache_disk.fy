module project_cache_disk;
import windows_rooted_file as rooted;
import sha256 as blocks;
import std::bytes as bytes;
import std::vec as v;

// This private store transports candidates, never checked compiler owners.
// Every returned buffer still requires project_cache_artifact.restore.
// Mandatory limits cover all recognized generations and unpublished stages.
pub struct Limits{usize entries;usize bytes;usize record;usize temporary_age;}
pub struct Stats{usize entries;usize bytes;usize evicted;}
pub struct Problem{u32 kind;u32 stage;u32 os_code;usize completed;rooted::Metadata observed;}
// 1 policy/name, 2 missing, 3 inventory, 4 read, 5 publication,
// 6 retention, 7 malformed/digest, 8 allocation, 9 work/record bound.
fn rooted::Metadata no_metadata(){return rooted::Metadata{.identity=0,.modified=0,.size=0,.attributes=0};}
fn Problem fail(u32 kind){return Problem{.kind,.stage=0,.os_code=0,.completed=0,.observed=no_metadata()};}
fn Problem malformed(rooted::Metadata observed){return Problem{.kind=7,.stage=0,.os_code=0,.completed=0,.observed};}
fn Problem io(u32 kind,rooted::Problem p){return Problem{.kind,.stage=p.stage,.os_code=p.os_code,.completed=p.completed,.observed=no_metadata()};}
pub fn usize record_limit(){return 268435456;}
pub fn usize work_limit(){return 1073741824;}
pub fn bool valid_limits(Limits p){return p.entries>0 && p.entries<=4096 && p.bytes>0 && p.bytes<=1073741824 && p.record>=8 && p.record<=record_limit() && p.record<=p.bytes && p.temporary_age>0;}
fn bool charge(ref mut usize work,usize cost){if(*work>work_limit() || cost>work_limit()-*work){return false;};*work+=cost;return true;}
fn bool scan_charge(ref mut usize work,usize cost){if(*work>16777216 || cost>16777216-*work){return false;};*work+=cost;return true;}
fn bool hex(ref u8[] p){var usize i=0;while(i<p.len){if(!(p[i]>=48 && p[i]<=57 || p[i]>=97 && p[i]<=102)){return false;}i+=1;}return true;}
pub fn bool valid_route(ref u8[] p){return p.len==81 && bytes::equal(p[..17],b"fylookup0-sha256-") && hex(p[17..]);}
// Generation names bind both the pre-compilation route and complete stored bytes.
fn bool generation_name(ref u8[] p){return p.len==144 && bytes::equal(p[..11],b"fyproject0-") && p[75]==45 && bytes::equal(p[140..],b".fyc") && hex(p[11..75]) && hex(p[76..140]);}
// Exclusive staging names contain only a full generation name and 0..15.
fn bool temporary_name(ref u8[] p){if(p.len<152 || p.len>153 || !bytes::equal(p[..6],b".stage") || p[150]!=45 || !generation_name(p[6..150])){return false;}return if(p.len==152){p[151]>=48 && p[151]<=57}else{p[151]==49 && p[152]>=48 && p[152]<=53};}
pub resource Store{rooted::Directory root;Limits limits;cleanup{}}
// Cursor and candidate are detached owners. The caller keeps Store alive and
// uses them only with that same store. A candidate can still fail IR admission.
pub resource Candidates{rooted::Entries entries;v::Vec<usize> order;bytes::Buffer route;cleanup{}}
pub resource Candidate{bytes::Buffer data;bytes::Buffer name;rooted::Metadata observed;cleanup{}}
pub fn ref u8[] candidate_bytes(ref Candidate p){return bytes::view(ref p.data);}
pub fn usize candidate_count(ref Candidates p){return v::len<usize>(ref p.order);}
@effects(filesystem) pub fn Result<Store,Problem> open(ref u8[] path,Limits limits){if(!valid_limits(limits)){return .err(fail(1));}let root=switch(rooted::open(path)){case .ok(p):{p}case .err(e):{return .err(io(3,e));}};return .ok(Store{.root,.limits});}

// The existing source SHA stream intentionally caps at 96MiB. Stored program
// envelopes have a distinct 256MiB bound; reuse its exact block compression
// without changing any source/compiler identity cap or hash preimage.
fn u32 octet(ref u8[] raw,usize index,usize padded){if(index<raw.len){return raw[index] as u32;}if(index==raw.len){return 128;}if(index<padded-8){return 0;}var usize value=raw.len*8;var usize shift=padded-1-index;while(shift>0){value/=256;shift-=1;}let digits=b"\x00\x01\x02\x03\x04\x05\x06\x07\x08\x09\x0a\x0b\x0c\x0d\x0e\x0f";return (digits[(value/16)%16] as u32)*16+(digits[value%16] as u32);}
pub fn Result<bytes::Buffer,Problem> digest(ref u8[] raw,ref mut usize work){
    if(raw.len>record_limit() || !charge(work,raw.len+(raw.len+72)/64+65)){return .err(fail(9));}
    var u32[8] h=[1779033703,3144134277,1013904242,2773480762,1359893119,2600822924,528734635,1541459225];let padded=((raw.len+72)/64)*64;var usize at=0;
    while(at<padded){var u32[16] words=[0;16];var usize i=0;while(i<16){let p=at+i*4;words[i]=octet(raw,p,padded)<<24|octet(raw,p+1,padded)<<16|octet(raw,p+2,padded)<<8|octet(raw,p+3,padded);i+=1;}h=blocks::compress(h,words);at+=64;}
    var out=bytes::empty();let digits=b"0123456789abcdef";var usize i=0;var u32 shift=28;while(i<64){if(!bytes::append_byte(ref mut out,digits[((h[i/8]>>shift)&15) as usize]).is_ok){return .err(fail(8));}i+=1;if(i%8==0){shift=28;}else{shift-=4;}}return .ok(out);
}
pub fn Result<bytes::Buffer,Problem> generation(ref u8[] route,ref u8[] raw,ref mut usize work){if(!valid_route(route)){return .err(fail(1));}let hash=switch(digest(raw,work)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!charge(work,145)){return .err(fail(9));}var out=bytes::empty();if(!bytes::append(ref mut out,b"fyproject0-").is_ok || !bytes::append(ref mut out,route[17..]).is_ok || !bytes::append_byte(ref mut out,45).is_ok || !bytes::append(ref mut out,bytes::view(ref hash)).is_ok || !bytes::append(ref mut out,b".fyc").is_ok){return .err(fail(8));}return .ok(out);}
fn bool envelope(ref u8[] raw){return raw.len>=8 && bytes::equal(raw[..8],b"FYCACHE0");}
fn void set(ref mut v::Vec<usize> order,usize at,usize value){let slot=v::at_mut<usize>(order,at);*slot=value;}
fn bool before(ref u8[] a,ref u8[] b){var usize i=0;while(i<a.len && i<b.len){if(a[i]!=b[i]){return a[i]<b[i];}i+=1;}return a.len<b.len;}
fn Result<v::Vec<usize>,Problem> ordered(ref rooted::Entries entries,ref mut usize work){var v::Vec<usize> order=v::empty<usize>();var usize i=0;
    while(i<rooted::count(entries)){if(!scan_charge(work,1)){return .err(fail(9));}let name=rooted::name(entries,i);if(rooted::regular(rooted::metadata(entries,i)) && (generation_name(name) || temporary_name(name))){if(!v::push<usize>(ref mut order,i).is_ok){return .err(fail(8));}let current=rooted::metadata(entries,i);var usize at=v::len<usize>(ref order)-1;while(at>0){if(!scan_charge(work,1)){return .err(fail(9));}let previous=*v::at<usize>(ref order,at-1);let other=rooted::metadata(entries,previous);var bool precedes=current.modified<other.modified;if(current.modified==other.modified){let b=rooted::name(entries,previous);if(!scan_charge(work,name.len+b.len+1)){return .err(fail(9));}precedes=before(name,b);}if(!precedes){break;}set(ref mut order,at,previous);at-=1;}set(ref mut order,at,i);}i+=1;
    }return .ok(order);
}
@effects(filesystem) fn Result<bytes::Buffer,Problem> read_generation(ref Store store,ref u8[] name,ref mut usize work,ref mut rooted::Metadata snapshot){
    var file=switch(rooted::open_read(ref store.root,name)){case .ok(p):{p}case .err(e):{return .err(io(if(rooted::missing(e)){2}else{4},e));}};
    let observed=switch(rooted::observe(ref file)){case .ok(p):{p}case .err(e):{return .err(io(4,e));}};let size=observed.size;if(size<8 || size>store.limits.record){return .err(malformed(observed));}if(!charge(work,size+2)){return .err(fail(9));}
    var out=bytes::empty();if(!bytes::resize_zeroed(ref mut out,size).is_ok){return .err(fail(8));}var usize at=0;while(at<size){let usize count=if(size-at>65536){65536}else{size-at};let used=switch(rooted::read(ref mut file,bytes::view_mut(ref mut out)[at..at+count])){case .ok(p):{p}case .err(e):{return .err(io(4,e));}};if(used==0){return .err(malformed(observed));}at+=used;}
    var u8[1] tail=[0;1];let extra=switch(rooted::read(ref mut file,ref mut tail)){case .ok(p):{p}case .err(e):{return .err(io(4,e));}};if(extra!=0){return .err(malformed(observed));}switch(rooted::close(file)){case .ok(_):{}case .err(e):{return .err(io(4,e));}};
    if(!envelope(bytes::view(ref out))){return .err(malformed(observed));}let hash=switch(digest(bytes::view(ref out),work)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!bytes::equal(bytes::view(ref hash),name[76..140])){return .err(malformed(observed));};*snapshot=observed;return .ok(out);
}
// All candidates remain untrusted even after content-address verification.
// Corrupt generations are skipped; exhausted work cannot publish a prefix.
@effects(filesystem) pub fn Result<bytes::Buffer,Problem> load(ref Store store,ref u8[] route,ref mut usize work,ref mut usize scan_work){if(!valid_route(route) || !valid_limits(store.limits)){return .err(fail(1));}if(!charge(work,0)){return .err(fail(9));}let entries=switch(rooted::scan(ref store.root,4096,scan_work)){case .ok(p):{p}case .err(e):{return .err(io(3,e));}};let order=switch(ordered(ref entries,scan_work)){case .ok(p):{p}case .err(e):{return .err(e);}};var usize i=v::len<usize>(ref order);
    while(i>0){i-=1;let name=rooted::name(ref entries,*v::at<usize>(ref order,i));if(generation_name(name) && bytes::equal(name[11..75],route[17..])){var observed=no_metadata();switch(read_generation(store,name,work,ref mut observed)){case .ok(p):{return .ok(p);}case .err(e):{if(e.kind==8 || e.kind==9){return .err(e);}}};}
    }return .err(fail(2));
}
// A single immutable inventory supports bounded newest-to-oldest IR admission.
@effects(filesystem) pub fn Result<Candidates,Problem> candidates(ref Store store,ref u8[] route,ref mut usize scan_work){
    if(!valid_route(route) || !valid_limits(store.limits)){return .err(fail(1));}let entries=switch(rooted::scan(ref store.root,4096,scan_work)){case .ok(p):{p}case .err(e):{return .err(io(3,e));}};let order=switch(ordered(ref entries,scan_work)){case .ok(p):{p}case .err(e):{return .err(e);}};
    if(!scan_charge(scan_work,route.len+1)){return .err(fail(9));}let key=switch(bytes::copy(route)){case .ok(p):{p}case .err(_):{return .err(fail(8));}};return .ok(Candidates{.entries,.order,.route=key});
}
@effects(filesystem) pub fn Result<Candidate,Problem> next(ref Store store,ref Candidates cursor,ref mut usize index,ref mut usize work){
    if(*index>candidate_count(cursor) || !charge(work,0)){return .err(fail(9));}
    while(*index>0){if(!charge(work,1)){return .err(fail(9));};*index-=1;let name=rooted::name(ref cursor.entries,*v::at<usize>(ref cursor.order,*index));if(generation_name(name) && bytes::equal(name[11..75],bytes::view(ref cursor.route)[17..])){
        var observed=no_metadata();switch(read_generation(store,name,work,ref mut observed)){
            case .ok(data):{if(!charge(work,name.len+1)){return .err(fail(9));}let leaf=switch(bytes::copy(name)){case .ok(p):{p}case .err(_):{return .err(fail(8));}};return .ok(Candidate{.data,.name=leaf,.observed});}
            case .err(e):{if(e.kind==8 || e.kind==9){return .err(e);}}
        };
    }}return .err(fail(2));
}
// Rejection drops only the exact file observed during the successful read.
// A later replacement, concurrent writer or reparse leaf is left intact.
@effects(filesystem) pub fn Result<bool,Problem> reject(ref Store store,ref Candidate candidate){return switch(rooted::erase_if(ref store.root,bytes::view(ref candidate.name),candidate.observed)){case .ok(p):{.ok(p)}case .err(e):{.err(io(6,e))}};}
// Successful sweep establishes bounds for this observed retained set. Active
// staged writers deny the delete-enabled open. Concurrent stores may change
// the directory after observation; no lock or global instantaneous bound exists.
@effects(filesystem,clock) pub fn Result<Stats,Problem> sweep(ref Store store,ref u8[] preserve,ref mut usize work){if(!valid_limits(store.limits) || preserve.len!=0 && !generation_name(preserve)){return .err(fail(1));}let entries=switch(rooted::scan(ref store.root,4096,work)){case .ok(p):{p}case .err(e):{return .err(io(3,e));}};let order=switch(ordered(ref entries,work)){case .ok(p):{p}case .err(e):{return .err(e);}};let now=rooted::now();var usize count=0;var usize size=0;var usize evicted=0;var v::Vec<usize> retained=v::empty<usize>();var usize i=0;
    while(i<v::len<usize>(ref order)){let at=*v::at<usize>(ref order,i);let name=rooted::name(ref entries,at);let info=rooted::metadata(ref entries,at);let temporary=temporary_name(name);let old=temporary && info.modified<=now && now-info.modified>=store.limits.temporary_age;let oversize=info.size>store.limits.record;
        if((old || oversize) && !bytes::equal(name,preserve)){switch(rooted::erase_if(ref store.root,name,info)){case .ok(done):{if(done){evicted+=1;i+=1;continue;}}case .err(_):{}};}
        if(oversize){return .err(fail(6));}if(!v::push<usize>(ref mut retained,at).is_ok){return .err(fail(8));}count+=1;size+=info.size;i+=1;
    }
    i=0;while((count>store.limits.entries || size>store.limits.bytes) && i<v::len<usize>(ref retained)){let at=*v::at<usize>(ref retained,i);let name=rooted::name(ref entries,at);let info=rooted::metadata(ref entries,at);i+=1;if(bytes::equal(name,preserve) || temporary_name(name) && (info.modified>now || now-info.modified<store.limits.temporary_age)){continue;}switch(rooted::erase_if(ref store.root,name,info)){case .ok(done):{if(done){count-=1;size-=info.size;evicted+=1;}}case .err(_):{}};}
    if(count>store.limits.entries || size>store.limits.bytes){return .err(fail(6));}return .ok(Stats{.entries=count,.bytes=size,.evicted});
}
@effects(filesystem) fn Result<rooted::File,Problem> stage(ref Store store,ref u8[] generation,ref mut usize work){var usize attempt=0;while(attempt<16){if(!charge(work,160)){return .err(fail(9));}var name=bytes::empty();if(!bytes::append(ref mut name,b".stage").is_ok || !bytes::append(ref mut name,generation).is_ok || !bytes::append_byte(ref mut name,45).is_ok || !bytes::append_usize(ref mut name,attempt).is_ok){return .err(fail(8));}switch(rooted::create_new(ref store.root,bytes::view(ref name))){case .ok(p):{return .ok(p);}case .err(e):{if(!rooted::collision(e)){return .err(io(5,e));}}};attempt+=1;}return .err(fail(5));}
// No-replace publication makes digest-named generations immutable. Collisions
// succeed after exact existing-byte comparison. A corrupt collision can be
// removed only against metadata retained by that failed read; publication then
// retries without replacement. A concurrently changed object stays intact.
@effects(filesystem,clock) pub fn Result<Stats,Problem> put(ref Store store,ref u8[] route,ref u8[] raw,ref mut usize work,ref mut usize scan_work){if(!valid_limits(store.limits) || !valid_route(route)){return .err(fail(1));}if(raw.len>store.limits.record || raw.len>store.limits.bytes){return .err(fail(9));}if(!envelope(raw)){return .err(fail(7));}let name=switch(generation(route,raw,work)){case .ok(p):{p}case .err(e):{return .err(e);}};var file=switch(stage(store,bytes::view(ref name),work)){case .ok(p):{p}case .err(e):{return .err(e);}};if(!charge(work,raw.len+2)){return .err(fail(9));}switch(rooted::write_all(ref mut file,raw)){case .ok(_):{}case .err(e):{return .err(io(5,e));}};switch(rooted::sync(ref mut file)){case .ok(_):{}case .err(e):{return .err(io(5,e));}};
    switch(rooted::publish_new(ref store.root,ref mut file,bytes::view(ref name))){
        case .ok(_):{switch(rooted::sync(ref mut file)){case .ok(_):{}case .err(e):{return .err(io(5,e));}};}
        case .err(e):{
            if(e.os_code!=80 && e.os_code!=183){return .err(io(5,e));}
            var snapshot=no_metadata();switch(read_generation(store,bytes::view(ref name),work,ref mut snapshot)){
                case .ok(old):{if(!charge(work,raw.len+1)){return .err(fail(9));}if(!bytes::equal(bytes::view(ref old),raw)){return .err(fail(7));}}
                case .err(e):{
                    if(e.kind!=7){return .err(e);}let removed=switch(rooted::erase_if(ref store.root,bytes::view(ref name),e.observed)){case .ok(p):{p}case .err(e):{return .err(io(5,e));}};if(!removed){return .err(e);}
                    switch(rooted::publish_new(ref store.root,ref mut file,bytes::view(ref name))){case .ok(_):{}case .err(e):{return .err(io(5,e));}};
                    switch(rooted::sync(ref mut file)){case .ok(_):{}case .err(e):{return .err(io(5,e));}};
                }
            };
        }
    };
    switch(rooted::close(file)){case .ok(_):{}case .err(e):{return .err(io(5,e));}};return sweep(store,bytes::view(ref name),scan_work);
}
