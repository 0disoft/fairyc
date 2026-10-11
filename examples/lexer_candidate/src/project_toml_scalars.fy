module project_toml_scalars;
import std::bytes as bytes;

// TOML atoms retained for manifest type/unknown-field rejection. This does not
// add floating-point or date values to the Fairy language or manifest schema.
// Kind: 6 integer, 7 float, 8 date/time. Zero is invalid syntax/range.
pub struct Token{u32 kind;usize end;}
struct Digits{usize end;usize count;u8 first;bool valid;}
fn i64 digit(u8 c){if(c>=48 && c<=57){return (c-48) as i64;}if(c>=65 && c<=70){return (c-55) as i64;}if(c>=97 && c<=102){return (c-87) as i64;}return 16;}
fn bool decimal(u8 c){return c>=48 && c<=57;}
fn Digits digits(ref u8[] raw,usize start,i64 base){
    var usize i=start;var usize count=0;var u8 first=0;var bool previous=false;
    while(i<raw.len){let u8 c=raw[i];if(c==95){if(!previous || i+1==raw.len || digit(raw[i+1])>=base){return Digits{.end=i,.count,.first,.valid=false};}previous=false;i+=1;continue;}
        if(digit(c)>=base){break;}if(count==0){first=c;}count+=1;previous=true;i+=1;
    }return Digits{.end=i,.count,.first,.valid=count!=0 && previous};
}
fn bool integer(ref u8[] raw){
    if(raw.len==0){return false;}var usize start=0;let bool negative=raw[0]==45;let bool signed=negative || raw[0]==43;if(signed){start=1;}if(start==raw.len){return false;}
    var i64 base=10;if(raw.len-start>=2 && raw[start]==48){let u8 prefix=raw[start+1];if(prefix==120 || prefix==111 || prefix==98){if(signed){return false;}base=if(prefix==120){16}else{if(prefix==111){8}else{2}};start+=2;}}
    let Digits d=digits(raw,start,base);if(!d.valid || d.end!=raw.len || (base==10 && d.count>1 && d.first==48)){return false;}
    let i64 limit=if(negative){-9223372036854775808}else{-9223372036854775807};var i64 value=0;var usize i=start;
    while(i<raw.len){if(raw[i]!=95){let i64 n=digit(raw[i]);if(value<limit/base || (value==limit/base && n>-(limit%base))){return false;}value=value*base-n;}i+=1;}return true;
}
fn bool float(ref u8[] raw){
    if(raw.len==0){return false;}let usize start=if(raw[0]==43 || raw[0]==45){1}else{0};if(start==raw.len){return false;}
    if(bytes::equal(raw[start..],b"inf") || bytes::equal(raw[start..],b"nan")){return true;}
    let Digits whole=digits(raw,start,10);if(!whole.valid || (whole.count>1 && whole.first==48)){return false;}
    var usize at=whole.end;var bool floating=false;
    if(at<raw.len && raw[at]==46){floating=true;let Digits fraction=digits(raw,at+1,10);if(!fraction.valid){return false;}at=fraction.end;}
    let usize mantissa_end=at;var i64 exponent=0;
    if(at<raw.len && (raw[at]==101 || raw[at]==69)){floating=true;at+=1;let bool minus=at<raw.len && raw[at]==45;if(at<raw.len && (raw[at]==43 || raw[at]==45)){at+=1;}
        let Digits exp=digits(raw,at,10);if(!exp.valid){return false;}var usize i=at;while(i<exp.end){if(raw[i]!=95 && exponent<2097152){exponent=exponent*10+digit(raw[i]);if(exponent>2097152){exponent=2097152;}}i+=1;}if(minus){exponent=-exponent;}at=exp.end;
    }
    if(!floating || at!=raw.len){return false;}
    // Exact overflow threshold: 2^1024 - 2^970, the midpoint above the largest
    // finite binary64 value. Underflow and signed zero remain valid, like Go.
    var i64 whole_digits=0;var i64 seen=0;var i64 first=-1;var usize first_byte=0;var bool fractional=false;var usize i=start;
    while(i<mantissa_end){let u8 c=raw[i];if(c==46){fractional=true;}else if(decimal(c)){if(!fractional){whole_digits+=1;}if(first<0 && c!=48){first=seen;first_byte=i;}seen+=1;}i+=1;}
    if(first<0){return true;}let i64 order=exponent+whole_digits-first-1;if(order<308){return true;}if(order>308){return false;}
    let ref u8[] boundary=b"179769313486231580793728971405303415079934132710037826936173778980444968292764750946649017977587207096330286416692887910946555547851940402630657488671505820681908902000708383676273854845817711531764475730270069855571366959622842914819860834936475292719074168444365510704342711559699508093042880177904174497792";
    i=first_byte;var usize n=0;while(n<boundary.len){while(i<mantissa_end && !decimal(raw[i])){i+=1;}let u8 c=if(i<mantissa_end){raw[i]}else{48};if(c!=boundary[n]){return c<boundary[n];}if(i<mantissa_end){i+=1;}n+=1;}return false;
}
fn usize pair(ref u8[] raw,usize at){return if(raw.len-at>=2 && decimal(raw[at]) && decimal(raw[at+1])){((raw[at]-48) as usize)*10+((raw[at+1]-48) as usize)}else{100};}
fn bool date(ref u8[] raw){
    if(raw.len<10 || raw[4]!=45 || raw[7]!=45){return false;}var usize year=0;var usize i=0;while(i<4){if(!decimal(raw[i])){return false;}year=year*10+((raw[i]-48) as usize);i+=1;}
    let usize month=pair(raw,5);let usize day=pair(raw,8);if(month<1 || month>12 || day<1){return false;}var usize limit=31;
    if(month==4 || month==6 || month==9 || month==11){limit=30;}if(month==2){limit=if(year%4==0 && (year%100!=0 || year%400==0)){29}else{28};}return day<=limit;
}
fn bool datetime(ref u8[] raw){
    var usize start=0;var bool dated=false;if(raw.len>=10 && raw[4]==45){if(!date(raw)){return false;}dated=true;if(raw.len==10){return true;}if(raw[10]!=84 && raw[10]!=116 && raw[10]!=32){return false;}start=11;}
    if(raw.len-start<5 || raw[start+2]!=58 || pair(raw,start)>23 || pair(raw,start+3)>59){return false;}var usize at=start+5;var bool seconds=false;
    if(at<raw.len && raw[at]==58){seconds=true;at+=1;if(raw.len-at<2 || pair(raw,at)>59){return false;}at+=2;}
    if(at<raw.len && raw[at]==46){if(!seconds){return false;}at+=1;let usize begin=at;while(at<raw.len && decimal(raw[at])){at+=1;}if(at==begin){return false;}}
    if(at==raw.len){return true;}if(!dated){return false;}
    if(raw[at]==90 || raw[at]==122){return at+1==raw.len;}
    if(raw[at]!=43 && raw[at]!=45 || raw.len-at!=6 || raw[at+3]!=58){return false;}
    // The pinned Go time parser accepts the inclusive 24/60 offset bounds.
    return pair(raw,at+1)<=24 && pair(raw,at+4)<=60;
}
pub fn u32 kind(ref u8[] raw){if(raw.len==0 || raw.len>1048576){return 0;}if(integer(raw)){return 6;}if(float(raw)){return 7;}if(datetime(raw)){return 8;}return 0;}
pub fn Token read(ref u8[] source,usize start){
    if(start>=source.len || source.len>1048576){return Token{.kind=0,.end=if(start<source.len){start}else{source.len}};}
    let bool dated=source.len-start>=10 && source[start+4]==45 && source[start+7]==45;let bool timed=source.len-start>=5 && source[start+2]==58;
    var usize end=start;while(end<source.len){let u8 c=source[end];if(c==9 || c==10 || c==13 || c==35 || c==44 || c==93 || c==125 || (c==32 && !dated && !timed)){break;}end+=1;}
    var usize cooked=end;while(cooked>start && source[cooked-1]==32){cooked-=1;}return Token{.kind=kind(source[start..cooked]),.end=cooked};
}
