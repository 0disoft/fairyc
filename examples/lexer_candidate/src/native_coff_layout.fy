module native_coff_layout;
import native_pe as read;
import std::bytes as bytes;
import std::vec as v;

// Source contracts: pecoff/object.go, big_object.go and debug/pe parsing.
// Names are spans into the input; no decoded string allocations are needed.
pub struct Name{usize start;usize end;}
pub struct Header{bool big;usize count;usize table;usize symbols;usize entries;usize stride;usize strings;usize strings_end;usize flags;}
pub struct Section{Name name;usize size;usize offset;usize flags;usize rel;usize count;usize stored;usize lines;usize selection;usize parent;usize definition;usize first;usize primary;}
pub struct Symbol{Name name;usize index;usize value;usize section;usize type;usize storage;usize aux;usize fallback;usize search;}
pub resource Layout{Header header;v::Vec<Section> sections;v::Vec<Symbol> symbols;v::Vec<usize> indices;cleanup{}}
// 1 header/bounds, 2 names/strings, 3 symbols, 4 allocation.
fn usize end(ref u8[] data,usize start,usize stop){var usize i=start;while(i<stop && data[i]!=0){i+=1;}return i;}
pub fn bool zero(ref u8[] data,usize start,usize stop){var usize i=start;while(i<stop){if(data[i]!=0){return false;}i+=1;}return true;}
pub fn bool equal(ref u8[] data,Name a,Name b){return bytes::equal(data[a.start..a.end],data[b.start..b.end]);}
fn Result<Name,u32> string(ref u8[] data,Header h,usize offset){
    if(offset<4 || offset-4>h.strings_end-h.strings || h.big && offset-4==h.strings_end-h.strings){return .err(2);}let usize first=h.strings+offset-4;let usize last=end(data,first,h.strings_end);if(h.big && last==h.strings_end){return .err(2);}return .ok(Name{.start=first,.end=last});
}
fn Result<Name,u32> name(ref u8[] data,Header h,usize at,bool section){
    let usize last=end(data,at,at+8);
    if(section && data[at]==47){if(h.big){return .err(2);}var usize i=at+1;if(i<last && data[i]==43){i+=1;}if(i==last){return .err(2);}var usize index=0;while(i<last){if(data[i]<48 || data[i]>57){return .err(2);}index=index*10+(data[i]-48) as usize;i+=1;}return string(data,h,index);}
    if(!section && read::word(data,at)==0 && (h.big || read::word(data,at+4)!=0)){return string(data,h,read::word(data,at+4));}
    if(h.big && (section && read::word(data,at)==0 || !zero(data,last,at+8))){return .err(2);}return .ok(Name{.start=at,.end=last});
}
pub fn Result<Header,u32> read_header(ref u8[] data){
    if(data.len<20 || data.len>67108864 || data[0]==77 && data[1]==90){return .err(1);}let bool big=read::half(data,0)==0 && read::half(data,2)==65535;
    var Header h=Header{.big,.count=0,.table=20,.symbols=0,.entries=0,.stride=18,.strings=0,.strings_end=0,.flags=0};
    if(big){if(data.len<56 || read::half(data,4)!=2 || read::half(data,6)!=34404 || !bytes::equal(data[12..28],b"\xc7\xa1\xba\xd1\xee\xba\xa9\x4b\xaf\x20\xfa\xf6\x6a\xa4\xdc\xb8") || !zero(data,28,44)){return .err(1);}h.table=56;h.count=read::word(data,44);h.symbols=read::word(data,48);h.entries=read::word(data,52);h.stride=20;if(h.entries==0 || h.entries>262144){return .err(3);}}
    else{if(data.len<96 || read::half(data,0)!=34404 || read::half(data,16)!=0){return .err(1);}h.count=read::half(data,2);h.flags=read::half(data,18);h.symbols=read::word(data,8);h.entries=if(h.symbols==0){0}else{read::word(data,12)};}
    if(h.count==0 || h.count>16384 || !read::range(data,h.table,h.count*40)){return .err(1);}let usize header_end=h.table+h.count*40;
    if(h.symbols>0){let usize stop=h.symbols+h.entries*h.stride;if(h.big && h.symbols<header_end || !read::range(data,h.symbols,h.entries*h.stride+4)){return .err(3);}let usize length=read::word(data,stop);if(h.big && length<4){return .err(2);}if(length>4){if(!read::range(data,stop,length)){return .err(2);}h.strings=stop+4;h.strings_end=stop+length;}else{h.strings=stop+4;h.strings_end=stop+4;}}
    return .ok(h);
}
pub fn Result<v::Vec<Section>,u32> read_sections(ref u8[] data,Header h){
    var v::Vec<Section> sections=v::empty<Section>();var usize i=0;
    while(i<h.count){let usize at=h.table+i*40;let n=name(data,h,at,true)?;let usize count=read::half(data,at+32);let usize rel=read::word(data,at+24);if(!h.big && count>0 && !read::range(data,rel,count*10)){return .err(1);}
        if(!v::push<Section>(ref mut sections,Section{.name=n,.size=read::word(data,at+16),.offset=read::word(data,at+20),.flags=read::word(data,at+36),.rel,.count,.stored=count,.lines=read::half(data,at+34),.selection=0,.parent=0,.definition=0,.first=0,.primary=0}).is_ok){return .err(4);}i+=1;
    }
    return .ok(sections);
}
pub fn Result<v::Vec<Symbol>,u32> read_symbols(ref u8[] data,Header h,ref mut v::Vec<usize> indices){
    var v::Vec<Symbol> symbols=v::empty<Symbol>();
    if(!v::reserve<usize>(indices,h.entries).is_ok){return .err(4);}var usize i=0;while(i<h.entries){if(!v::push<usize>(indices,0).is_ok){return .err(4);}i+=1;}i=0;
    while(i<h.entries){let usize at=h.symbols+i*h.stride;let n=name(data,h,at,false)?;let usize auxiliary=data[at+h.stride-1] as usize;if(auxiliary>=h.entries-i){return .err(3);}let usize raw=if(h.big){read::word(data,at+12)}else{read::half(data,at+12)};
        let usize section=raw;
        let symbol=Symbol{.name=n,.index=i,.value=read::word(data,at+8),.section,.type=read::half(data,at+h.stride-4),.storage=data[at+h.stride-2] as usize,.aux=auxiliary,.fallback=0,.search=0};
        if(!v::push<Symbol>(ref mut symbols,symbol).is_ok){return .err(4);}let slot=v::at_mut<usize>(indices,i);*slot=v::len<Symbol>(ref symbols);i+=1+auxiliary;
    }
    return .ok(symbols);
}
pub fn Layout finish(Header h,v::Vec<Section> sections,v::Vec<Symbol> symbols,v::Vec<usize> indices){return Layout{.header=h,.sections,.symbols,.indices};}
pub fn Result<Layout,u32> parse(ref u8[] data){
    let h=read_header(data)?;let sections=read_sections(data,h)?;var v::Vec<usize> indices=v::empty<usize>();let symbols=read_symbols(data,h,ref mut indices)?;return .ok(finish(h,sections,symbols,indices));
}
pub fn bool text(ref u8[] data,Name n){let value=data[n.start..n.end];return bytes::equal(value,b".text") || value.len>=6 && bytes::equal(value[..6],b".text$");}

pub fn Header header(ref Layout layout){return layout.header;}
pub fn ref v::Vec<Section> sections(ref Layout layout){return ref layout.sections;}
pub fn ref v::Vec<Symbol> symbols(ref Layout layout){return ref layout.symbols;}
pub fn ref v::Vec<usize> indices(ref Layout layout){return ref layout.indices;}
