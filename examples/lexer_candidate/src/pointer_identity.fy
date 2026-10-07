module pointer_identity;
// Nullable inert raw pointers have their own closed type namespace. They do
// not carry safe-reference origins, acquire loans, or imply dereference rights.
pub fn bool element(u32 type){return type>=1 && type<=5 || type==7;}
pub fn bool valid(u32 type){return type==1048601 || type>=1048578 && type<=1048591 && element((type-1048576)/2);}
pub fn u32 inner(u32 type){assert(valid(type));return if(type==1048601){1048591}else{(type-1048576)/2};}
pub fn bool mutable(u32 type){return valid(type) && type%2==1;}
pub fn u32 make(u32 type,bool exclusive){return if(type==1048591 && exclusive){1048601}else if(element(type)){1048576+type*2+if(exclusive){1}else{0}}else{0};}
pub fn bool borrowable(u32 type){return type==1048591;}
