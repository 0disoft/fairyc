module project_manifest_values;

// Manifest-v1 value rules. TOML decoding, filesystem containment and package
// graph validation have separate owners; this leaf operates on decoded bytes.
fn bool lower(u8 value){return value>=97 && value<=122;}
fn bool digit(u8 value){return value>=48 && value<=57;}
fn bool alpha(u8 value){return lower(value) || (value>=65 && value<=90);}
fn u8 folded(u8 value){return if(value>=65 && value<=90){value+32}else{value};}

pub fn bool name(ref u8[] value){
    if(value.len==0 || !lower(value[0])){return false;}
    var usize i=1;var bool hyphen=false;
    while(i<value.len){
        let u8 c=value[i];
        if(c==45){if(hyphen){return false;}hyphen=true;}
        else {if(!lower(c) && !digit(c)){return false;}hyphen=false;}
        i+=1;
    }
    return !hyphen;
}

pub fn bool uid(ref u8[] value){
    var usize start=0;var usize i=0;var usize count=0;
    while(i<=value.len){
        if(i==value.len || value[i]==46){
            if(!name(value[start..i])){return false;}count+=1;start=i+1;
        }i+=1;
    }
    return count>=2;
}

fn bool equal_folded(ref u8[] value,ref u8[] expected){
    if(value.len!=expected.len){return false;}var usize i=0;
    while(i<value.len){if(folded(value[i])!=expected[i]){return false;}i+=1;}return true;
}

pub fn bool segment(ref u8[] value){
    if(value.len==0 || (!alpha(value[0]) && !digit(value[0]) && value[0]!=95) || value[value.len-1]==46){return false;}
    var usize base_end=value.len;var usize i=0;
    while(i<value.len){
        let u8 c=value[i];
        if(!alpha(c) && !digit(c) && c!=95 && c!=46 && c!=45){return false;}
        if(c==46 && base_end==value.len){base_end=i;}i+=1;
    }
    let ref u8[] base=value[..base_end];
    if(equal_folded(base,b"con") || equal_folded(base,b"prn") || equal_folded(base,b"aux") || equal_folded(base,b"nul")){return false;}
    if(base.len==4 && base[3]>=49 && base[3]<=57 && (equal_folded(base[..3],b"com") || equal_folded(base[..3],b"lpt"))){return false;}
    return true;
}

pub fn bool path(ref u8[] value){
    var usize start=0;var usize i=0;
    while(i<=value.len){
        if(i==value.len || value[i]==47){if(!segment(value[start..i])){return false;}start=i+1;}i+=1;
    }
    return true;
}

fn usize number_end(ref u8[] value,usize start){
    if(start==value.len || !digit(value[start])){return start;}
    var usize i=start;while(i<value.len && digit(value[i])){i+=1;}
    if(i-start>1 && value[start]==48){return start;}return i;
}

pub fn bool language(ref u8[] value){
    let ref u8[] prefix=b"fairyc-language/";
    if(value.len<=prefix.len){return false;}var usize i=0;
    while(i<prefix.len){if(value[i]!=prefix[i]){return false;}i+=1;}
    return number_end(value,prefix.len)==value.len;
}

fn bool identifiers(ref u8[] value,usize begin,usize end,bool prerelease){
    if(begin==end){return false;}
    var usize start=begin;var usize i=begin;var bool numeric=true;
    while(i<=end){
        if(i==end || value[i]==46){
            if(i==start || (prerelease && numeric && i-start>1 && value[start]==48)){return false;}
            start=i+1;numeric=true;
        }else{
            let u8 c=value[i];if(!alpha(c) && !digit(c) && c!=45){return false;}
            if(!digit(c)){numeric=false;}
        }i+=1;
    }
    return true;
}

pub fn bool version(ref u8[] value){
    var usize i=0;var usize component=0;
    while(component<3){
        let usize end=number_end(value,i);if(end==i){return false;}i=end;component+=1;
        if(component<3){if(i==value.len || value[i]!=46){return false;}i+=1;}
    }
    if(i==value.len){return true;}
    if(value[i]==45){
        let usize start=i+1;i=start;while(i<value.len && value[i]!=43){i+=1;}
        if(!identifiers(value,start,i,true)){return false;}
        if(i==value.len){return true;}
    }
    if(value[i]!=43){return false;}
    return identifiers(value,i+1,value.len,false);
}

// Go string ordering is unsigned byte lexicographic ordering. Callers first
// validate each path/name, then require strict ordering for uniqueness too.
pub fn bool before(ref u8[] left,ref u8[] right){
    var usize i=0;while(i<left.len && i<right.len){
        if(left[i]!=right[i]){return left[i]<right[i];}i+=1;
    }return left.len<right.len;
}
