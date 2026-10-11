module project_key_index;
import project_manifest_values as rules;
import project_path as work;
import std::bytes as bytes;
import std::vec as v;

pub struct Span{usize start;usize end;}
pub resource Index{bytes::Buffer keys;v::Vec<Span> spans;v::Vec<usize> order;cleanup{}}
pub fn usize count(ref Index index){return v::len<Span>(ref index.spans);}
pub fn ref u8[] key(ref Index index,usize at){assert(at<count(index));let Span span=*v::at<Span>(ref index.spans,at);return bytes::view(ref index.keys)[span.start..span.end];}
pub fn bool append(ref mut bytes::Buffer names,ref mut v::Vec<Span> spans,ref u8[] value){if(v::len<Span>(spans)>=1024 || value.len>1048576){return false;}let usize start=bytes::len(names);if(!bytes::append(names,value).is_ok){return false;}if(!v::push<Span>(spans,Span{.start,.end=bytes::len(names)}).is_ok){discard bytes::truncate(names,start);return false;}return true;}
fn ref u8[] view(ref bytes::Buffer names,ref v::Vec<Span> spans,usize at){let Span span=*v::at<Span>(spans,at);return bytes::view(names)[span.start..span.end];}
fn void put(ref mut v::Vec<usize> values,usize at,usize value){let ref mut usize slot=v::at_mut<usize>(values,at);*slot=value;}
// 1 allocation/invalid state, 2 exhausted shared work budget.
pub fn Result<Index,u32> build(bytes::Buffer names,v::Vec<Span> spans,ref mut usize meter){
    let usize size=v::len<Span>(ref spans);if(size>1024){return .err(1);}var v::Vec<usize> order=v::empty<usize>();var v::Vec<usize> scratch=v::empty<usize>();var usize at=0;
    while(at<size){let Span span=*v::at<Span>(ref spans,at);if(span.start>span.end || span.end>bytes::len(ref names) || span.end-span.start>1048576){return .err(1);}if(!v::push<usize>(ref mut order,at).is_ok || !v::push<usize>(ref mut scratch,0).is_ok){return .err(1);}at+=1;}
    var usize width=1;while(width<size){var usize start=0;while(start<size){let usize middle=if(width<size-start){start+width}else{size};let usize end=if(width<size-middle){middle+width}else{size};var usize left=start;var usize right=middle;var usize out=start;
        while(out<end){var bool take_left=right==end;if(left<middle && right<end){let ref u8[] a=view(ref names,ref spans,*v::at<usize>(ref order,left));let ref u8[] b=view(ref names,ref spans,*v::at<usize>(ref order,right));if(!work::charge(meter,a.len+b.len+1)){return .err(2);}take_left=!rules::before(b,a);}put(ref mut scratch,out,if(take_left){*v::at<usize>(ref order,left)}else{*v::at<usize>(ref order,right)});if(take_left){left+=1;}else{right+=1;}out+=1;}start=end;
    }at=0;while(at<size){put(ref mut order,at,*v::at<usize>(ref scratch,at));at+=1;}width*=2;}
    return .ok(Index{.keys=names,.spans,.order});
}
pub fn usize ordered(ref Index index,usize at){assert(v::len<usize>(ref index.order)==count(index) && at<count(index));return *v::at<usize>(ref index.order,at);}
// Returns original append position plus one; zero means absent.
pub fn usize find(ref Index index,ref u8[] value){assert(v::len<usize>(ref index.order)==count(index));var usize low=0;var usize high=count(index);while(low<high){let usize middle=low+(high-low)/2;let usize at=ordered(index,middle);let ref u8[] candidate=key(index,at);if(bytes::equal(candidate,value)){return at+1;}if(rules::before(candidate,value)){low=middle+1;}else{high=middle;}}return 0;}
pub fn Result<usize,u32> find_metered(ref Index index,ref u8[] value,ref mut usize meter){assert(v::len<usize>(ref index.order)==count(index));var usize low=0;var usize high=count(index);while(low<high){let usize middle=low+(high-low)/2;let usize at=ordered(index,middle);let ref u8[] candidate=key(index,at);if(!work::charge(meter,candidate.len+value.len+1)){return .err(2);}if(bytes::equal(candidate,value)){return .ok(at+1);}if(rules::before(candidate,value)){low=middle+1;}else{high=middle;}}return .ok(0);}
