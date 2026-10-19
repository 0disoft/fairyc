module native_pe_unwind;
import native_pe as pe;
import std::vec as v;

// Source contract: internal/pecoff/unwind.go. Shared RVAs are inspected once,
// preserving the Go report's first-function-size behavior for shared records.
pub struct Report{usize functions;usize unique_infos;usize handlers;usize chains;usize version2;usize maximum_codes;}
struct Info{usize codes;usize flags;usize version;}
fn usize slots(usize op,usize info){
    if(op==0 || op==2 || op==3){return 1;}if(op==1){return if(info==0){2}else if(info==1){3}else{0};}if(op==4 || op==8){return 2;}if(op==5 || op==9){return 3;}if(op==10 && info<=1){return 1;}return 0;
}
fn Result<Info,u32> info(ref u8[] data,pe::Header h,usize rva,usize function_size){
    let first=pe::map(data,h,rva,4)?;let usize header=first.offset;if(first.section.flags&1073741824==0 || first.section.flags&536870912!=0){return .err(8);}
    let usize version=(data[header]&7) as usize;let usize flags=(data[header]>>3) as usize;let usize prolog=data[header+1] as usize;let usize count=data[header+2] as usize;let u8 frame=data[header+3];
    if(version!=1 && version!=2 || flags>7 || (flags&4)!=0 && (flags&3)!=0 || prolog>function_size || frame&15==0 && frame>>4!=0){return .err(8);}
    let usize aligned=count+count%2;let usize size=4+aligned*2+if(flags&3!=0){4}else{0}+if(flags&4!=0){12}else{0};let record=pe::map(data,h,rva,size)?;var usize slot=0;var usize previous=256;var bool epilog=false;
    while(slot<count){let usize offset=data[record.offset+4+slot*2] as usize;let u8 value=data[record.offset+5+slot*2];let usize op=(value&15) as usize;let usize op_info=(value>>4) as usize;
        if(version==2 && op==6){let usize consumed=if(!epilog && op_info&1==0){2}else{1};if(slot+consumed>count){return .err(9);}epilog=true;slot+=consumed;continue;}
        if(version==2 && op==7){slot+=1;continue;}let usize consumed=slots(op,op_info);if(offset>prolog || offset>previous || consumed==0 || slot+consumed>count){return .err(9);}previous=offset;slot+=consumed;
    }
    let usize tail=record.offset+4+aligned*2;
    if(flags&3!=0){let usize handler=pe::word(data,tail);if(handler==0 || handler==4294967295 || !pe::executable_range(data,h,handler,handler+1)){return .err(10);}}
    if(flags&4!=0){let usize begin=pe::word(data,tail);let usize end=pe::word(data,tail+4);let usize chain=pe::word(data,tail+8);if(begin==0 || begin>=end || chain==0 || chain%4!=0 || !pe::executable_range(data,h,begin,end)){return .err(10);}}
    return .ok(Info{.codes=count,.flags,.version});
}
// PE parse failures keep codes 1..7; 8 info header, 9 op slots/order,
// 10 handler/chain, 11 directory/functions, 12 allocation.
pub fn Result<Report,u32> inspect(ref u8[] data){
    let h=pe::inspect(data)?;let usize rva=pe::directory_rva(data,h,3);let usize size=pe::directory_size(data,h,3);if(rva==0 || size==0 || size%12!=0){return .err(11);}let directory=pe::map(data,h,rva,size)?;
    if(!pe::named(data,directory.section,b".pdata") || directory.section.flags&1073741824==0 || directory.section.flags&536870912!=0){return .err(11);}
    let usize count=size/12;var Report report=Report{.functions=count,.unique_infos=0,.handlers=0,.chains=0,.version2=0,.maximum_codes=0};var usize capacity=2;while(capacity<count*2){capacity*=2;}
    var v::Vec<usize> seen=v::empty<usize>();if(!v::reserve<usize>(ref mut seen,capacity).is_ok){return .err(12);}var usize i=0;while(i<capacity){if(!v::push<usize>(ref mut seen,0).is_ok){return .err(12);}i+=1;}
    i=0;var usize previous=0;while(i<count){let usize at=directory.offset+i*12;let usize begin=pe::word(data,at);let usize end=pe::word(data,at+4);let usize unwind=pe::word(data,at+8);
        if(begin==0 || begin>=end || unwind==0 || unwind%4!=0 || i>0 && begin<previous || !pe::executable_range(data,h,begin,end)){return .err(11);}previous=end;i+=1;
        var usize key=((unwind/4)*2654435761)%capacity;while(*v::at<usize>(ref seen,key)!=0 && *v::at<usize>(ref seen,key)!=unwind){key=(key+1)%capacity;}
        if(*v::at<usize>(ref seen,key)==unwind){continue;}let slot=v::at_mut<usize>(ref mut seen,key);*slot=unwind;
        let parsed=info(data,h,unwind,end-begin)?;report.unique_infos+=1;if(parsed.flags&3!=0){report.handlers+=1;}if(parsed.flags&4!=0){report.chains+=1;}if(parsed.version==2){report.version2+=1;}if(parsed.codes>report.maximum_codes){report.maximum_codes=parsed.codes;}
    }return .ok(report);
}
