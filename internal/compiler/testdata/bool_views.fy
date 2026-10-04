module main;
fn usize count(ref bool[] flags) {
 var usize total=0;
 for(let bool flag in flags){if(flag){total+=1;}}
 for(let ref bool flag in flags){assert(*flag==flags[0] || *flag!=flags[0]);}
 return total;
}
fn ref bool first(ref bool[] flags){for(let ref bool flag in flags){return flag;}return ref flags[0];}
fn void flip(ref mut bool[] flags){for(let ref mut bool flag in flags){*flag=!*flag;}}
fn void early(ref mut bool[] flags){for(let ref mut bool flag in flags){*flag=true;return;}}
fn Result<i32,u32> fail(){return .err(7);}
fn Result<i32,u32> propagate(ref mut bool[] flags){for(let ref mut bool flag in flags){*flag=true;let i32 unused=fail()?;}return .ok(0);}
fn usize by_value(bool[2] data){var usize total=0;for(let bool flag in data){if(flag){total+=1;}}return total;}
fn i32 main(){
 var bool[5] flags=[false;5];
 {let ref mut bool cell=ref mut flags[1];*cell=true;}
 flags[3]=true;
 {let ref bool cell=ref flags[1];assert(*cell);}
 assert(count(ref flags)==2 && by_value([true,false])==1);
 let ref bool[] shared=ref flags;
 assert(!*first(shared) && shared.len==5 && !shared.is_empty);
 assert(count(shared[..])==2 && count(shared[1..4])==2);
 assert(count(shared[5..])==0 && shared[5..].is_empty);
 var usize seen=0;
 for(let bool flag in flags){if(flag){seen+=1;}}
 assert(seen==2);
 for(let ref bool flag in flags){assert(*flag==flags[seen-2]);break;}
 var usize visits=0;
 for(let ref mut bool flag in flags){visits+=1;*flag=!*flag;if(visits==1){continue;}if(visits==3){break;}}
 assert(visits==3 && flags[0] && !flags[1] && flags[2] && flags[3] && !flags[4]);
 let ref mut bool[] view=ref mut flags;
 {let ref mut bool[] child=view[1..4];flip(child);}
 assert(view[0] && view[1] && !view[2] && !view[3] && !view[4]);
 seen=0;
 for(let ref bool flag in view){if(*flag){seen+=1;}}
 assert(seen==2);
 visits=0;
 for(let bool flag in view){if(visits==0){view[1]=false;assert(flag);}if(visits==1){assert(!flag);}visits+=1;}
 assert(visits==5);
 flip(view[5..]);early(view[2..]);assert(view[2] && !view[3]);
 switch(propagate(view[3..])){case .ok(_):{assert(false);}case .err(code):{assert(code==7);}};
 assert(view[3] && !view[4]);
 {let ref mut bool cell=ref mut view[4];*cell=true;}
 assert(view[4]);
 var bool[1] other=[false];
 for(let ref mut bool flag in flags){for(let ref mut bool second in other){*second=*flag;}*flag=!*flag;}
 assert(!flags[0] && flags[1] && !flags[2] && !flags[3] && !flags[4] && other[0]);
 for(let ref mut bool flags in flags){*flags=!*flags;}
 assert(flags[0] && !flags[1] && flags[2] && flags[3] && flags[4]);
 return 0;
}
