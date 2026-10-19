module native_pe;
import std::bytes as bytes;

// Source contract: internal/pecoff/executable.go. All offsets are borrowed
// from the validated input; names include COFF string-table section names.
pub struct Header{usize optional;usize table;usize count;usize directories;usize strings;usize strings_end;}
pub struct Section{usize name;usize name_end;usize address;usize virtual_size;usize offset;usize size;usize flags;}
// 1 size/signature, 2 header, 3 symbol/string table, 4 section name,
// 5 section range/order, 6 code/entry. No output artifact grants conformance.
pub fn usize half(ref u8[] data,usize at){return (data[at] as usize)+(data[at+1] as usize)*256;}
pub fn usize word(ref u8[] data,usize at){return half(data,at)+half(data,at+2)*65536;}
pub fn bool range(ref u8[] data,usize offset,usize size){return offset<=data.len && size<=data.len-offset;}
fn bool power(usize n){return n>0 && (n&(n-1))==0;}
fn usize end_name(ref u8[] data,usize start,usize end){var usize i=start;while(i<end && data[i]!=0){i+=1;}return i;}
fn Result<usize,u32> string_at(Header h,usize index){if(index<4 || index-4>h.strings_end-h.strings){return .err(3);}return .ok(h.strings+index-4);}
pub fn Result<Section,u32> section(ref u8[] data,Header h,usize index){
    if(index>=h.count || !range(data,h.table+index*40,40)){return .err(2);}let usize at=h.table+index*40;var usize name=at;var usize stop=end_name(data,at,at+8);
    if(data[at]==47){var usize i=at+1;if(i<stop && data[i]==43){i+=1;}if(i==stop){return .err(4);}var usize value=0;
        while(i<stop){if(data[i]<48 || data[i]>57){return .err(4);}value=value*10+(data[i]-48) as usize;i+=1;}
        name=string_at(h,value)?;stop=end_name(data,name,h.strings_end);
    }
    return .ok(Section{.name,.name_end=stop,.address=word(data,at+12),.virtual_size=word(data,at+8),.offset=word(data,at+20),.size=word(data,at+16),.flags=word(data,at+36)});
}
pub fn bool named(ref u8[] data,Section s,ref u8[] name){return bytes::equal(data[s.name..s.name_end],name);}
pub fn usize span(Section s){return if(s.virtual_size>s.size){s.virtual_size}else{s.size};}
pub fn bool code(Section s){return (s.flags&32)!=0 && (s.flags&536870912)!=0;}
pub fn Result<Header,u32> inspect(ref u8[] data){
    if(data.len<96 || data.len>67108864 || data[0]!=77 || data[1]!=90){return .err(1);}let usize pe=word(data,60);
    if(!range(data,pe,24) || !bytes::equal(data[pe..pe+4],b"PE\0\0") || half(data,pe+4)!=34404){return .err(1);}let usize optional=pe+24;let usize size=half(data,pe+20);let usize count=half(data,pe+6);let usize flags=half(data,pe+22);
    if(size<112 || count==0 || count>96 || !range(data,optional,size+count*40) || half(data,optional)!=523 || flags&2==0 || flags&32==0 || flags&8192!=0 || half(data,optional+68)!=3){return .err(2);}
    let usize directories=word(data,optional+108);if(directories*8!=size-112){return .err(2);}
    let usize entry=word(data,optional+16);let usize alignment=word(data,optional+32);let usize file_alignment=word(data,optional+36);let usize headers=word(data,optional+60);let usize image=word(data,optional+56);
    if(entry==0 || word(data,optional+24)==0 && word(data,optional+28)==0 || word(data,optional+24)%65536!=0 || !power(alignment) || !power(file_alignment) || file_alignment<512 || file_alignment>65536 || alignment<file_alignment || headers==0 || headers>data.len || headers%file_alignment!=0 || image==0 || image%alignment!=0){return .err(2);}
    var Header h=Header{.optional,.table=optional+size,.count,.directories,.strings=0,.strings_end=0};let usize symbols=word(data,pe+12);let usize entries=word(data,pe+16);
    if(symbols>0){if(!range(data,symbols,entries*18+4)){return .err(3);}let usize strings=symbols+entries*18;let usize length=word(data,strings);if(length>4){if(!range(data,strings,length)){return .err(3);}h.strings=strings+4;h.strings_end=strings+length;}
        var usize index=0;while(index<entries){let usize at=symbols+index*18;let usize aux=data[at+17] as usize;if(aux>=entries-index){return .err(3);}if(word(data,at)==0 && word(data,at+4)!=0){let usize checked=string_at(h,word(data,at+4))?;}index+=1+aux;}
    }
    var usize i=0;var usize previous_address=0;var usize previous_end=0;var bool text=false;var bool entry_found=false;
    while(i<count){let s=section(data,h,i)?;if(s.name==s.name_end){return .err(4);}var usize j=0;
        while(j<i){let other=section(data,h,j)?;if(bytes::equal(data[s.name..s.name_end],data[other.name..other.name_end])){return .err(4);}j+=1;}
        let usize width=span(s);if(s.address==0 || s.address%alignment!=0 || i>0 && (s.address<=previous_address || s.address<previous_end) || width==0 || s.address+width>image || s.size>0 && (s.offset<headers || s.offset%file_alignment!=0 || s.size%file_alignment!=0 || !range(data,s.offset,s.size))){return .err(5);}
        let usize record=h.table+i*40;let usize relocations=half(data,record+32);if(!range(data,word(data,record+24),relocations*10) && relocations>0){return .err(5);}
        previous_address=s.address;previous_end=s.address+width;
        if(named(data,s,b".text")){if(!code(s) || s.flags&1073741824==0){return .err(6);}text=true;}
        if(entry>=s.address && entry<s.address+width){if(!code(s)){return .err(6);}entry_found=true;}i+=1;
    }
    if(!text || !entry_found){return .err(6);}return .ok(h);
}
pub struct Mapped{usize offset;Section section;}
pub fn Result<Mapped,u32> map(ref u8[] data,Header h,usize rva,usize size){
    if(size==0){return .err(7);}var usize i=0;while(i<h.count){let s=section(data,h,i)?;if(rva>=s.address && rva+size<=s.address+s.size){let usize offset=s.offset+rva-s.address;if(!range(data,offset,size)){return .err(7);}return .ok(Mapped{.offset,.section=s});}i+=1;}return .err(7);
}
pub fn bool executable_range(ref u8[] data,Header h,usize begin,usize end){
    if(begin>=end){return false;}var usize i=0;while(i<h.count){let s=switch(section(data,h,i)){case .ok(s):{s}case .err(_):{return false;}};if(begin>=s.address && end<=s.address+span(s) && code(s)){return true;}i+=1;}return false;
}
pub fn usize directory_rva(ref u8[] data,Header h,usize index){return if(index<h.directories && index<16){word(data,h.optional+112+index*8)}else{0};}
pub fn usize directory_size(ref u8[] data,Header h,usize index){return if(index<h.directories && index<16){word(data,h.optional+116+index*8)}else{0};}
