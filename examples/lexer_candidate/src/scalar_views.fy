module scalar_views;

// Closed scalar slice identities. Byte descriptors keep their original IDs;
// the remaining pairs are disjoint from scalars, C scalars and nominal types.
pub fn bool extended(u32 type){return type>=18 && type<=29 && type!=22 && type!=23;}
pub fn bool valid(u32 type){return type==10 || type==11 || extended(type);}
pub fn u32 element(u32 type){assert(valid(type));return if(type==10 || type==11){3}else{(type-16)/2};}
pub fn bool mutable(u32 type){return valid(type) && type%2==1;}
pub fn u32 make(u32 element,bool exclusive){return if(element<1 || element>6){0}else if(element==3){if(exclusive){11}else{10}}else{16+element*2+if(exclusive){1}else{0}};}
pub fn bool shared_from(u32 target,u32 source){return valid(target) && !mutable(target) && mutable(source) && element(target)==element(source);}
