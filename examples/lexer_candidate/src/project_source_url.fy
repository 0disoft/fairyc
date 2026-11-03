module project_source_url;

// Source-pin URLs, not a general URL parser. Match the Go preparation policy:
// absolute HTTPS, nonempty host and path, no credentials/query/fragment.
fn bool digit(u8 c){return c>=48 && c<=57;}
fn u8 folded(u8 c){return if(c>=65 && c<=90){c+32}else{c};}
fn u32 hex(u8 c){if(digit(c)){return ((c-48) as u32);}let u8 lower=folded(c);return if(lower>=97 && lower<=102){((lower-97) as u32)+10}else{16};}
fn bool host_byte(u32 c){
    if(c>=128 || (c>=48 && c<=57) || (c>=65 && c<=90) || (c>=97 && c<=122)){return true;}
    return c==33 || c==34 || c==36 || c==38 || c==39 || c==40 || c==41 || c==42 || c==43 || c==44 || c==45 || c==46 || c==58 || c==59 || c==60 || c==61 || c==62 || c==91 || c==93 || c==95 || c==126;
}
fn bool escaped(ref u8[] value,bool host,bool zone){
    var usize i=0;while(i<value.len){let u8 c=value[i];
        if(c==37){
            if(value.len-i<3 || hex(value[i+1])==16 || hex(value[i+2])==16){return false;}
            let u32 decoded=hex(value[i+1])*16+hex(value[i+2]);
            if(host && !zone && decoded<128 && decoded!=37){return false;}
            if(zone && decoded!=37 && decoded!=32 && (decoded>=128 || !host_byte(decoded))){return false;}
            i+=3;
        }else{if(host && !host_byte(c as u32)){return false;}i+=1;}
    }return true;
}
fn bool port(ref u8[] value){
    if(value.len==0){return true;}if(value[0]!=58){return false;}
    if(value.len==1){return true;}var u32 number=0;var usize i=1;
    while(i<value.len){if(!digit(value[i])){return false;}number=number*10+((value[i]-48) as u32);if(number>65535){return false;}i+=1;}
    return number!=0;
}
fn bool ipv4(ref u8[] value){
    var usize i=0;var usize parts=0;while(i<value.len){let usize start=i;var u32 number=0;
        while(i<value.len && digit(value[i])){number=number*10+((value[i]-48) as u32);if(number>255){return false;}i+=1;}
        if(i==start || (i-start>1 && value[start]==48)){return false;}parts+=1;
        if(i<value.len){if(value[i]!=46 || i+1==value.len){return false;}i+=1;}
    }return parts==4;
}
fn bool ipv6(ref u8[] value){
    var usize i=0;var usize groups=0;var bool compressed=false;
    if(value.len==0){return false;}
    if(value[0]==58){if(value.len<2 || value[1]!=58){return false;}compressed=true;i=2;}
    while(i<value.len){
        let usize start=i;while(i<value.len && value[i]!=58){i+=1;}
        var usize at=start;var bool dotted=false;while(at<i){if(value[at]==46){dotted=true;}at+=1;}
        if(dotted){if(i!=value.len || !ipv4(value[start..i])){return false;}groups+=2;}
        else{if(i==start || i-start>4){return false;}at=start;while(at<i){if(hex(value[at])==16){return false;}at+=1;}groups+=1;}
        if(groups>8){return false;}
        if(i<value.len){i+=1;if(i<value.len && value[i]==58){if(compressed){return false;}compressed=true;i+=1;}else if(i==value.len){return false;}}
    }return if(compressed){groups<8}else{groups==8};
}
fn bool authority(ref u8[] value){
    if(value.len==0){return false;}var usize at=0;
    while(at<value.len){if(value[at]==64 || (value[at]==91 && at!=0)){return false;}at+=1;}
    if(value[0]==91){
        var usize end=0;at=1;while(at<value.len){if(value[at]==93){end=at;}at+=1;}
        if(end==0 || !port(value[end+1..])){return false;}
        var usize zone=end;at=1;while(at+2<end){if(value[at]==37 && value[at+1]==50 && value[at+2]==53){zone=at;break;}at+=1;}
        if(!ipv6(value[1..zone])){return false;}
        return zone==end || (zone+3<end && escaped(value[zone..end],true,true));
    }
    var usize end=value.len;at=0;while(at<value.len){if(value[at]==58){end=at;break;}at+=1;}
    return end!=0 && port(value[end..]) && escaped(value[..end],true,false);
}
pub fn bool valid(ref u8[] value){
    if(value.len<9 || folded(value[0])!=104 || folded(value[1])!=116 || folded(value[2])!=116 || folded(value[3])!=112 || folded(value[4])!=115 || value[5]!=58 || value[6]!=47 || value[7]!=47){return false;}
    var usize i=0;var usize slash=value.len;
    while(i<value.len){let u8 c=value[i];if(c<33 || c==127 || c==92 || c==35 || c==63){return false;}if(i>=8 && c==47 && slash==value.len){slash=i;}i+=1;}
    return slash<value.len && authority(value[8..slash]) && escaped(value[slash..],false,false);
}
