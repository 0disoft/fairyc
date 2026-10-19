module native_coff;
import native_coff_layout as l;
import native_pe as read;
import std::vec as v;

// Error codes retain categories, not the Go inspector's diagnostic wording.
// 5 symbol semantics, 6 relocations, 7 raw ranges, 8 section definitions,
// 9 COMDAT identity/association, 10 missing or malformed code.
pub resource Report{l::Layout layout;usize relocations;cleanup{}}
struct Span{usize start;usize end;}
fn Result<usize,u32> add_range(ref mut v::Vec<Span> ranges,usize start,usize stop){
    var usize i=0;while(i<v::len<Span>(ranges)){let old=*v::at<Span>(ranges,i);if(start<old.end && old.start<stop){return .err(7);}i+=1;}
    if(!v::push<Span>(ranges,Span{.start,.end=stop}).is_ok){return .err(4);}return .ok(0);
}
fn Result<usize,u32> symbols(ref u8[] data,l::Header h,ref mut v::Vec<l::Section> sections,ref mut v::Vec<l::Symbol> records,ref v::Vec<usize> indices){
    let usize count=v::len<l::Symbol>(records);if(count==0){return .err(5);}var usize i=0;
    while(i<count){let s=*v::at<l::Symbol>(records,i);if(s.name.start==s.name.end){return .err(5);}
        if(h.big){if(s.aux!=0 || s.section>h.count && s.section<4294967294){return .err(5);}}
        else{
            if(s.storage==2 && s.section==0 && s.value==0 && s.aux!=0){return .err(5);}
            if(s.storage==105){if(s.section!=0 || s.value!=0 || s.aux!=1){return .err(5);}let usize aux=h.symbols+(s.index+1)*18;let usize fallback=read::word(data,aux);let usize search=read::word(data,aux+4);
                if(search<1 || search>3 || !l::zero(data,aux+8,aux+18) || fallback==s.index || fallback>=h.entries || *v::at<usize>(indices,fallback)==0){return .err(5);}let own=v::at_mut<l::Symbol>(records,i);own.fallback=fallback;own.search=search;
            }
        }
        if(s.section>0 && s.section<=h.count){let target=v::at_mut<l::Section>(sections,s.section-1);if(target.first==0){target.first=i+1;}target.primary+=1;}i+=1;
    }return .ok(count);
}
fn Result<usize,u32> sections(ref u8[] data,l::Header h,ref mut v::Vec<l::Section> sections,ref v::Vec<l::Symbol> records,ref v::Vec<usize> indices){
    let usize header_end=h.table+h.count*40;var v::Vec<Span> ranges=v::empty<Span>();var usize total=0;var bool text=false;
    if(h.big){let usize added=add_range(ref mut ranges,h.symbols,h.strings_end)?;}var usize i=0;
    while(i<h.count){var s=*v::at<l::Section>(sections,i);if(s.name.start==s.name.end || h.big && s.flags&4096!=0){return .err(8);}
        if(s.size>0){if(s.offset==0){if(s.flags&128==0){return .err(7);}}else{if(s.offset<header_end || !read::range(data,s.offset,s.size)){return .err(7);}let usize added=add_range(ref mut ranges,s.offset,s.offset+s.size)?;}}
        let bool overflow=s.flags&16777216!=0;if(overflow && s.count!=65535){return .err(6);}var usize first=0;
        if(s.count>0){if(s.rel<header_end || !read::range(data,s.rel,10)){return .err(6);}if(overflow){s.stored=read::word(data,s.rel);if(s.stored<=65535 || read::word(data,s.rel+4)!=0 || read::half(data,s.rel+8)!=0){return .err(6);}first=1;}
            s.count=s.stored-first;if(s.count>1048576-total || !read::range(data,s.rel,s.stored*10)){return .err(6);}if(h.big){let usize added=add_range(ref mut ranges,s.rel,s.rel+s.stored*10)?;}
            var usize at=first;while(at<s.stored){let usize record=s.rel+at*10;let usize symbol=read::word(data,record+4);if(read::half(data,record+8)>16 || symbol>=h.entries){return .err(6);}let usize index=*v::at<usize>(indices,symbol);if(index==0){return .err(6);}let target=*v::at<l::Symbol>(records,index-1);if(target.name.start==target.name.end){return .err(6);}at+=1;}
        }
        total+=s.count;if(l::text(data,s.name)){if(s.flags&32==0 || s.flags&536870912==0 || s.flags&1073741824==0){return .err(10);}text=true;}
        let slot=v::at_mut<l::Section>(sections,i);*slot=s;i+=1;
    }
    if(!text){return .err(10);}return .ok(total);
}
fn Result<usize,u32> comdat(ref u8[] data,l::Header h,ref mut v::Vec<l::Section> sections,ref v::Vec<l::Symbol> records){
    if(h.big){return .ok(0);}var usize i=0;
    while(i<v::len<l::Symbol>(records)){let symbol=*v::at<l::Symbol>(records,i);i+=1;if(symbol.section<=0 || symbol.section>h.count){continue;}let usize index=symbol.section-1;let s=*v::at<l::Section>(sections,index);
        if(symbol.storage!=3 || !l::equal(data,symbol.name,s.name) || symbol.value!=0 || symbol.type!=0){continue;}if(symbol.aux!=1 || s.definition!=0){return .err(8);}let usize aux=h.symbols+(symbol.index+1)*18;
        if(read::word(data,aux)!=s.size || read::half(data,aux+4)!=if(s.count>65535){65535}else{s.count} || read::half(data,aux+6)!=s.lines){return .err(8);}let usize parent=read::half(data,aux+12);let usize selection=data[aux+14] as usize;
        if(s.flags&4096==0){if(selection!=0 || parent!=0){return .err(8);}continue;}
        if(selection<1 || selection>6 || selection==5 && (parent==0 || parent>h.count || parent==index+1) || selection!=5 && parent!=0){return .err(9);}let slot=v::at_mut<l::Section>(sections,index);slot.selection=selection;slot.parent=parent;slot.definition=i;
    }
    i=0;while(i<h.count){let s=*v::at<l::Section>(sections,i);if(s.flags&4096!=0){if(s.definition==0 || s.definition!=s.first || s.selection!=5 && s.primary<2){return .err(9);}var usize current=i;var usize steps=0;
            while(true){if(steps>=h.count){return .err(9);}steps+=1;let next=*v::at<l::Section>(sections,current);if(next.definition==0 || next.flags&4096==0){return .err(9);}if(next.selection!=5){break;}current=next.parent-1;}
        }i+=1;
    }return .ok(0);
}
pub fn Result<Report,u32> inspect(ref u8[] data){
    let h=l::read_header(data)?;if(!h.big && (h.flags&8194)!=0){return .err(1);}var section_records=l::read_sections(data,h)?;var v::Vec<usize> indices=v::empty<usize>();var records=l::read_symbols(data,h,ref mut indices)?;
    let usize primary=symbols(data,h,ref mut section_records,ref mut records,ref indices)?;let usize relocations=sections(data,h,ref mut section_records,ref records,ref indices)?;let usize definitions=comdat(data,h,ref mut section_records,ref records)?;return .ok(Report{.layout=l::finish(h,section_records,records,indices),.relocations});
}
// cdriver.inspectProjectObject's data-only fallback, including debug/pe name,
// auxiliary and 16-bit relocation-table parsing but no code-symbol admission.
pub fn bool data_only(ref u8[] data){
    let layout=switch(l::parse(data)){case .ok(p):{p}case .err(_):{return false;}};if(l::header(ref layout).big || l::header(ref layout).count>96){return false;}var usize i=0;
    while(i<l::header(ref layout).count){let s=*v::at<l::Section>(l::sections(ref layout),i);if(s.size>0 && (s.flags&32!=0 || s.flags&536870912!=0) || s.offset>0 && !read::range(data,s.offset,s.size)){return false;}i+=1;}return true;
}
pub fn u32 project_problem(ref u8[] data){return switch(inspect(data)){case .ok(_):{0}case .err(code):{if(data_only(data)){0}else{code}}};}

pub fn ref l::Layout layout(ref Report report){return ref report.layout;}
pub fn usize relocations(ref Report report){return report.relocations;}
