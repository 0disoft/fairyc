module project_import_path;
import syntax as syntax;
import project_path as work;
import std::bytes as bytes;

pub resource Path{bytes::Buffer name;usize start;usize end;cleanup{}}
pub fn ref u8[] name(ref Path path){return bytes::view(ref path.name);}
pub fn usize start(ref Path path){return path.start;}
pub fn usize end(ref Path path){return path.end;}
// 1 invalid/unverified declaration, 2 allocation, 3 exhausted work budget.
// Reuse verified import tokens. No rescan or 64 KiB fragment limit.
pub fn Result<Path,u32> read(ref syntax::Parsed tree,usize index,ref mut usize meter){
    if(index>=syntax::declaration_count(tree)){return .err(1);}let d=syntax::declaration_at(tree,index);if(d.kind!=2 || d.start>=d.end || d.end>syntax::source_view(tree).len){return .err(1);}
    var usize low=0;var usize high=syntax::token_count(tree);while(low<high){if(!work::charge(meter,1)){return .err(3);}let usize middle=low+(high-low)/2;if(syntax::token_info(tree,middle).start<d.start){low=middle+1;}else{high=middle;}}
    var bytes::Buffer name=bytes::empty();var bool keyword=false;var usize start=0;var usize end=0;
    while(low<syntax::token_count(tree)){let t=syntax::token_info(tree,low);if(t.start>=d.name_start){break;}if(t.end>d.end || !work::charge(meter,1)){return .err(if(t.end>d.end){1}else{3});}if(t.kind!=2){let ref u8[] value=syntax::source_view(tree)[t.start..t.end];if(!work::charge(meter,value.len+1)){return .err(3);}
        if(!keyword){if(!bytes::equal(value,b"import")){return .err(1);}keyword=true;}
        else if(bytes::equal(value,b"as")){if(bytes::len(ref name)==0){return .err(1);}return .ok(Path{.name,.start,.end});}
        else{if(t.kind!=3 && !bytes::equal(value,b"::")){return .err(1);}if(bytes::len(ref name)==0){start=t.start;}if(!bytes::append(ref mut name,value).is_ok){return .err(2);}end=t.end;}
    }low+=1;}return .err(1);
}
